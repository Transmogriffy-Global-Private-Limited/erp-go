package sales

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

var (
	ErrIssueNotFound        = errors.New("sales issue not found")
	ErrIssueNotReversible   = errors.New("sales issue is not reversible")
	ErrIssueAlreadyReversed = errors.New("sales issue is already reversed")
)

func (s Store) ReverseIssue(ctx context.Context, tenantID string, actorID string, issueID string, reason string) (Issue, error) {
	reason = strings.TrimSpace(reason)
	var issue Issue
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
SELECT si.id::text, si.issue_number, so.id::text, so.order_number,
       c.id::text, c.code, si.customer_name, si.reference, si.notes,
       si.issued_at, si.status, si.stock_movement_id::text, sm.movement_number,
       COALESCE(si.reversal_stock_movement_id::text, ''),
       COALESCE(rsm.movement_number, ''), si.reversal_reason, si.reversed_at,
       si.created_at
FROM sales.issues si
JOIN sales.orders so ON so.tenant_id = si.tenant_id AND so.id = si.sales_order_id
JOIN sales.customers c ON c.tenant_id = si.tenant_id AND c.id = si.customer_id
JOIN inventory.stock_movements sm
  ON sm.tenant_id = si.tenant_id
 AND sm.id = si.stock_movement_id
LEFT JOIN inventory.stock_movements rsm
  ON rsm.tenant_id = si.tenant_id
 AND rsm.id = si.reversal_stock_movement_id
WHERE si.id = $1::uuid
FOR UPDATE OF si
`, issueID).Scan(
			&issue.ID, &issue.IssueNumber, &issue.SalesOrderID, &issue.SalesOrderNumber,
			&issue.Customer.ID, &issue.Customer.Code, &issue.CustomerName,
			&issue.Reference, &issue.Notes, &issue.IssuedAt, &issue.Status,
			&issue.StockMovementID, &issue.StockMovementNumber,
			&issue.ReversalStockMovementID, &issue.ReversalStockMovementNumber,
			&issue.ReversalReason, &issue.ReversedAt, &issue.CreatedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrIssueNotFound
			}
			return fmt.Errorf("lock sales issue: %w", err)
		}
		issue.Customer.Name = issue.CustomerName
		if issue.Status == "reversed" {
			return ErrIssueAlreadyReversed
		}
		if issue.Status != "posted" {
			return ErrIssueNotReversible
		}

		var previousOrderStatus string
		if err := tx.QueryRow(ctx, `
SELECT status FROM sales.orders WHERE id = $1::uuid FOR UPDATE
`, issue.SalesOrderID).Scan(&previousOrderStatus); err != nil {
			return fmt.Errorf("lock sales order for issue reversal: %w", err)
		}

		rows, err := tx.Query(ctx, `
SELECT sil.id::text, sil.item_id::text, i.sku, i.name,
       sil.location_id::text, loc.code, loc.name, sil.quantity_issued::text
FROM sales.issue_lines sil
JOIN inventory.items i ON i.tenant_id = sil.tenant_id AND i.id = sil.item_id
JOIN inventory.locations loc ON loc.tenant_id = sil.tenant_id AND loc.id = sil.location_id
WHERE sil.issue_id = $1::uuid
ORDER BY sil.item_id, sil.location_id
`, issue.ID)
		if err != nil {
			return fmt.Errorf("query sales issue lines for reversal: %w", err)
		}
		issue.Lines = make([]IssueLine, 0)
		movementLines := make([]inventory.PostIssueMovementLine, 0)
		for rows.Next() {
			var line IssueLine
			if err := rows.Scan(
				&line.ID, &line.ItemID, &line.ItemSKU, &line.ItemName,
				&line.LocationID, &line.LocationCode, &line.LocationName, &line.QuantityIssued,
			); err != nil {
				rows.Close()
				return fmt.Errorf("scan sales issue line for reversal: %w", err)
			}
			issue.Lines = append(issue.Lines, line)
			movementLines = append(movementLines, inventory.PostIssueMovementLine{
				ItemID: line.ItemID, LocationID: line.LocationID, Quantity: line.QuantityIssued,
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate sales issue lines for reversal: %w", err)
		}
		rows.Close()
		if len(issue.Lines) == 0 {
			return ErrIssueNotReversible
		}

		movement, err := inventory.PostIssueReversalMovement(
			ctx, tx, tenantID, actorID, issue.IssueNumber, reason, movementLines,
		)
		if err != nil {
			return err
		}
		issue.ReversalStockMovementID = movement.ID
		issue.ReversalStockMovementNumber = movement.MovementNumber

		if err := tx.QueryRow(ctx, `
UPDATE sales.issues
SET status = 'reversed', reversal_stock_movement_id = $2::uuid,
    reversal_reason = $3, reversed_by = $4::uuid, reversed_at = now()
WHERE id = $1::uuid
RETURNING status, reversal_reason, reversed_at
`, issue.ID, issue.ReversalStockMovementID, reason, actorID).Scan(
			&issue.Status, &issue.ReversalReason, &issue.ReversedAt,
		); err != nil {
			return fmt.Errorf("mark sales issue reversed: %w", err)
		}

		orderStatus, err := salesOrderStatusFromPostedIssues(ctx, tx, issue.SalesOrderID)
		if err != nil {
			return err
		}
		if orderStatus != previousOrderStatus {
			if _, err := tx.Exec(ctx, `
UPDATE sales.orders SET status = $2, updated_at = now() WHERE id = $1::uuid
`, issue.SalesOrderID, orderStatus); err != nil {
				return fmt.Errorf("reopen sales order fulfillment progress: %w", err)
			}
			order, err := loadSalesOrder(ctx, tx, issue.SalesOrderID)
			if err != nil {
				return err
			}
			if err := insertSalesOrderEventRecords(ctx, tx, tenantID, actorID, order, "fulfillment_reopened"); err != nil {
				return err
			}
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID,
			Action: "sales.issue.reverse", TargetType: "sales.issue", TargetID: issue.ID,
			Metadata: map[string]any{
				"issue_number": issue.IssueNumber, "sales_order_id": issue.SalesOrderID,
				"reason": reason, "original_stock_movement_id": issue.StockMovementID,
				"reversal_stock_movement_id": issue.ReversalStockMovementID,
			},
		}); err != nil {
			return err
		}
		return outbox.Insert(ctx, tx, outbox.Event{
			TenantID: tenantID, EventType: "sales.issue.reversed.v1", EventVersion: 1,
			AggregateType: "sales.issue", AggregateID: issue.ID,
			Payload: map[string]any{
				"issue_id": issue.ID, "issue_number": issue.IssueNumber,
				"sales_order_id": issue.SalesOrderID, "reason": reason,
				"original_stock_movement_id": issue.StockMovementID,
				"reversal_stock_movement_id": issue.ReversalStockMovementID,
			},
		})
	})
	if err != nil {
		return Issue{}, err
	}
	return issue, nil
}

func salesOrderStatusFromPostedIssues(ctx context.Context, tx pgx.Tx, orderID string) (string, error) {
	var anyIssued bool
	var fullyFulfilled bool
	if err := tx.QueryRow(ctx, `
SELECT
    COALESCE(bool_or(COALESCE(issue_progress.issued_quantity, 0) > 0), false),
    COALESCE(bool_and(COALESCE(issue_progress.issued_quantity, 0) >= ordered.quantity), false)
FROM (
    SELECT item_id, SUM(quantity) AS quantity
    FROM sales.order_lines
    WHERE sales_order_id = $1::uuid
    GROUP BY item_id
) ordered
LEFT JOIN LATERAL (
    SELECT SUM(sil.quantity_issued) AS issued_quantity
    FROM sales.issues si
    JOIN sales.issue_lines sil
      ON sil.tenant_id = si.tenant_id
     AND sil.issue_id = si.id
    WHERE si.sales_order_id = $1::uuid
      AND sil.item_id = ordered.item_id
      AND si.status = 'posted'
) issue_progress ON true
`, orderID).Scan(&anyIssued, &fullyFulfilled); err != nil {
		return "", fmt.Errorf("calculate reopened sales order fulfillment status: %w", err)
	}
	if fullyFulfilled {
		return "fulfilled", nil
	}
	if anyIssued {
		return "partially_fulfilled", nil
	}
	return "confirmed", nil
}
