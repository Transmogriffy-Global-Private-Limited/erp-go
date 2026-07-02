package inventory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

type StockMovement struct {
	ID             string              `json:"id"`
	MovementNumber string              `json:"movement_number"`
	MovementType   string              `json:"movement_type"`
	Reference      string              `json:"reference"`
	Notes          string              `json:"notes"`
	OccurredAt     time.Time           `json:"occurred_at"`
	Status         string              `json:"status"`
	Lines          []StockMovementLine `json:"lines"`
	CreatedAt      time.Time           `json:"created_at"`
}

type StockMovementLine struct {
	ID            string `json:"id"`
	ItemID        string `json:"item_id"`
	ItemSKU       string `json:"item_sku"`
	ItemName      string `json:"item_name"`
	LocationID    string `json:"location_id"`
	LocationCode  string `json:"location_code"`
	LocationName  string `json:"location_name"`
	QuantityDelta string `json:"quantity_delta"`
}

type CreateStockMovementInput struct {
	MovementType string                    `json:"movement_type"`
	Reference    string                    `json:"reference"`
	Notes        string                    `json:"notes"`
	OccurredAt   time.Time                 `json:"occurred_at"`
	Lines        []CreateStockMovementLine `json:"lines"`
}

type CreateStockMovementLine struct {
	ItemID        string `json:"item_id"`
	LocationID    string `json:"location_id"`
	QuantityDelta string `json:"quantity_delta"`
}

type StockBalance struct {
	ItemID       string `json:"item_id"`
	ItemSKU      string `json:"item_sku"`
	ItemName     string `json:"item_name"`
	BaseUnitID   string `json:"base_unit_id"`
	BaseUnitCode string `json:"base_unit_code"`
	LocationID   string `json:"location_id"`
	LocationCode string `json:"location_code"`
	LocationName string `json:"location_name"`
	Quantity     string `json:"quantity"`
}

func (s Store) ListStockMovements(ctx context.Context, tenantID string) ([]StockMovement, error) {
	movements := make([]StockMovement, 0)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT
    id::text,
    movement_number,
    movement_type,
    reference,
    notes,
    occurred_at,
    status,
    created_at
FROM inventory.stock_movements
ORDER BY occurred_at DESC, created_at DESC, id DESC
LIMIT 100
`)
		if err != nil {
			return fmt.Errorf("query inventory stock movements: %w", err)
		}
		defer rows.Close()

		movementIndex := map[string]int{}

		for rows.Next() {
			var movement StockMovement
			if err := rows.Scan(
				&movement.ID,
				&movement.MovementNumber,
				&movement.MovementType,
				&movement.Reference,
				&movement.Notes,
				&movement.OccurredAt,
				&movement.Status,
				&movement.CreatedAt,
			); err != nil {
				return fmt.Errorf("scan inventory stock movement: %w", err)
			}

			movement.Lines = make([]StockMovementLine, 0)
			movementIndex[movement.ID] = len(movements)
			movements = append(movements, movement)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate inventory stock movements: %w", err)
		}

		if len(movements) == 0 {
			return nil
		}

		lineRows, err := tx.Query(ctx, `
SELECT
    l.id::text,
    l.movement_id::text,
    l.item_id::text,
    i.sku,
    i.name,
    l.location_id::text,
    loc.code,
    loc.name,
    l.quantity_delta::text
FROM inventory.stock_movement_lines l
JOIN inventory.items i
    ON i.tenant_id = l.tenant_id
   AND i.id = l.item_id
JOIN inventory.locations loc
    ON loc.tenant_id = l.tenant_id
   AND loc.id = l.location_id
ORDER BY l.created_at ASC, l.id ASC
`)
		if err != nil {
			return fmt.Errorf("query inventory stock movement lines: %w", err)
		}
		defer lineRows.Close()

		for lineRows.Next() {
			var movementID string
			var line StockMovementLine
			if err := lineRows.Scan(
				&line.ID,
				&movementID,
				&line.ItemID,
				&line.ItemSKU,
				&line.ItemName,
				&line.LocationID,
				&line.LocationCode,
				&line.LocationName,
				&line.QuantityDelta,
			); err != nil {
				return fmt.Errorf("scan inventory stock movement line: %w", err)
			}

			index, ok := movementIndex[movementID]
			if !ok {
				continue
			}

			movements[index].Lines = append(movements[index].Lines, line)
		}

		if err := lineRows.Err(); err != nil {
			return fmt.Errorf("iterate inventory stock movement lines: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (s Store) CreateStockMovement(ctx context.Context, tenantID string, actorID string, input CreateStockMovementInput) (StockMovement, error) {
	input.MovementType = strings.TrimSpace(input.MovementType)
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)

	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}

	var movement StockMovement

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
INSERT INTO inventory.stock_movements (
    tenant_id,
    movement_number,
    movement_type,
    reference,
    notes,
    occurred_at,
    status,
    created_by
)
VALUES (
    $1,
    'SM-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12)),
    $2,
    $3,
    $4,
    $5,
    'posted',
    $6::uuid
)
RETURNING id::text, movement_number, movement_type, reference, notes, occurred_at, status, created_at
`, tenantID, input.MovementType, input.Reference, input.Notes, input.OccurredAt, actorID).Scan(
			&movement.ID,
			&movement.MovementNumber,
			&movement.MovementType,
			&movement.Reference,
			&movement.Notes,
			&movement.OccurredAt,
			&movement.Status,
			&movement.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert inventory stock movement: %w", err)
		}

		movement.Lines = make([]StockMovementLine, 0, len(input.Lines))

		for _, inputLine := range input.Lines {
			inputLine.ItemID = strings.TrimSpace(inputLine.ItemID)
			inputLine.LocationID = strings.TrimSpace(inputLine.LocationID)
			inputLine.QuantityDelta = strings.TrimSpace(inputLine.QuantityDelta)

			var line StockMovementLine
			err := tx.QueryRow(ctx, `
INSERT INTO inventory.stock_movement_lines (
    tenant_id,
    movement_id,
    item_id,
    location_id,
    quantity_delta
)
VALUES (
    $1,
    $2::uuid,
    $3::uuid,
    $4::uuid,
    $5::numeric
)
RETURNING id::text, item_id::text, location_id::text, quantity_delta::text
`, tenantID, movement.ID, inputLine.ItemID, inputLine.LocationID, inputLine.QuantityDelta).Scan(
				&line.ID,
				&line.ItemID,
				&line.LocationID,
				&line.QuantityDelta,
			)
			if err != nil {
				return fmt.Errorf("insert inventory stock movement line: %w", err)
			}

			if err := tx.QueryRow(ctx, `
SELECT i.sku, i.name, loc.code, loc.name
FROM inventory.items i
JOIN inventory.locations loc
    ON loc.tenant_id = i.tenant_id
WHERE i.tenant_id = $1::uuid
  AND i.id = $2::uuid
  AND loc.id = $3::uuid
`, tenantID, line.ItemID, line.LocationID).Scan(
				&line.ItemSKU,
				&line.ItemName,
				&line.LocationCode,
				&line.LocationName,
			); err != nil {
				return fmt.Errorf("load inventory stock movement line refs: %w", err)
			}

			movement.Lines = append(movement.Lines, line)
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID:   tenantID,
			ActorType:  "tenant_user",
			ActorID:    actorID,
			Action:     "inventory.stock_movement.create",
			TargetType: "inventory.stock_movement",
			TargetID:   movement.ID,
			Metadata: map[string]any{
				"movement_number": movement.MovementNumber,
				"movement_type":   movement.MovementType,
				"reference":       movement.Reference,
				"line_count":      len(movement.Lines),
			},
		}); err != nil {
			return err
		}

		if err := outbox.Insert(ctx, tx, outbox.Event{
			TenantID:      tenantID,
			EventType:     "inventory.stock_movement.created.v1",
			EventVersion:  1,
			AggregateType: "inventory.stock_movement",
			AggregateID:   movement.ID,
			Payload: map[string]any{
				"movement_id":     movement.ID,
				"movement_number": movement.MovementNumber,
				"movement_type":   movement.MovementType,
				"reference":       movement.Reference,
				"line_count":      len(movement.Lines),
			},
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return StockMovement{}, err
	}

	return movement, nil
}

func (s Store) ListStockBalances(ctx context.Context, tenantID string) ([]StockBalance, error) {
	balances := make([]StockBalance, 0)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT
    i.id::text,
    i.sku,
    i.name,
    COALESCE(i.base_unit_id::text, ''),
    COALESCE(u.code, ''),
    loc.id::text,
    loc.code,
    loc.name,
    SUM(l.quantity_delta)::text
FROM inventory.stock_movement_lines l
JOIN inventory.items i
    ON i.tenant_id = l.tenant_id
   AND i.id = l.item_id
LEFT JOIN inventory.units u
    ON u.tenant_id = i.tenant_id
   AND u.id = i.base_unit_id
JOIN inventory.locations loc
    ON loc.tenant_id = l.tenant_id
   AND loc.id = l.location_id
GROUP BY i.id, i.sku, i.name, i.base_unit_id, u.code, loc.id, loc.code, loc.name
ORDER BY i.sku ASC, loc.code ASC
`)
		if err != nil {
			return fmt.Errorf("query inventory stock balances: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var balance StockBalance
			if err := rows.Scan(
				&balance.ItemID,
				&balance.ItemSKU,
				&balance.ItemName,
				&balance.BaseUnitID,
				&balance.BaseUnitCode,
				&balance.LocationID,
				&balance.LocationCode,
				&balance.LocationName,
				&balance.Quantity,
			); err != nil {
				return fmt.Errorf("scan inventory stock balance: %w", err)
			}

			balances = append(balances, balance)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate inventory stock balances: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return balances, nil
}
