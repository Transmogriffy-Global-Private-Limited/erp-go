package inventory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

var (
	ErrStockCountReferenceNotFound = errors.New("stock count item or location not found or inactive")
	ErrStockCountDuplicateItem     = errors.New("stock count contains duplicate item")
)

type StockCount struct {
	ID                  string           `json:"id"`
	CountNumber         string           `json:"count_number"`
	LocationID          string           `json:"location_id"`
	LocationCode        string           `json:"location_code"`
	Reference           string           `json:"reference"`
	Notes               string           `json:"notes"`
	CountedAt           time.Time        `json:"counted_at"`
	Status              string           `json:"status"`
	StockMovementID     string           `json:"stock_movement_id"`
	StockMovementNumber string           `json:"stock_movement_number"`
	Lines               []StockCountLine `json:"lines"`
	CreatedAt           time.Time        `json:"created_at"`
}
type StockCountLine struct {
	ID               string `json:"id"`
	ItemID           string `json:"item_id"`
	ItemSKU          string `json:"item_sku"`
	ItemName         string `json:"item_name"`
	SystemQuantity   string `json:"system_quantity"`
	CountedQuantity  string `json:"counted_quantity"`
	VarianceQuantity string `json:"variance_quantity"`
}
type CreateStockCountInput struct {
	LocationID string                 `json:"location_id"`
	Reference  string                 `json:"reference"`
	Notes      string                 `json:"notes"`
	CountedAt  time.Time              `json:"counted_at"`
	Lines      []CreateStockCountLine `json:"lines"`
}
type CreateStockCountLine struct {
	ItemID          string `json:"item_id"`
	CountedQuantity string `json:"counted_quantity"`
}

func (s Store) ListStockCounts(ctx context.Context, tenantID string) ([]StockCount, error) {
	counts := make([]StockCount, 0)
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT sc.id::text,sc.count_number,sc.location_id::text,l.code,sc.reference,sc.notes,sc.counted_at,sc.status,sc.stock_movement_id::text,sm.movement_number,sc.created_at FROM inventory.stock_counts sc JOIN inventory.locations l ON l.tenant_id=sc.tenant_id AND l.id=sc.location_id JOIN inventory.stock_movements sm ON sm.tenant_id=sc.tenant_id AND sm.id=sc.stock_movement_id ORDER BY sc.counted_at DESC,sc.id DESC LIMIT 100`)
		if err != nil {
			return err
		}
		defer rows.Close()
		index := map[string]int{}
		for rows.Next() {
			var c StockCount
			if err := rows.Scan(&c.ID, &c.CountNumber, &c.LocationID, &c.LocationCode, &c.Reference, &c.Notes, &c.CountedAt, &c.Status, &c.StockMovementID, &c.StockMovementNumber, &c.CreatedAt); err != nil {
				return err
			}
			c.Lines = []StockCountLine{}
			index[c.ID] = len(counts)
			counts = append(counts, c)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		lr, err := tx.Query(ctx, `SELECT scl.id::text,scl.stock_count_id::text,scl.item_id::text,i.sku,i.name,scl.system_quantity::text,scl.counted_quantity::text,scl.variance_quantity::text FROM inventory.stock_count_lines scl JOIN inventory.items i ON i.tenant_id=scl.tenant_id AND i.id=scl.item_id ORDER BY scl.created_at,scl.id`)
		if err != nil {
			return err
		}
		defer lr.Close()
		for lr.Next() {
			var cid string
			var l StockCountLine
			if err := lr.Scan(&l.ID, &cid, &l.ItemID, &l.ItemSKU, &l.ItemName, &l.SystemQuantity, &l.CountedQuantity, &l.VarianceQuantity); err != nil {
				return err
			}
			if i, ok := index[cid]; ok {
				counts[i].Lines = append(counts[i].Lines, l)
			}
		}
		return lr.Err()
	})
	return counts, err
}

func (s Store) CreateStockCount(ctx context.Context, tenantID, actorID string, input CreateStockCountInput) (StockCount, error) {
	input.LocationID = strings.TrimSpace(input.LocationID)
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.CountedAt.IsZero() {
		input.CountedAt = time.Now().UTC()
	}
	for i := range input.Lines {
		input.Lines[i].ItemID = strings.TrimSpace(input.Lines[i].ItemID)
		input.Lines[i].CountedQuantity = strings.TrimSpace(input.Lines[i].CountedQuantity)
	}
	sort.Slice(input.Lines, func(i, j int) bool { return input.Lines[i].ItemID < input.Lines[j].ItemID })
	for i := 1; i < len(input.Lines); i++ {
		if input.Lines[i-1].ItemID == input.Lines[i].ItemID {
			return StockCount{}, ErrStockCountDuplicateItem
		}
	}
	var count StockCount
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT id::text,code FROM inventory.locations WHERE id=$1::uuid AND status='active'`, input.LocationID).Scan(&count.LocationID, &count.LocationCode); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrStockCountReferenceNotFound
			}
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT 'SC-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,12))`).Scan(&count.CountNumber); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO inventory.stock_movements(tenant_id,movement_number,movement_type,reference,notes,occurred_at,status,created_by) VALUES($1,'SM-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,12)),'adjustment',$2,$3,$4,'posted',$5::uuid) RETURNING id::text,movement_number`, tenantID, count.CountNumber, input.Notes, input.CountedAt, actorID).Scan(&count.StockMovementID, &count.StockMovementNumber); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO inventory.stock_counts(tenant_id,count_number,location_id,reference,notes,counted_at,stock_movement_id,created_by) VALUES($1,$2,$3::uuid,$4,$5,$6,$7::uuid,$8::uuid) RETURNING id::text,reference,notes,counted_at,status,created_at`, tenantID, count.CountNumber, count.LocationID, input.Reference, input.Notes, input.CountedAt, count.StockMovementID, actorID).Scan(&count.ID, &count.Reference, &count.Notes, &count.CountedAt, &count.Status, &count.CreatedAt); err != nil {
			return err
		}
		count.Lines = []StockCountLine{}
		for _, in := range input.Lines {
			key := tenantID + ":" + in.ItemID + ":" + count.LocationID
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); err != nil {
				return err
			}
			var line StockCountLine
			if err := tx.QueryRow(ctx, `SELECT id::text,sku,name FROM inventory.items WHERE id=$1::uuid AND status='active'`, in.ItemID).Scan(&line.ItemID, &line.ItemSKU, &line.ItemName); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrStockCountReferenceNotFound
				}
				return err
			}
			if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(quantity_delta),0)::numeric(18,3)::text,$3::numeric(18,3)::text,($3::numeric-COALESCE(SUM(quantity_delta),0))::numeric(18,3)::text FROM inventory.stock_movement_lines WHERE item_id=$1::uuid AND location_id=$2::uuid`, line.ItemID, count.LocationID, in.CountedQuantity).Scan(&line.SystemQuantity, &line.CountedQuantity, &line.VarianceQuantity); err != nil {
				return err
			}
			if err := tx.QueryRow(ctx, `INSERT INTO inventory.stock_count_lines(tenant_id,stock_count_id,item_id,system_quantity,counted_quantity,variance_quantity) VALUES($1,$2::uuid,$3::uuid,$4::numeric,$5::numeric,$6::numeric) RETURNING id::text`, tenantID, count.ID, line.ItemID, line.SystemQuantity, line.CountedQuantity, line.VarianceQuantity).Scan(&line.ID); err != nil {
				return err
			}
			if line.VarianceQuantity != "0.000" {
				if _, err := tx.Exec(ctx, `INSERT INTO inventory.stock_movement_lines(tenant_id,movement_id,item_id,location_id,quantity_delta) VALUES($1,$2::uuid,$3::uuid,$4::uuid,$5::numeric)`, tenantID, count.StockMovementID, line.ItemID, count.LocationID, line.VarianceQuantity); err != nil {
					return err
				}
			}
			count.Lines = append(count.Lines, line)
		}
		if err := audit.Insert(ctx, tx, audit.Entry{TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID, Action: "inventory.stock_count.post", TargetType: "inventory.stock_count", TargetID: count.ID, Metadata: map[string]any{"count_number": count.CountNumber, "location_id": count.LocationID, "stock_movement_id": count.StockMovementID, "line_count": len(count.Lines)}}); err != nil {
			return err
		}
		if err := outbox.Insert(ctx, tx, outbox.Event{TenantID: tenantID, EventType: "inventory.stock_movement.created.v1", EventVersion: 1, AggregateType: "inventory.stock_movement", AggregateID: count.StockMovementID, Payload: map[string]any{"movement_id": count.StockMovementID, "movement_number": count.StockMovementNumber, "movement_type": "adjustment", "reference": count.CountNumber, "line_count": len(count.Lines)}}); err != nil {
			return err
		}
		return outbox.Insert(ctx, tx, outbox.Event{TenantID: tenantID, EventType: "inventory.stock_count.posted.v1", EventVersion: 1, AggregateType: "inventory.stock_count", AggregateID: count.ID, Payload: map[string]any{"count_id": count.ID, "count_number": count.CountNumber, "location_id": count.LocationID, "stock_movement_id": count.StockMovementID, "line_count": len(count.Lines)}})
	})
	return count, err
}
