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
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return Store{
		db: db,
	}
}

type Receipt struct {
	ID                  string             `json:"id"`
	ReceiptNumber       string             `json:"receipt_number"`
	PurchaseOrderID     string             `json:"purchase_order_id"`
	PurchaseOrderNumber string             `json:"purchase_order_number"`
	Supplier            PurchaseOrderParty `json:"supplier"`
	SupplierName        string             `json:"supplier_name"`
	Reference           string             `json:"reference"`
	Notes               string             `json:"notes"`
	ReceivedAt          time.Time          `json:"received_at"`
	Status              string             `json:"status"`
	StockMovementID     string             `json:"stock_movement_id"`
	Lines               []ReceiptLine      `json:"lines"`
	CreatedAt           time.Time          `json:"created_at"`
}

type ReceiptLine struct {
	ID               string `json:"id"`
	ItemID           string `json:"item_id"`
	ItemSKU          string `json:"item_sku"`
	ItemName         string `json:"item_name"`
	LocationID       string `json:"location_id"`
	LocationCode     string `json:"location_code"`
	LocationName     string `json:"location_name"`
	QuantityReceived string `json:"quantity_received"`
}

type CreateReceiptInput struct {
	PurchaseOrderID string              `json:"purchase_order_id"`
	Reference       string              `json:"reference"`
	Notes           string              `json:"notes"`
	ReceivedAt      time.Time           `json:"received_at"`
	Lines           []CreateReceiptLine `json:"lines"`
}

var (
	ErrReceiptPurchaseOrderNotReceivable = errors.New("purchase order not found or not receivable")
	ErrReceiptItemNotOnOrder             = errors.New("receipt item is not on purchase order")
	ErrReceiptQuantityExceedsRemaining   = errors.New("receipt quantity exceeds purchase order remaining quantity")
)

type CreateReceiptLine struct {
	ItemID           string `json:"item_id"`
	LocationID       string `json:"location_id"`
	QuantityReceived string `json:"quantity_received"`
}

func (s Store) ListReceipts(ctx context.Context, tenantID string) ([]Receipt, error) {
	receipts := make([]Receipt, 0)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
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
    r.created_at
FROM purchase.receipts r
LEFT JOIN purchase.purchase_orders po
    ON po.tenant_id = r.tenant_id
   AND po.id = r.purchase_order_id
LEFT JOIN purchase.suppliers s
    ON s.tenant_id = r.tenant_id
   AND s.id = r.supplier_id
ORDER BY r.received_at DESC, r.created_at DESC, r.id DESC
LIMIT 100
`)
		if err != nil {
			return fmt.Errorf("query purchase receipts: %w", err)
		}
		defer rows.Close()

		receiptIndex := map[string]int{}

		for rows.Next() {
			var receipt Receipt
			if err := rows.Scan(
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
				&receipt.CreatedAt,
			); err != nil {
				return fmt.Errorf("scan purchase receipt: %w", err)
			}

			receipt.Lines = make([]ReceiptLine, 0)
			receipt.Supplier.Name = receipt.SupplierName
			receiptIndex[receipt.ID] = len(receipts)
			receipts = append(receipts, receipt)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate purchase receipts: %w", err)
		}

		if len(receipts) == 0 {
			return nil
		}

		lineRows, err := tx.Query(ctx, `
SELECT
    l.id::text,
    l.receipt_id::text,
    l.item_id::text,
    i.sku,
    i.name,
    l.location_id::text,
    loc.code,
    loc.name,
    l.quantity_received::text
FROM purchase.receipt_lines l
JOIN inventory.items i
    ON i.tenant_id = l.tenant_id
   AND i.id = l.item_id
JOIN inventory.locations loc
    ON loc.tenant_id = l.tenant_id
   AND loc.id = l.location_id
ORDER BY l.created_at ASC, l.id ASC
`)
		if err != nil {
			return fmt.Errorf("query purchase receipt lines: %w", err)
		}
		defer lineRows.Close()

		for lineRows.Next() {
			var receiptID string
			var line ReceiptLine

			if err := lineRows.Scan(
				&line.ID,
				&receiptID,
				&line.ItemID,
				&line.ItemSKU,
				&line.ItemName,
				&line.LocationID,
				&line.LocationCode,
				&line.LocationName,
				&line.QuantityReceived,
			); err != nil {
				return fmt.Errorf("scan purchase receipt line: %w", err)
			}

			index, ok := receiptIndex[receiptID]
			if !ok {
				continue
			}

			receipts[index].Lines = append(receipts[index].Lines, line)
		}

		if err := lineRows.Err(); err != nil {
			return fmt.Errorf("iterate purchase receipt lines: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

func (s Store) CreateReceipt(ctx context.Context, tenantID string, actorID string, input CreateReceiptInput) (Receipt, error) {
	input.PurchaseOrderID = strings.TrimSpace(input.PurchaseOrderID)
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)

	if input.ReceivedAt.IsZero() {
		input.ReceivedAt = time.Now().UTC()
	}

	var receipt Receipt

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var previousOrderStatus string
		if err := tx.QueryRow(ctx, `
SELECT po.id::text, po.order_number, s.id::text, s.code, s.name, po.status
FROM purchase.purchase_orders po
JOIN purchase.suppliers s
    ON s.tenant_id = po.tenant_id
   AND s.id = po.supplier_id
WHERE po.id = $1::uuid
  AND po.status IN ('approved', 'partially_received')
FOR UPDATE OF po
`, input.PurchaseOrderID).Scan(
			&receipt.PurchaseOrderID,
			&receipt.PurchaseOrderNumber,
			&receipt.Supplier.ID,
			&receipt.Supplier.Code,
			&receipt.Supplier.Name,
			&previousOrderStatus,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReceiptPurchaseOrderNotReceivable
			}
			return fmt.Errorf("load approved purchase order: %w", err)
		}
		receipt.SupplierName = receipt.Supplier.Name

		var receiptNumber string
		if err := tx.QueryRow(ctx, `
SELECT 'PR-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12))
`).Scan(&receiptNumber); err != nil {
			return fmt.Errorf("generate purchase receipt number: %w", err)
		}

		var stockMovementID string
		var stockMovementNumber string

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
    'receipt',
    $2,
    $3,
    $4,
    'posted',
    $5::uuid
)
RETURNING id::text, movement_number
`, tenantID, receiptNumber, input.Notes, input.ReceivedAt, actorID).Scan(
			&stockMovementID,
			&stockMovementNumber,
		); err != nil {
			return fmt.Errorf("insert purchase receipt stock movement: %w", err)
		}

		if err := tx.QueryRow(ctx, `
INSERT INTO purchase.receipts (
    tenant_id,
    receipt_number,
    supplier_id,
    purchase_order_id,
    supplier_name,
    reference,
    notes,
    received_at,
    status,
    stock_movement_id,
    created_by
)
VALUES (
    $1,
    $2,
    $3::uuid,
    $4::uuid,
    $5,
    $6,
    $7,
    $8,
    'posted',
    $9::uuid,
    $10::uuid
)
RETURNING id::text, receipt_number, supplier_name, reference, notes, received_at, status, stock_movement_id::text, created_at
`, tenantID, receiptNumber, receipt.Supplier.ID, receipt.PurchaseOrderID, receipt.SupplierName, input.Reference, input.Notes, input.ReceivedAt, stockMovementID, actorID).Scan(
			&receipt.ID,
			&receipt.ReceiptNumber,
			&receipt.SupplierName,
			&receipt.Reference,
			&receipt.Notes,
			&receipt.ReceivedAt,
			&receipt.Status,
			&receipt.StockMovementID,
			&receipt.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert purchase receipt: %w", err)
		}

		receipt.Lines = make([]ReceiptLine, 0, len(input.Lines))

		for _, inputLine := range input.Lines {
			inputLine.ItemID = strings.TrimSpace(inputLine.ItemID)
			inputLine.LocationID = strings.TrimSpace(inputLine.LocationID)
			inputLine.QuantityReceived = strings.TrimSpace(inputLine.QuantityReceived)

			var quantityAllowed bool
			if err := tx.QueryRow(ctx, `
SELECT $3::numeric <= (
    SUM(pol.quantity) - COALESCE((
        SELECT SUM(rl.quantity_received)
        FROM purchase.receipt_lines rl
        JOIN purchase.receipts r
            ON r.tenant_id = rl.tenant_id
           AND r.id = rl.receipt_id
        WHERE r.purchase_order_id = $1::uuid
          AND rl.item_id = $2::uuid
    ), 0)
)
FROM purchase.purchase_order_lines pol
WHERE pol.purchase_order_id = $1::uuid
  AND pol.item_id = $2::uuid
GROUP BY pol.purchase_order_id
`, receipt.PurchaseOrderID, inputLine.ItemID, inputLine.QuantityReceived).Scan(&quantityAllowed); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrReceiptItemNotOnOrder
				}
				return fmt.Errorf("check purchase order remaining quantity: %w", err)
			}
			if !quantityAllowed {
				return ErrReceiptQuantityExceedsRemaining
			}

			var line ReceiptLine
			if err := tx.QueryRow(ctx, `
INSERT INTO purchase.receipt_lines (
    tenant_id,
    receipt_id,
    item_id,
    location_id,
    quantity_received
)
VALUES (
    $1,
    $2::uuid,
    $3::uuid,
    $4::uuid,
    $5::numeric
)
RETURNING id::text, item_id::text, location_id::text, quantity_received::text
`, tenantID, receipt.ID, inputLine.ItemID, inputLine.LocationID, inputLine.QuantityReceived).Scan(
				&line.ID,
				&line.ItemID,
				&line.LocationID,
				&line.QuantityReceived,
			); err != nil {
				return fmt.Errorf("insert purchase receipt line: %w", err)
			}

			if _, err := tx.Exec(ctx, `
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
`, tenantID, stockMovementID, inputLine.ItemID, inputLine.LocationID, inputLine.QuantityReceived); err != nil {
				return fmt.Errorf("insert purchase receipt stock movement line: %w", err)
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
				return fmt.Errorf("load purchase receipt line refs: %w", err)
			}

			receipt.Lines = append(receipt.Lines, line)
		}

		var fullyReceived bool
		if err := tx.QueryRow(ctx, `
SELECT bool_and(COALESCE(receipt_progress.received_quantity, 0) >= ordered.quantity)
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
) receipt_progress ON true
`, receipt.PurchaseOrderID).Scan(&fullyReceived); err != nil {
			return fmt.Errorf("calculate purchase order receipt progress: %w", err)
		}

		orderStatus := "partially_received"
		if fullyReceived {
			orderStatus = "received"
		}
		if orderStatus != previousOrderStatus {
			if _, err := tx.Exec(ctx, `
UPDATE purchase.purchase_orders
SET status = $2,
    updated_at = now()
WHERE id = $1::uuid
`, receipt.PurchaseOrderID, orderStatus); err != nil {
				return fmt.Errorf("update purchase order receipt status: %w", err)
			}

			order, err := loadPurchaseOrder(ctx, tx, receipt.PurchaseOrderID)
			if err != nil {
				return err
			}
			if err := insertPurchaseOrderEventRecords(ctx, tx, tenantID, actorID, order, orderStatus); err != nil {
				return err
			}
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID:   tenantID,
			ActorType:  "tenant_user",
			ActorID:    actorID,
			Action:     "purchase.receipt.create",
			TargetType: "purchase.receipt",
			TargetID:   receipt.ID,
			Metadata: map[string]any{
				"receipt_number":        receipt.ReceiptNumber,
				"purchase_order_id":     receipt.PurchaseOrderID,
				"purchase_order_number": receipt.PurchaseOrderNumber,
				"supplier_id":           receipt.Supplier.ID,
				"supplier_name":         receipt.SupplierName,
				"reference":             receipt.Reference,
				"line_count":            len(receipt.Lines),
				"stock_movement_id":     stockMovementID,
				"stock_movement_number": stockMovementNumber,
			},
		}); err != nil {
			return err
		}

		if err := outbox.Insert(ctx, tx, outbox.Event{
			TenantID:      tenantID,
			EventType:     "purchase.receipt.created.v1",
			EventVersion:  1,
			AggregateType: "purchase.receipt",
			AggregateID:   receipt.ID,
			Payload: map[string]any{
				"receipt_id":            receipt.ID,
				"receipt_number":        receipt.ReceiptNumber,
				"purchase_order_id":     receipt.PurchaseOrderID,
				"purchase_order_number": receipt.PurchaseOrderNumber,
				"supplier_id":           receipt.Supplier.ID,
				"supplier_name":         receipt.SupplierName,
				"reference":             receipt.Reference,
				"line_count":            len(receipt.Lines),
				"stock_movement_id":     stockMovementID,
				"stock_movement_number": stockMovementNumber,
			},
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return Receipt{}, err
	}

	return receipt, nil
}
