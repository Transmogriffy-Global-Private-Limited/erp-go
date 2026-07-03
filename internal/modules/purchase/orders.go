package purchase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPurchaseOrderSupplierNotFound = errors.New("purchase order supplier not found or inactive")
	ErrPurchaseOrderItemNotFound     = errors.New("purchase order item not found or inactive")
	ErrPurchaseOrderNotFound         = errors.New("purchase order not found")
	ErrPurchaseOrderNotDraft         = errors.New("purchase order is not draft")
)

type PurchaseOrder struct {
	ID                string              `json:"id"`
	OrderNumber       string              `json:"order_number"`
	Supplier          PurchaseOrderParty  `json:"supplier"`
	SupplierReference string              `json:"supplier_reference"`
	Notes             string              `json:"notes"`
	CurrencyCode      string              `json:"currency_code"`
	OrderedAt         time.Time           `json:"ordered_at"`
	ExpectedAt        *time.Time          `json:"expected_at"`
	Status            string              `json:"status"`
	ApprovedAt        *time.Time          `json:"approved_at"`
	Lines             []PurchaseOrderLine `json:"lines"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type PurchaseOrderParty struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type PurchaseOrderLine struct {
	ID        string `json:"id"`
	ItemID    string `json:"item_id"`
	ItemSKU   string `json:"item_sku"`
	ItemName  string `json:"item_name"`
	Quantity  string `json:"quantity"`
	UnitPrice string `json:"unit_price"`
	LineTotal string `json:"line_total"`
}

type CreatePurchaseOrderInput struct {
	SupplierID        string                    `json:"supplier_id"`
	SupplierReference string                    `json:"supplier_reference"`
	Notes             string                    `json:"notes"`
	CurrencyCode      string                    `json:"currency_code"`
	OrderedAt         time.Time                 `json:"ordered_at"`
	ExpectedAt        *time.Time                `json:"expected_at"`
	Lines             []CreatePurchaseOrderLine `json:"lines"`
}

type CreatePurchaseOrderLine struct {
	ItemID    string `json:"item_id"`
	Quantity  string `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}

func (s Store) ListPurchaseOrders(ctx context.Context, tenantID string) ([]PurchaseOrder, error) {
	orders := make([]PurchaseOrder, 0)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT
    po.id::text,
    po.order_number,
    s.id::text,
    s.code,
    s.name,
    po.supplier_reference,
    po.notes,
    po.currency_code,
    po.ordered_at,
    po.expected_at,
    po.status,
    po.approved_at,
    po.created_at,
    po.updated_at
FROM purchase.purchase_orders po
JOIN purchase.suppliers s
    ON s.tenant_id = po.tenant_id
   AND s.id = po.supplier_id
ORDER BY po.ordered_at DESC, po.created_at DESC, po.id DESC
LIMIT 100
`)
		if err != nil {
			return fmt.Errorf("query purchase orders: %w", err)
		}
		defer rows.Close()

		orderIndex := map[string]int{}
		for rows.Next() {
			var order PurchaseOrder
			if err := rows.Scan(
				&order.ID,
				&order.OrderNumber,
				&order.Supplier.ID,
				&order.Supplier.Code,
				&order.Supplier.Name,
				&order.SupplierReference,
				&order.Notes,
				&order.CurrencyCode,
				&order.OrderedAt,
				&order.ExpectedAt,
				&order.Status,
				&order.ApprovedAt,
				&order.CreatedAt,
				&order.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan purchase order: %w", err)
			}

			order.Lines = make([]PurchaseOrderLine, 0)
			orderIndex[order.ID] = len(orders)
			orders = append(orders, order)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate purchase orders: %w", err)
		}

		if len(orders) == 0 {
			return nil
		}

		lineRows, err := tx.Query(ctx, `
SELECT
    pol.id::text,
    pol.purchase_order_id::text,
    pol.item_id::text,
    i.sku,
    i.name,
    pol.quantity::text,
    pol.unit_price::text,
    (pol.quantity * pol.unit_price)::text
FROM purchase.purchase_order_lines pol
JOIN inventory.items i
    ON i.tenant_id = pol.tenant_id
   AND i.id = pol.item_id
ORDER BY pol.created_at ASC, pol.id ASC
`)
		if err != nil {
			return fmt.Errorf("query purchase order lines: %w", err)
		}
		defer lineRows.Close()

		for lineRows.Next() {
			var orderID string
			var line PurchaseOrderLine
			if err := lineRows.Scan(
				&line.ID,
				&orderID,
				&line.ItemID,
				&line.ItemSKU,
				&line.ItemName,
				&line.Quantity,
				&line.UnitPrice,
				&line.LineTotal,
			); err != nil {
				return fmt.Errorf("scan purchase order line: %w", err)
			}

			if index, ok := orderIndex[orderID]; ok {
				orders[index].Lines = append(orders[index].Lines, line)
			}
		}

		return lineRows.Err()
	})
	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s Store) CreatePurchaseOrder(ctx context.Context, tenantID string, actorID string, input CreatePurchaseOrderInput) (PurchaseOrder, error) {
	input.SupplierID = strings.TrimSpace(input.SupplierID)
	input.SupplierReference = strings.TrimSpace(input.SupplierReference)
	input.Notes = strings.TrimSpace(input.Notes)
	input.CurrencyCode = strings.ToUpper(strings.TrimSpace(input.CurrencyCode))
	if input.OrderedAt.IsZero() {
		input.OrderedAt = time.Now().UTC()
	}

	var order PurchaseOrder
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
SELECT id::text, code, name
FROM purchase.suppliers
WHERE id = $1::uuid
  AND status = 'active'
`, input.SupplierID).Scan(&order.Supplier.ID, &order.Supplier.Code, &order.Supplier.Name); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrPurchaseOrderSupplierNotFound
			}
			return fmt.Errorf("load purchase order supplier: %w", err)
		}

		if err := tx.QueryRow(ctx, `
INSERT INTO purchase.purchase_orders (
    tenant_id,
    order_number,
    supplier_id,
    supplier_reference,
    notes,
    currency_code,
    ordered_at,
    expected_at,
    status,
    created_by
)
VALUES (
    $1,
    'PO-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12)),
    $2::uuid,
    $3,
    $4,
    $5,
    $6,
    $7,
    'draft',
    $8::uuid
)
RETURNING id::text, order_number, supplier_reference, notes, currency_code,
          ordered_at, expected_at, status, approved_at, created_at, updated_at
`, tenantID, input.SupplierID, input.SupplierReference, input.Notes, input.CurrencyCode, input.OrderedAt, input.ExpectedAt, actorID).Scan(
			&order.ID,
			&order.OrderNumber,
			&order.SupplierReference,
			&order.Notes,
			&order.CurrencyCode,
			&order.OrderedAt,
			&order.ExpectedAt,
			&order.Status,
			&order.ApprovedAt,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert purchase order: %w", err)
		}

		order.Lines = make([]PurchaseOrderLine, 0, len(input.Lines))
		for _, inputLine := range input.Lines {
			inputLine.ItemID = strings.TrimSpace(inputLine.ItemID)
			inputLine.Quantity = strings.TrimSpace(inputLine.Quantity)
			inputLine.UnitPrice = strings.TrimSpace(inputLine.UnitPrice)

			var line PurchaseOrderLine
			if err := tx.QueryRow(ctx, `
SELECT id::text, sku, name
FROM inventory.items
WHERE id = $1::uuid
  AND status = 'active'
`, inputLine.ItemID).Scan(&line.ItemID, &line.ItemSKU, &line.ItemName); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrPurchaseOrderItemNotFound
				}
				return fmt.Errorf("load purchase order item: %w", err)
			}

			if err := tx.QueryRow(ctx, `
INSERT INTO purchase.purchase_order_lines (
    tenant_id,
    purchase_order_id,
    item_id,
    quantity,
    unit_price
)
VALUES ($1, $2::uuid, $3::uuid, $4::numeric, $5::numeric)
RETURNING id::text, quantity::text, unit_price::text, (quantity * unit_price)::text
`, tenantID, order.ID, line.ItemID, inputLine.Quantity, inputLine.UnitPrice).Scan(
				&line.ID,
				&line.Quantity,
				&line.UnitPrice,
				&line.LineTotal,
			); err != nil {
				return fmt.Errorf("insert purchase order line: %w", err)
			}

			order.Lines = append(order.Lines, line)
		}

		if err := insertPurchaseOrderEventRecords(ctx, tx, tenantID, actorID, order, "create"); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return PurchaseOrder{}, err
	}

	return order, nil
}

func (s Store) ApprovePurchaseOrder(ctx context.Context, tenantID string, actorID string, orderID string) (PurchaseOrder, error) {
	var order PurchaseOrder
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var status string
		if err := tx.QueryRow(ctx, `
SELECT status
FROM purchase.purchase_orders
WHERE id = $1::uuid
FOR UPDATE
`, orderID).Scan(&status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrPurchaseOrderNotFound
			}
			return fmt.Errorf("lock purchase order: %w", err)
		}

		if status != "draft" {
			return ErrPurchaseOrderNotDraft
		}

		if _, err := tx.Exec(ctx, `
UPDATE purchase.purchase_orders
SET status = 'approved',
    approved_by = $2::uuid,
    approved_at = now(),
    updated_at = now()
WHERE id = $1::uuid
`, orderID, actorID); err != nil {
			return fmt.Errorf("approve purchase order: %w", err)
		}

		loaded, err := loadPurchaseOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		order = loaded

		return insertPurchaseOrderEventRecords(ctx, tx, tenantID, actorID, order, "approve")
	})
	if err != nil {
		return PurchaseOrder{}, err
	}

	return order, nil
}

func loadPurchaseOrder(ctx context.Context, tx pgx.Tx, orderID string) (PurchaseOrder, error) {
	var order PurchaseOrder
	if err := tx.QueryRow(ctx, `
SELECT
    po.id::text, po.order_number, s.id::text, s.code, s.name,
    po.supplier_reference, po.notes, po.currency_code, po.ordered_at,
    po.expected_at, po.status, po.approved_at, po.created_at, po.updated_at
FROM purchase.purchase_orders po
JOIN purchase.suppliers s
    ON s.tenant_id = po.tenant_id
   AND s.id = po.supplier_id
WHERE po.id = $1::uuid
`, orderID).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.Supplier.ID,
		&order.Supplier.Code,
		&order.Supplier.Name,
		&order.SupplierReference,
		&order.Notes,
		&order.CurrencyCode,
		&order.OrderedAt,
		&order.ExpectedAt,
		&order.Status,
		&order.ApprovedAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	); err != nil {
		return PurchaseOrder{}, fmt.Errorf("load purchase order: %w", err)
	}

	order.Lines = make([]PurchaseOrderLine, 0)
	rows, err := tx.Query(ctx, `
SELECT pol.id::text, pol.item_id::text, i.sku, i.name,
       pol.quantity::text, pol.unit_price::text, (pol.quantity * pol.unit_price)::text
FROM purchase.purchase_order_lines pol
JOIN inventory.items i
    ON i.tenant_id = pol.tenant_id
   AND i.id = pol.item_id
WHERE pol.purchase_order_id = $1::uuid
ORDER BY pol.created_at ASC, pol.id ASC
`, orderID)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("query purchase order lines: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var line PurchaseOrderLine
		if err := rows.Scan(&line.ID, &line.ItemID, &line.ItemSKU, &line.ItemName, &line.Quantity, &line.UnitPrice, &line.LineTotal); err != nil {
			return PurchaseOrder{}, fmt.Errorf("scan purchase order line: %w", err)
		}
		order.Lines = append(order.Lines, line)
	}

	return order, rows.Err()
}

func insertPurchaseOrderEventRecords(ctx context.Context, tx pgx.Tx, tenantID string, actorID string, order PurchaseOrder, action string) error {
	eventType := "purchase.order.created.v1"
	if action == "approve" {
		eventType = "purchase.order.approved.v1"
	}

	if err := audit.Insert(ctx, tx, audit.Entry{
		TenantID:   tenantID,
		ActorType:  "tenant_user",
		ActorID:    actorID,
		Action:     "purchase.order." + action,
		TargetType: "purchase.order",
		TargetID:   order.ID,
		Metadata: map[string]any{
			"order_number":  order.OrderNumber,
			"supplier_id":   order.Supplier.ID,
			"currency_code": order.CurrencyCode,
			"line_count":    len(order.Lines),
			"status":        order.Status,
		},
	}); err != nil {
		return err
	}

	return outbox.Insert(ctx, tx, outbox.Event{
		TenantID:      tenantID,
		EventType:     eventType,
		EventVersion:  1,
		AggregateType: "purchase.order",
		AggregateID:   order.ID,
		Payload: map[string]any{
			"purchase_order_id": order.ID,
			"order_number":      order.OrderNumber,
			"supplier_id":       order.Supplier.ID,
			"currency_code":     order.CurrencyCode,
			"line_count":        len(order.Lines),
			"status":            order.Status,
		},
	})
}
