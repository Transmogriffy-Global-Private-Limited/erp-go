package purchase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

var (
	ErrReceiptNotFound        = errors.New("purchase receipt not found")
	ErrReceiptNotReversible   = errors.New("purchase receipt is not reversible")
	ErrReceiptAlreadyReversed = errors.New("purchase receipt is already reversed")
)

func (s Store) ReverseReceipt(ctx context.Context, tenantID string, actorID string, receiptID string, reason string) (Receipt, error) {
	reason = strings.TrimSpace(reason)
	var receipt Receipt

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
SELECT
    r.id::text,
    r.receipt_number,
    COALESCE(r.purchase_order_id::text, ''),
    COALESCE(po.order_number, ''),
    COALESCE(r.supplier_id::text, ''),
    COALESCE(s.code, ''),
    r.supplier_name,
    r.reference,
    r.notes,
    r.received_at,
    r.status,
    r.stock_movement_id::text,
    COALESCE(r.reversal_stock_movement_id::text, ''),
    r.reversal_reason,
    r.reversed_at,
    r.created_at
FROM purchase.receipts r
LEFT JOIN purchase.purchase_orders po
    ON po.tenant_id = r.tenant_id
   AND po.id = r.purchase_order_id
LEFT JOIN purchase.suppliers s
    ON s.tenant_id = r.tenant_id
   AND s.id = r.supplier_id
WHERE r.id = $1::uuid
FOR UPDATE OF r
`, receiptID).Scan(
			&receipt.ID,
			&receipt.ReceiptNumber,
			&receipt.PurchaseOrderID,
			&receipt.PurchaseOrderNumber,
			&receipt.Supplier.ID,
			&receipt.Supplier.Code,
			&receipt.SupplierName,
			&receipt.Reference,
			&receipt.Notes,
			&receipt.ReceivedAt,
			&receipt.Status,
			&receipt.StockMovementID,
			&receipt.ReversalStockMovementID,
			&receipt.ReversalReason,
			&receipt.ReversedAt,
			&receipt.CreatedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReceiptNotFound
			}
			return fmt.Errorf("lock purchase receipt: %w", err)
		}
		receipt.Supplier.Name = receipt.SupplierName

		if receipt.Status == "reversed" {
			return ErrReceiptAlreadyReversed
		}
		if receipt.Status != "posted" || receipt.PurchaseOrderID == "" {
			return ErrReceiptNotReversible
		}

		var previousOrderStatus string
		if err := tx.QueryRow(ctx, `
SELECT status
FROM purchase.purchase_orders
WHERE id = $1::uuid
FOR UPDATE
`, receipt.PurchaseOrderID).Scan(&previousOrderStatus); err != nil {
			return fmt.Errorf("lock purchase order for receipt reversal: %w", err)
		}

		var reversalMovementNumber string
		if err := tx.QueryRow(ctx, `
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
    'receipt_reversal',
    $2,
    $3,
    now(),
    'posted',
    $4::uuid
)
RETURNING id::text, movement_number
`, tenantID, receipt.ReceiptNumber, reason, actorID).Scan(
			&receipt.ReversalStockMovementID,
			&reversalMovementNumber,
		); err != nil {
			return fmt.Errorf("insert receipt reversal stock movement: %w", err)
		}

		rows, err := tx.Query(ctx, `
SELECT rl.id::text, rl.item_id::text, i.sku, i.name,
       rl.location_id::text, loc.code, loc.name, rl.quantity_received::text
FROM purchase.receipt_lines rl
JOIN inventory.items i
    ON i.tenant_id = rl.tenant_id
   AND i.id = rl.item_id
JOIN inventory.locations loc
    ON loc.tenant_id = rl.tenant_id
   AND loc.id = rl.location_id
WHERE rl.receipt_id = $1::uuid
ORDER BY rl.created_at ASC, rl.id ASC
`, receipt.ID)
		if err != nil {
			return fmt.Errorf("query receipt lines for reversal: %w", err)
		}
		defer rows.Close()

		receipt.Lines = make([]ReceiptLine, 0)
		for rows.Next() {
			var line ReceiptLine
			if err := rows.Scan(
				&line.ID,
				&line.ItemID,
				&line.ItemSKU,
				&line.ItemName,
				&line.LocationID,
				&line.LocationCode,
				&line.LocationName,
				&line.QuantityReceived,
			); err != nil {
				return fmt.Errorf("scan receipt line for reversal: %w", err)
			}

			receipt.Lines = append(receipt.Lines, line)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate receipt lines for reversal: %w", err)
		}
		rows.Close()

		for _, line := range receipt.Lines {
			if _, err := tx.Exec(ctx, `
INSERT INTO inventory.stock_movement_lines (
    tenant_id,
    movement_id,
    item_id,
    location_id,
    quantity_delta
)
VALUES ($1, $2::uuid, $3::uuid, $4::uuid, -($5::numeric))
`, tenantID, receipt.ReversalStockMovementID, line.ItemID, line.LocationID, line.QuantityReceived); err != nil {
				return fmt.Errorf("insert receipt reversal stock movement line: %w", err)
			}
		}

		if err := tx.QueryRow(ctx, `
UPDATE purchase.receipts
SET status = 'reversed',
    reversal_stock_movement_id = $2::uuid,
    reversal_reason = $3,
    reversed_by = $4::uuid,
    reversed_at = now()
WHERE id = $1::uuid
RETURNING status, reversal_reason, reversed_at
`, receipt.ID, receipt.ReversalStockMovementID, reason, actorID).Scan(
			&receipt.Status,
			&receipt.ReversalReason,
			&receipt.ReversedAt,
		); err != nil {
			return fmt.Errorf("mark purchase receipt reversed: %w", err)
		}

		orderStatus, err := purchaseOrderStatusFromPostedReceipts(ctx, tx, receipt.PurchaseOrderID)
		if err != nil {
			return err
		}
		if orderStatus != previousOrderStatus {
			if _, err := tx.Exec(ctx, `
UPDATE purchase.purchase_orders
SET status = $2,
    updated_at = now()
WHERE id = $1::uuid
`, receipt.PurchaseOrderID, orderStatus); err != nil {
				return fmt.Errorf("reopen purchase order receipt progress: %w", err)
			}

			order, err := loadPurchaseOrder(ctx, tx, receipt.PurchaseOrderID)
			if err != nil {
				return err
			}
			if err := insertPurchaseOrderEventRecords(ctx, tx, tenantID, actorID, order, "receipt_progress_reopened"); err != nil {
				return err
			}
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID:   tenantID,
			ActorType:  "tenant_user",
			ActorID:    actorID,
			Action:     "purchase.receipt.reverse",
			TargetType: "purchase.receipt",
			TargetID:   receipt.ID,
			Metadata: map[string]any{
				"receipt_number":                 receipt.ReceiptNumber,
				"purchase_order_id":              receipt.PurchaseOrderID,
				"reason":                         reason,
				"original_stock_movement_id":     receipt.StockMovementID,
				"reversal_stock_movement_id":     receipt.ReversalStockMovementID,
				"reversal_stock_movement_number": reversalMovementNumber,
			},
		}); err != nil {
			return err
		}

		return outbox.Insert(ctx, tx, outbox.Event{
			TenantID:      tenantID,
			EventType:     "purchase.receipt.reversed.v1",
			EventVersion:  1,
			AggregateType: "purchase.receipt",
			AggregateID:   receipt.ID,
			Payload: map[string]any{
				"receipt_id":                     receipt.ID,
				"receipt_number":                 receipt.ReceiptNumber,
				"purchase_order_id":              receipt.PurchaseOrderID,
				"reason":                         reason,
				"original_stock_movement_id":     receipt.StockMovementID,
				"reversal_stock_movement_id":     receipt.ReversalStockMovementID,
				"reversal_stock_movement_number": reversalMovementNumber,
			},
		})
	})
	if err != nil {
		return Receipt{}, err
	}

	return receipt, nil
}

func purchaseOrderStatusFromPostedReceipts(ctx context.Context, tx pgx.Tx, orderID string) (string, error) {
	var anyReceived bool
	var fullyReceived bool
	if err := tx.QueryRow(ctx, `
SELECT
    COALESCE(bool_or(COALESCE(receipt_progress.received_quantity, 0) > 0), false),
    COALESCE(bool_and(COALESCE(receipt_progress.received_quantity, 0) >= ordered.quantity), false)
FROM (
    SELECT item_id, SUM(quantity) AS quantity
    FROM purchase.purchase_order_lines
    WHERE purchase_order_id = $1::uuid
    GROUP BY item_id
) ordered
LEFT JOIN LATERAL (
    SELECT SUM(rl.quantity_received) AS received_quantity
    FROM purchase.receipts r
    JOIN purchase.receipt_lines rl
        ON rl.tenant_id = r.tenant_id
       AND rl.receipt_id = r.id
    WHERE r.purchase_order_id = $1::uuid
      AND rl.item_id = ordered.item_id
      AND r.status = 'posted'
) receipt_progress ON true
`, orderID).Scan(&anyReceived, &fullyReceived); err != nil {
		return "", fmt.Errorf("calculate purchase order status after reversal: %w", err)
	}

	if fullyReceived {
		return "received", nil
	}
	if anyReceived {
		return "partially_received", nil
	}
	return "approved", nil
}
