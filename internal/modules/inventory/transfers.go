package inventory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

var (
	ErrTransferReferenceNotFound = errors.New("transfer item or location not found or inactive")
	ErrTransferSameLocation      = errors.New("transfer locations must differ")
	ErrTransferInsufficientStock = errors.New("insufficient available stock for transfer")
	ErrTransferDuplicateItem     = errors.New("transfer contains duplicate item")
)

type Transfer struct {
	ID                  string         `json:"id"`
	TransferNumber      string         `json:"transfer_number"`
	Reference           string         `json:"reference"`
	Notes               string         `json:"notes"`
	TransferredAt       time.Time      `json:"transferred_at"`
	Status              string         `json:"status"`
	StockMovementID     string         `json:"stock_movement_id"`
	StockMovementNumber string         `json:"stock_movement_number"`
	Lines               []TransferLine `json:"lines"`
	CreatedAt           time.Time      `json:"created_at"`
}

type TransferLine struct {
	ID                      string `json:"id"`
	ItemID                  string `json:"item_id"`
	ItemSKU                 string `json:"item_sku"`
	ItemName                string `json:"item_name"`
	SourceLocationID        string `json:"source_location_id"`
	SourceLocationCode      string `json:"source_location_code"`
	DestinationLocationID   string `json:"destination_location_id"`
	DestinationLocationCode string `json:"destination_location_code"`
	Quantity                string `json:"quantity"`
}

type CreateTransferInput struct {
	Reference     string               `json:"reference"`
	Notes         string               `json:"notes"`
	TransferredAt time.Time            `json:"transferred_at"`
	Lines         []CreateTransferLine `json:"lines"`
}

type CreateTransferLine struct {
	ItemID                string `json:"item_id"`
	SourceLocationID      string `json:"source_location_id"`
	DestinationLocationID string `json:"destination_location_id"`
	Quantity              string `json:"quantity"`
}

func (s Store) ListTransfers(ctx context.Context, tenantID string) ([]Transfer, error) {
	transfers := make([]Transfer, 0)
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT t.id::text, t.transfer_number, t.reference, t.notes, t.transferred_at,
       t.status, t.stock_movement_id::text, sm.movement_number, t.created_at
FROM inventory.transfers t
JOIN inventory.stock_movements sm ON sm.tenant_id = t.tenant_id AND sm.id = t.stock_movement_id
ORDER BY t.transferred_at DESC, t.created_at DESC, t.id DESC LIMIT 100
`)
		if err != nil {
			return fmt.Errorf("query inventory transfers: %w", err)
		}
		defer rows.Close()
		index := map[string]int{}
		for rows.Next() {
			var transfer Transfer
			if err := rows.Scan(&transfer.ID, &transfer.TransferNumber, &transfer.Reference, &transfer.Notes,
				&transfer.TransferredAt, &transfer.Status, &transfer.StockMovementID,
				&transfer.StockMovementNumber, &transfer.CreatedAt); err != nil {
				return fmt.Errorf("scan inventory transfer: %w", err)
			}
			transfer.Lines = make([]TransferLine, 0)
			index[transfer.ID] = len(transfers)
			transfers = append(transfers, transfer)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		lineRows, err := tx.Query(ctx, `
SELECT tl.id::text, tl.transfer_id::text, tl.item_id::text, i.sku, i.name,
       tl.source_location_id::text, src.code,
       tl.destination_location_id::text, dst.code, tl.quantity::text
FROM inventory.transfer_lines tl
JOIN inventory.items i ON i.tenant_id = tl.tenant_id AND i.id = tl.item_id
JOIN inventory.locations src ON src.tenant_id = tl.tenant_id AND src.id = tl.source_location_id
JOIN inventory.locations dst ON dst.tenant_id = tl.tenant_id AND dst.id = tl.destination_location_id
ORDER BY tl.created_at, tl.id
`)
		if err != nil {
			return fmt.Errorf("query inventory transfer lines: %w", err)
		}
		defer lineRows.Close()
		for lineRows.Next() {
			var transferID string
			var line TransferLine
			if err := lineRows.Scan(&line.ID, &transferID, &line.ItemID, &line.ItemSKU, &line.ItemName,
				&line.SourceLocationID, &line.SourceLocationCode,
				&line.DestinationLocationID, &line.DestinationLocationCode, &line.Quantity); err != nil {
				return fmt.Errorf("scan inventory transfer line: %w", err)
			}
			if i, ok := index[transferID]; ok {
				transfers[i].Lines = append(transfers[i].Lines, line)
			}
		}
		return lineRows.Err()
	})
	return transfers, err
}

func (s Store) CreateTransfer(ctx context.Context, tenantID, actorID string, input CreateTransferInput) (Transfer, error) {
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.TransferredAt.IsZero() {
		input.TransferredAt = time.Now().UTC()
	}
	for i := range input.Lines {
		input.Lines[i].ItemID = strings.TrimSpace(input.Lines[i].ItemID)
		input.Lines[i].SourceLocationID = strings.TrimSpace(input.Lines[i].SourceLocationID)
		input.Lines[i].DestinationLocationID = strings.TrimSpace(input.Lines[i].DestinationLocationID)
		input.Lines[i].Quantity = strings.TrimSpace(input.Lines[i].Quantity)
	}
	sort.Slice(input.Lines, func(i, j int) bool { return input.Lines[i].ItemID < input.Lines[j].ItemID })
	for i, line := range input.Lines {
		if line.SourceLocationID == line.DestinationLocationID {
			return Transfer{}, ErrTransferSameLocation
		}
		if i > 0 && input.Lines[i-1].ItemID == line.ItemID {
			return Transfer{}, ErrTransferDuplicateItem
		}
	}

	var transfer Transfer
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT 'TR-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12))`).Scan(&transfer.TransferNumber); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
INSERT INTO inventory.stock_movements (tenant_id, movement_number, movement_type, reference, notes, occurred_at, status, created_by)
VALUES ($1, 'SM-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12)), 'transfer', $2, $3, $4, 'posted', $5::uuid)
RETURNING id::text, movement_number
`, tenantID, transfer.TransferNumber, input.Notes, input.TransferredAt, actorID).Scan(&transfer.StockMovementID, &transfer.StockMovementNumber); err != nil {
			return fmt.Errorf("insert transfer stock movement: %w", err)
		}
		if err := tx.QueryRow(ctx, `
INSERT INTO inventory.transfers (tenant_id, transfer_number, reference, notes, transferred_at, stock_movement_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6::uuid, $7::uuid)
RETURNING id::text, reference, notes, transferred_at, status, created_at
`, tenantID, transfer.TransferNumber, input.Reference, input.Notes, input.TransferredAt, transfer.StockMovementID, actorID).Scan(
			&transfer.ID, &transfer.Reference, &transfer.Notes, &transfer.TransferredAt, &transfer.Status, &transfer.CreatedAt); err != nil {
			return fmt.Errorf("insert inventory transfer: %w", err)
		}
		transfer.Lines = make([]TransferLine, 0, len(input.Lines))
		for _, in := range input.Lines {
			keys := []string{tenantID + ":" + in.ItemID + ":" + in.SourceLocationID, tenantID + ":" + in.ItemID + ":" + in.DestinationLocationID}
			sort.Strings(keys)
			for _, key := range keys {
				if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
					return err
				}
			}
			var line TransferLine
			if err := tx.QueryRow(ctx, `
SELECT i.id::text, i.sku, i.name, src.id::text, src.code, dst.id::text, dst.code
FROM inventory.items i
JOIN inventory.locations src ON src.tenant_id = i.tenant_id
JOIN inventory.locations dst ON dst.tenant_id = i.tenant_id
WHERE i.id = $1::uuid AND i.status = 'active'
  AND src.id = $2::uuid AND src.status = 'active'
  AND dst.id = $3::uuid AND dst.status = 'active'
`, in.ItemID, in.SourceLocationID, in.DestinationLocationID).Scan(
				&line.ItemID, &line.ItemSKU, &line.ItemName, &line.SourceLocationID,
				&line.SourceLocationCode, &line.DestinationLocationID, &line.DestinationLocationCode); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrTransferReferenceNotFound
				}
				return err
			}
			var available bool
			if err := tx.QueryRow(ctx, `
SELECT $3::numeric <= (
  COALESCE((SELECT SUM(quantity_delta) FROM inventory.stock_movement_lines WHERE item_id=$1::uuid AND location_id=$2::uuid), 0)
  - COALESCE((SELECT SUM(quantity) FROM inventory.reservations WHERE item_id=$1::uuid AND location_id=$2::uuid AND status='active' AND (expires_at IS NULL OR expires_at > now())), 0)
)
`, in.ItemID, in.SourceLocationID, in.Quantity).Scan(&available); err != nil {
				return err
			}
			if !available {
				return ErrTransferInsufficientStock
			}
			if err := tx.QueryRow(ctx, `
INSERT INTO inventory.transfer_lines (tenant_id, transfer_id, item_id, source_location_id, destination_location_id, quantity)
VALUES ($1,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::numeric)
RETURNING id::text, quantity::text
`, tenantID, transfer.ID, line.ItemID, line.SourceLocationID, line.DestinationLocationID, in.Quantity).Scan(&line.ID, &line.Quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
INSERT INTO inventory.stock_movement_lines (tenant_id,movement_id,item_id,location_id,quantity_delta)
VALUES ($1,$2::uuid,$3::uuid,$4::uuid,-($6::numeric)), ($1,$2::uuid,$3::uuid,$5::uuid,$6::numeric)
`, tenantID, transfer.StockMovementID, line.ItemID, line.SourceLocationID, line.DestinationLocationID, line.Quantity); err != nil {
				return err
			}
			transfer.Lines = append(transfer.Lines, line)
		}
		if err := audit.Insert(ctx, tx, audit.Entry{TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID, Action: "inventory.transfer.post", TargetType: "inventory.transfer", TargetID: transfer.ID, Metadata: map[string]any{"transfer_number": transfer.TransferNumber, "stock_movement_id": transfer.StockMovementID, "line_count": len(transfer.Lines)}}); err != nil {
			return err
		}
		if err := outbox.Insert(ctx, tx, outbox.Event{TenantID: tenantID, EventType: "inventory.stock_movement.created.v1", EventVersion: 1, AggregateType: "inventory.stock_movement", AggregateID: transfer.StockMovementID, Payload: map[string]any{"movement_id": transfer.StockMovementID, "movement_number": transfer.StockMovementNumber, "movement_type": "transfer", "reference": transfer.TransferNumber, "line_count": len(transfer.Lines) * 2}}); err != nil {
			return err
		}
		return outbox.Insert(ctx, tx, outbox.Event{TenantID: tenantID, EventType: "inventory.transfer.posted.v1", EventVersion: 1, AggregateType: "inventory.transfer", AggregateID: transfer.ID, Payload: map[string]any{"transfer_id": transfer.ID, "transfer_number": transfer.TransferNumber, "stock_movement_id": transfer.StockMovementID, "line_count": len(transfer.Lines)}})
	})
	return transfer, err
}
