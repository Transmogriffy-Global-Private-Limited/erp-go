package inventory

import (
	"context"
	"fmt"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
)

type CostLayer struct {
	ID           string    `json:"id"`
	ItemID       string    `json:"item_id"`
	ItemSKU      string    `json:"item_sku"`
	ItemName     string    `json:"item_name"`
	UnitCost     string    `json:"unit_cost"`
	CurrencyCode string    `json:"currency_code"`
	Source       string    `json:"source"`
	EffectiveAt  time.Time `json:"effective_at"`
	CreatedAt    time.Time `json:"created_at"`
}
type CreateCostLayerInput struct {
	ItemID       string    `json:"item_id"`
	UnitCost     string    `json:"unit_cost"`
	CurrencyCode string    `json:"currency_code"`
	Source       string    `json:"source"`
	EffectiveAt  time.Time `json:"effective_at"`
}
type ValuationLine struct {
	ItemID       string `json:"item_id"`
	ItemSKU      string `json:"item_sku"`
	ItemName     string `json:"item_name"`
	LocationID   string `json:"location_id"`
	LocationCode string `json:"location_code"`
	Quantity     string `json:"quantity"`
	UnitCost     string `json:"unit_cost"`
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}
type InventorySummary struct {
	ItemID         string `json:"item_id"`
	ItemSKU        string `json:"item_sku"`
	ItemName       string `json:"item_name"`
	OnHand         string `json:"on_hand"`
	Reserved       string `json:"reserved"`
	Available      string `json:"available"`
	UnitCost       string `json:"unit_cost"`
	CurrencyCode   string `json:"currency_code"`
	InventoryValue string `json:"inventory_value"`
}

func (s Store) ListCostLayers(ctx context.Context, tenantID string) ([]CostLayer, error) {
	v := []CostLayer{}
	e := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		r, e := tx.Query(ctx, `SELECT c.id::text,c.item_id::text,i.sku,i.name,c.unit_cost::text,c.currency_code,c.source,c.effective_at,c.created_at FROM inventory.cost_layers c JOIN inventory.items i ON i.tenant_id=c.tenant_id AND i.id=c.item_id ORDER BY c.effective_at DESC,c.created_at DESC LIMIT 200`)
		if e != nil {
			return e
		}
		defer r.Close()
		for r.Next() {
			var x CostLayer
			if e := r.Scan(&x.ID, &x.ItemID, &x.ItemSKU, &x.ItemName, &x.UnitCost, &x.CurrencyCode, &x.Source, &x.EffectiveAt, &x.CreatedAt); e != nil {
				return e
			}
			v = append(v, x)
		}
		return r.Err()
	})
	return v, e
}
func (s Store) CreateCostLayer(ctx context.Context, tenantID, actorID string, in CreateCostLayerInput) (CostLayer, error) {
	in.ItemID = strings.TrimSpace(in.ItemID)
	in.UnitCost = strings.TrimSpace(in.UnitCost)
	in.CurrencyCode = strings.ToUpper(strings.TrimSpace(in.CurrencyCode))
	in.Source = strings.TrimSpace(in.Source)
	if in.Source == "" {
		in.Source = "manual"
	}
	if in.EffectiveAt.IsZero() {
		in.EffectiveAt = time.Now().UTC()
	}
	var v CostLayer
	e := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, `INSERT INTO inventory.cost_layers(tenant_id,item_id,unit_cost,currency_code,source,effective_at,created_by) SELECT $1,i.id,$3::numeric,$4,$5,$6,$7::uuid FROM inventory.items i WHERE i.id=$2::uuid AND i.status='active' RETURNING id::text,item_id::text,unit_cost::text,currency_code,source,effective_at,created_at`, tenantID, in.ItemID, in.UnitCost, in.CurrencyCode, in.Source, in.EffectiveAt, actorID).Scan(&v.ID, &v.ItemID, &v.UnitCost, &v.CurrencyCode, &v.Source, &v.EffectiveAt, &v.CreatedAt); e != nil {
			return fmt.Errorf("insert inventory cost layer: %w", e)
		}
		if e := tx.QueryRow(ctx, `SELECT sku,name FROM inventory.items WHERE id=$1::uuid`, v.ItemID).Scan(&v.ItemSKU, &v.ItemName); e != nil {
			return e
		}
		if e := audit.Insert(ctx, tx, audit.Entry{TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID, Action: "inventory.cost_layer.create", TargetType: "inventory.cost_layer", TargetID: v.ID, Metadata: map[string]any{"item_id": v.ItemID, "unit_cost": v.UnitCost, "currency_code": v.CurrencyCode, "source": v.Source}}); e != nil {
			return e
		}
		return outbox.Insert(ctx, tx, outbox.Event{TenantID: tenantID, EventType: "inventory.cost.changed.v1", EventVersion: 1, AggregateType: "inventory.item", AggregateID: v.ItemID, Payload: map[string]any{"cost_layer_id": v.ID, "item_id": v.ItemID, "unit_cost": v.UnitCost, "currency_code": v.CurrencyCode, "source": v.Source}})
	})
	return v, e
}
func (s Store) ListValuation(ctx context.Context, tenantID string) ([]ValuationLine, error) {
	v := []ValuationLine{}
	e := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		r, e := tx.Query(ctx, `SELECT i.id::text,i.sku,i.name,l.id::text,l.code,b.quantity::numeric(18,3)::text,COALESCE(c.unit_cost,0)::numeric(18,4)::text,COALESCE(c.currency_code,''),(b.quantity*COALESCE(c.unit_cost,0))::numeric(18,4)::text FROM (SELECT item_id,location_id,SUM(quantity_delta) quantity FROM inventory.stock_movement_lines GROUP BY item_id,location_id) b JOIN inventory.items i ON i.id=b.item_id JOIN inventory.locations l ON l.id=b.location_id LEFT JOIN LATERAL(SELECT unit_cost,currency_code FROM inventory.cost_layers WHERE item_id=b.item_id AND effective_at<=now() ORDER BY effective_at DESC,created_at DESC LIMIT 1)c ON true ORDER BY i.sku,l.code`)
		if e != nil {
			return e
		}
		defer r.Close()
		for r.Next() {
			var x ValuationLine
			if e := r.Scan(&x.ItemID, &x.ItemSKU, &x.ItemName, &x.LocationID, &x.LocationCode, &x.Quantity, &x.UnitCost, &x.CurrencyCode, &x.Value); e != nil {
				return e
			}
			v = append(v, x)
		}
		return r.Err()
	})
	return v, e
}
func (s Store) ListInventorySummary(ctx context.Context, tenantID string) ([]InventorySummary, error) {
	v := []InventorySummary{}
	e := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		r, e := tx.Query(ctx, `SELECT i.id::text,i.sku,i.name,b.quantity::numeric(18,3)::text,COALESCE(r.quantity,0)::numeric(18,3)::text,GREATEST(b.quantity-COALESCE(r.quantity,0),0)::numeric(18,3)::text,COALESCE(c.unit_cost,0)::numeric(18,4)::text,COALESCE(c.currency_code,''),(b.quantity*COALESCE(c.unit_cost,0))::numeric(18,4)::text FROM (SELECT item_id,SUM(quantity_delta) quantity FROM inventory.stock_movement_lines GROUP BY item_id)b JOIN inventory.items i ON i.id=b.item_id LEFT JOIN(SELECT item_id,SUM(quantity) quantity FROM inventory.reservations WHERE status='active' AND(expires_at IS NULL OR expires_at>now()) GROUP BY item_id)r ON r.item_id=b.item_id LEFT JOIN LATERAL(SELECT unit_cost,currency_code FROM inventory.cost_layers WHERE item_id=b.item_id AND effective_at<=now() ORDER BY effective_at DESC,created_at DESC LIMIT 1)c ON true ORDER BY i.sku`)
		if e != nil {
			return e
		}
		defer r.Close()
		for r.Next() {
			var x InventorySummary
			if e := r.Scan(&x.ItemID, &x.ItemSKU, &x.ItemName, &x.OnHand, &x.Reserved, &x.Available, &x.UnitCost, &x.CurrencyCode, &x.InventoryValue); e != nil {
				return e
			}
			v = append(v, x)
		}
		return r.Err()
	})
	return v, e
}
