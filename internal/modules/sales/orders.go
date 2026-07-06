package sales

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
	ErrSalesOrderCustomerNotFound = errors.New("sales order customer not found or inactive")
	ErrSalesOrderItemNotFound     = errors.New("sales order item not found or inactive")
	ErrSalesOrderNotFound         = errors.New("sales order not found")
	ErrSalesOrderNotDraft         = errors.New("sales order is not draft")
)

type SalesOrder struct {
	ID                  string           `json:"id"`
	OrderNumber         string           `json:"order_number"`
	Customer            SalesOrderParty  `json:"customer"`
	CustomerReference   string           `json:"customer_reference"`
	Notes               string           `json:"notes"`
	CurrencyCode        string           `json:"currency_code"`
	OrderedAt           time.Time        `json:"ordered_at"`
	RequestedDeliveryAt *time.Time       `json:"requested_delivery_at"`
	Status              string           `json:"status"`
	ConfirmedAt         *time.Time       `json:"confirmed_at"`
	Lines               []SalesOrderLine `json:"lines"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

type SalesOrderParty struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type SalesOrderLine struct {
	ID        string `json:"id"`
	ItemID    string `json:"item_id"`
	ItemSKU   string `json:"item_sku"`
	ItemName  string `json:"item_name"`
	Quantity  string `json:"quantity"`
	UnitPrice string `json:"unit_price"`
	LineTotal string `json:"line_total"`
}

type CreateSalesOrderInput struct {
	CustomerID          string                 `json:"customer_id"`
	CustomerReference   string                 `json:"customer_reference"`
	Notes               string                 `json:"notes"`
	CurrencyCode        string                 `json:"currency_code"`
	OrderedAt           time.Time              `json:"ordered_at"`
	RequestedDeliveryAt *time.Time             `json:"requested_delivery_at"`
	Lines               []CreateSalesOrderLine `json:"lines"`
}

type CreateSalesOrderLine struct {
	ItemID    string `json:"item_id"`
	Quantity  string `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}

func (s Store) ListOrders(ctx context.Context, tenantID string) ([]SalesOrder, error) {
	orders := make([]SalesOrder, 0)
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT so.id::text, so.order_number, c.id::text, c.code, c.name,
       so.customer_reference, so.notes, so.currency_code, so.ordered_at,
       so.requested_delivery_at, so.status, so.confirmed_at, so.created_at, so.updated_at
FROM sales.orders so
JOIN sales.customers c ON c.tenant_id = so.tenant_id AND c.id = so.customer_id
ORDER BY so.ordered_at DESC, so.created_at DESC, so.id DESC
LIMIT 100
`)
		if err != nil {
			return fmt.Errorf("query sales orders: %w", err)
		}
		defer rows.Close()

		orderIndex := map[string]int{}
		for rows.Next() {
			var order SalesOrder
			if err := rows.Scan(
				&order.ID, &order.OrderNumber,
				&order.Customer.ID, &order.Customer.Code, &order.Customer.Name,
				&order.CustomerReference, &order.Notes, &order.CurrencyCode,
				&order.OrderedAt, &order.RequestedDeliveryAt, &order.Status,
				&order.ConfirmedAt, &order.CreatedAt, &order.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan sales order: %w", err)
			}
			order.Lines = make([]SalesOrderLine, 0)
			orderIndex[order.ID] = len(orders)
			orders = append(orders, order)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate sales orders: %w", err)
		}
		if len(orders) == 0 {
			return nil
		}

		lineRows, err := tx.Query(ctx, `
SELECT sol.id::text, sol.sales_order_id::text, sol.item_id::text, i.sku, i.name,
       sol.quantity::text, sol.unit_price::text, (sol.quantity * sol.unit_price)::text
FROM sales.order_lines sol
JOIN inventory.items i ON i.tenant_id = sol.tenant_id AND i.id = sol.item_id
ORDER BY sol.created_at ASC, sol.id ASC
`)
		if err != nil {
			return fmt.Errorf("query sales order lines: %w", err)
		}
		defer lineRows.Close()
		for lineRows.Next() {
			var orderID string
			var line SalesOrderLine
			if err := lineRows.Scan(
				&line.ID, &orderID, &line.ItemID, &line.ItemSKU, &line.ItemName,
				&line.Quantity, &line.UnitPrice, &line.LineTotal,
			); err != nil {
				return fmt.Errorf("scan sales order line: %w", err)
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

func (s Store) CreateOrder(ctx context.Context, tenantID string, actorID string, input CreateSalesOrderInput) (SalesOrder, error) {
	input.CustomerID = strings.TrimSpace(input.CustomerID)
	input.CustomerReference = strings.TrimSpace(input.CustomerReference)
	input.Notes = strings.TrimSpace(input.Notes)
	input.CurrencyCode = strings.ToUpper(strings.TrimSpace(input.CurrencyCode))
	if input.OrderedAt.IsZero() {
		input.OrderedAt = time.Now().UTC()
	}

	var order SalesOrder
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
SELECT id::text, code, name
FROM sales.customers
WHERE id = $1::uuid AND status = 'active'
`, input.CustomerID).Scan(&order.Customer.ID, &order.Customer.Code, &order.Customer.Name); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSalesOrderCustomerNotFound
			}
			return fmt.Errorf("load sales order customer: %w", err)
		}

		if err := tx.QueryRow(ctx, `
INSERT INTO sales.orders (
    tenant_id, order_number, customer_id, customer_reference, notes,
    currency_code, ordered_at, requested_delivery_at, status, created_by
)
VALUES (
    $1, 'SO-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12)),
    $2::uuid, $3, $4, $5, $6, $7, 'draft', $8::uuid
)
RETURNING id::text, order_number, customer_reference, notes, currency_code,
          ordered_at, requested_delivery_at, status, confirmed_at, created_at, updated_at
`, tenantID, input.CustomerID, input.CustomerReference, input.Notes, input.CurrencyCode, input.OrderedAt, input.RequestedDeliveryAt, actorID).Scan(
			&order.ID, &order.OrderNumber, &order.CustomerReference, &order.Notes,
			&order.CurrencyCode, &order.OrderedAt, &order.RequestedDeliveryAt,
			&order.Status, &order.ConfirmedAt, &order.CreatedAt, &order.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert sales order: %w", err)
		}

		order.Lines = make([]SalesOrderLine, 0, len(input.Lines))
		for _, inputLine := range input.Lines {
			var line SalesOrderLine
			if err := tx.QueryRow(ctx, `
SELECT id::text, sku, name
FROM inventory.items
WHERE id = $1::uuid AND status = 'active'
`, inputLine.ItemID).Scan(&line.ItemID, &line.ItemSKU, &line.ItemName); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrSalesOrderItemNotFound
				}
				return fmt.Errorf("load sales order item: %w", err)
			}

			if err := tx.QueryRow(ctx, `
INSERT INTO sales.order_lines (tenant_id, sales_order_id, item_id, quantity, unit_price)
VALUES ($1, $2::uuid, $3::uuid, $4::numeric, $5::numeric)
RETURNING id::text, quantity::text, unit_price::text, (quantity * unit_price)::text
`, tenantID, order.ID, line.ItemID, inputLine.Quantity, inputLine.UnitPrice).Scan(
				&line.ID, &line.Quantity, &line.UnitPrice, &line.LineTotal,
			); err != nil {
				return fmt.Errorf("insert sales order line: %w", err)
			}
			order.Lines = append(order.Lines, line)
		}

		return insertSalesOrderEventRecords(ctx, tx, tenantID, actorID, order, "create")
	})
	if err != nil {
		return SalesOrder{}, err
	}
	return order, nil
}

func (s Store) ConfirmOrder(ctx context.Context, tenantID string, actorID string, orderID string) (SalesOrder, error) {
	var order SalesOrder
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM sales.orders WHERE id = $1::uuid FOR UPDATE`, orderID).Scan(&status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSalesOrderNotFound
			}
			return fmt.Errorf("lock sales order: %w", err)
		}
		if status != "draft" {
			return ErrSalesOrderNotDraft
		}
		if _, err := tx.Exec(ctx, `
UPDATE sales.orders
SET status = 'confirmed', confirmed_by = $2::uuid, confirmed_at = now(), updated_at = now()
WHERE id = $1::uuid
`, orderID, actorID); err != nil {
			return fmt.Errorf("confirm sales order: %w", err)
		}

		loaded, err := loadSalesOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		order = loaded
		return insertSalesOrderEventRecords(ctx, tx, tenantID, actorID, order, "confirm")
	})
	if err != nil {
		return SalesOrder{}, err
	}
	return order, nil
}

func loadSalesOrder(ctx context.Context, tx pgx.Tx, orderID string) (SalesOrder, error) {
	var order SalesOrder
	if err := tx.QueryRow(ctx, `
SELECT so.id::text, so.order_number, c.id::text, c.code, c.name,
       so.customer_reference, so.notes, so.currency_code, so.ordered_at,
       so.requested_delivery_at, so.status, so.confirmed_at, so.created_at, so.updated_at
FROM sales.orders so
JOIN sales.customers c ON c.tenant_id = so.tenant_id AND c.id = so.customer_id
WHERE so.id = $1::uuid
`, orderID).Scan(
		&order.ID, &order.OrderNumber,
		&order.Customer.ID, &order.Customer.Code, &order.Customer.Name,
		&order.CustomerReference, &order.Notes, &order.CurrencyCode,
		&order.OrderedAt, &order.RequestedDeliveryAt, &order.Status,
		&order.ConfirmedAt, &order.CreatedAt, &order.UpdatedAt,
	); err != nil {
		return SalesOrder{}, fmt.Errorf("load sales order: %w", err)
	}

	order.Lines = make([]SalesOrderLine, 0)
	rows, err := tx.Query(ctx, `
SELECT sol.id::text, sol.item_id::text, i.sku, i.name,
       sol.quantity::text, sol.unit_price::text, (sol.quantity * sol.unit_price)::text
FROM sales.order_lines sol
JOIN inventory.items i ON i.tenant_id = sol.tenant_id AND i.id = sol.item_id
WHERE sol.sales_order_id = $1::uuid
ORDER BY sol.created_at ASC, sol.id ASC
`, orderID)
	if err != nil {
		return SalesOrder{}, fmt.Errorf("query sales order lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var line SalesOrderLine
		if err := rows.Scan(&line.ID, &line.ItemID, &line.ItemSKU, &line.ItemName, &line.Quantity, &line.UnitPrice, &line.LineTotal); err != nil {
			return SalesOrder{}, fmt.Errorf("scan sales order line: %w", err)
		}
		order.Lines = append(order.Lines, line)
	}
	return order, rows.Err()
}

func insertSalesOrderEventRecords(ctx context.Context, tx pgx.Tx, tenantID string, actorID string, order SalesOrder, action string) error {
	eventType := "sales.order.created.v1"
	if action == "confirm" {
		eventType = "sales.order.confirmed.v1"
	}
	if err := audit.Insert(ctx, tx, audit.Entry{
		TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID,
		Action: "sales.order." + action, TargetType: "sales.order", TargetID: order.ID,
		Metadata: map[string]any{
			"order_number": order.OrderNumber, "customer_id": order.Customer.ID,
			"currency_code": order.CurrencyCode, "line_count": len(order.Lines), "status": order.Status,
		},
	}); err != nil {
		return err
	}
	return outbox.Insert(ctx, tx, outbox.Event{
		TenantID: tenantID, EventType: eventType, EventVersion: 1,
		AggregateType: "sales.order", AggregateID: order.ID,
		Payload: map[string]any{
			"sales_order_id": order.ID, "order_number": order.OrderNumber,
			"customer_id": order.Customer.ID, "currency_code": order.CurrencyCode,
			"line_count": len(order.Lines), "status": order.Status,
		},
	})
}
