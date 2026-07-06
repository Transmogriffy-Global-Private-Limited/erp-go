package sales

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

var (
	ErrIssueSalesOrderNotFulfillable = errors.New("sales order not found or not fulfillable")
	ErrIssueItemNotOnOrder           = errors.New("issue item is not on sales order")
	ErrIssueQuantityExceedsRemaining = errors.New("issue quantity exceeds sales order remaining quantity")
	ErrIssueDuplicateItem            = errors.New("issue contains duplicate item")
)

type Issue struct {
	ID                  string          `json:"id"`
	IssueNumber         string          `json:"issue_number"`
	SalesOrderID        string          `json:"sales_order_id"`
	SalesOrderNumber    string          `json:"sales_order_number"`
	Customer            SalesOrderParty `json:"customer"`
	CustomerName        string          `json:"customer_name"`
	Reference           string          `json:"reference"`
	Notes               string          `json:"notes"`
	IssuedAt            time.Time       `json:"issued_at"`
	Status              string          `json:"status"`
	StockMovementID     string          `json:"stock_movement_id"`
	StockMovementNumber string          `json:"stock_movement_number"`
	Lines               []IssueLine     `json:"lines"`
	CreatedAt           time.Time       `json:"created_at"`
}

type IssueLine struct {
	ID             string `json:"id"`
	ItemID         string `json:"item_id"`
	ItemSKU        string `json:"item_sku"`
	ItemName       string `json:"item_name"`
	LocationID     string `json:"location_id"`
	LocationCode   string `json:"location_code"`
	LocationName   string `json:"location_name"`
	QuantityIssued string `json:"quantity_issued"`
}

type CreateIssueInput struct {
	SalesOrderID string            `json:"sales_order_id"`
	Reference    string            `json:"reference"`
	Notes        string            `json:"notes"`
	IssuedAt     time.Time         `json:"issued_at"`
	Lines        []CreateIssueLine `json:"lines"`
}

type CreateIssueLine struct {
	ItemID         string `json:"item_id"`
	LocationID     string `json:"location_id"`
	QuantityIssued string `json:"quantity_issued"`
}

func (s Store) ListIssues(ctx context.Context, tenantID string) ([]Issue, error) {
	issues := make([]Issue, 0)
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT si.id::text, si.issue_number, so.id::text, so.order_number,
       c.id::text, c.code, si.customer_name, si.reference, si.notes,
       si.issued_at, si.status, si.stock_movement_id::text,
       sm.movement_number, si.created_at
FROM sales.issues si
JOIN sales.orders so ON so.tenant_id = si.tenant_id AND so.id = si.sales_order_id
JOIN sales.customers c ON c.tenant_id = si.tenant_id AND c.id = si.customer_id
JOIN inventory.stock_movements sm
  ON sm.tenant_id = si.tenant_id
 AND sm.id = si.stock_movement_id
ORDER BY si.issued_at DESC, si.created_at DESC, si.id DESC
LIMIT 100
`)
		if err != nil {
			return fmt.Errorf("query sales issues: %w", err)
		}
		defer rows.Close()

		issueIndex := map[string]int{}
		for rows.Next() {
			var issue Issue
			if err := rows.Scan(
				&issue.ID, &issue.IssueNumber, &issue.SalesOrderID, &issue.SalesOrderNumber,
				&issue.Customer.ID, &issue.Customer.Code, &issue.CustomerName,
				&issue.Reference, &issue.Notes, &issue.IssuedAt, &issue.Status,
				&issue.StockMovementID, &issue.StockMovementNumber, &issue.CreatedAt,
			); err != nil {
				return fmt.Errorf("scan sales issue: %w", err)
			}
			issue.Customer.Name = issue.CustomerName
			issue.Lines = make([]IssueLine, 0)
			issueIndex[issue.ID] = len(issues)
			issues = append(issues, issue)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate sales issues: %w", err)
		}
		if len(issues) == 0 {
			return nil
		}

		lineRows, err := tx.Query(ctx, `
SELECT sil.id::text, sil.issue_id::text, sil.item_id::text, i.sku, i.name,
       sil.location_id::text, loc.code, loc.name, sil.quantity_issued::text
FROM sales.issue_lines sil
JOIN inventory.items i ON i.tenant_id = sil.tenant_id AND i.id = sil.item_id
JOIN inventory.locations loc ON loc.tenant_id = sil.tenant_id AND loc.id = sil.location_id
ORDER BY sil.created_at ASC, sil.id ASC
`)
		if err != nil {
			return fmt.Errorf("query sales issue lines: %w", err)
		}
		defer lineRows.Close()
		for lineRows.Next() {
			var issueID string
			var line IssueLine
			if err := lineRows.Scan(
				&line.ID, &issueID, &line.ItemID, &line.ItemSKU, &line.ItemName,
				&line.LocationID, &line.LocationCode, &line.LocationName, &line.QuantityIssued,
			); err != nil {
				return fmt.Errorf("scan sales issue line: %w", err)
			}
			if index, ok := issueIndex[issueID]; ok {
				issues[index].Lines = append(issues[index].Lines, line)
			}
		}
		return lineRows.Err()
	})
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func (s Store) CreateIssue(ctx context.Context, tenantID string, actorID string, input CreateIssueInput) (Issue, error) {
	input.SalesOrderID = strings.TrimSpace(input.SalesOrderID)
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.IssuedAt.IsZero() {
		input.IssuedAt = time.Now().UTC()
	}
	for i := range input.Lines {
		input.Lines[i].ItemID = strings.TrimSpace(input.Lines[i].ItemID)
		input.Lines[i].LocationID = strings.TrimSpace(input.Lines[i].LocationID)
		input.Lines[i].QuantityIssued = strings.TrimSpace(input.Lines[i].QuantityIssued)
	}
	sort.Slice(input.Lines, func(i, j int) bool {
		if input.Lines[i].ItemID == input.Lines[j].ItemID {
			return input.Lines[i].LocationID < input.Lines[j].LocationID
		}
		return input.Lines[i].ItemID < input.Lines[j].ItemID
	})
	for i := 1; i < len(input.Lines); i++ {
		if input.Lines[i-1].ItemID == input.Lines[i].ItemID {
			return Issue{}, ErrIssueDuplicateItem
		}
	}

	var issue Issue
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var previousOrderStatus string
		if err := tx.QueryRow(ctx, `
SELECT so.id::text, so.order_number, c.id::text, c.code, c.name, so.status
FROM sales.orders so
JOIN sales.customers c ON c.tenant_id = so.tenant_id AND c.id = so.customer_id
WHERE so.id = $1::uuid
  AND so.status IN ('confirmed', 'partially_fulfilled')
FOR UPDATE OF so
`, input.SalesOrderID).Scan(
			&issue.SalesOrderID, &issue.SalesOrderNumber,
			&issue.Customer.ID, &issue.Customer.Code, &issue.Customer.Name,
			&previousOrderStatus,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrIssueSalesOrderNotFulfillable
			}
			return fmt.Errorf("load fulfillable sales order: %w", err)
		}
		issue.CustomerName = issue.Customer.Name

		if err := tx.QueryRow(ctx, `
SELECT 'SI-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12))
`).Scan(&issue.IssueNumber); err != nil {
			return fmt.Errorf("generate sales issue number: %w", err)
		}

		movementLines := make([]inventory.PostIssueMovementLine, 0, len(input.Lines))
		for _, inputLine := range input.Lines {
			var allowed bool
			if err := tx.QueryRow(ctx, `
SELECT $3::numeric <= (
    sol.quantity - COALESCE((
        SELECT SUM(sil.quantity_issued)
        FROM sales.issues si
        JOIN sales.issue_lines sil
          ON sil.tenant_id = si.tenant_id
         AND sil.issue_id = si.id
        WHERE si.sales_order_id = $1::uuid
          AND sil.item_id = $2::uuid
          AND si.status = 'posted'
    ), 0)
)
FROM sales.order_lines sol
WHERE sol.sales_order_id = $1::uuid
  AND sol.item_id = $2::uuid
`, issue.SalesOrderID, inputLine.ItemID, inputLine.QuantityIssued).Scan(&allowed); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrIssueItemNotOnOrder
				}
				return fmt.Errorf("check sales order remaining quantity: %w", err)
			}
			if !allowed {
				return ErrIssueQuantityExceedsRemaining
			}
			movementLines = append(movementLines, inventory.PostIssueMovementLine{
				ItemID: inputLine.ItemID, LocationID: inputLine.LocationID, Quantity: inputLine.QuantityIssued,
			})
		}

		movement, err := inventory.PostIssueMovement(
			ctx, tx, tenantID, actorID, issue.IssueNumber, input.Notes, input.IssuedAt, movementLines,
		)
		if err != nil {
			return err
		}
		issue.StockMovementID = movement.ID
		issue.StockMovementNumber = movement.MovementNumber

		if err := tx.QueryRow(ctx, `
INSERT INTO sales.issues (
    tenant_id, issue_number, sales_order_id, customer_id, customer_name,
    reference, notes, issued_at, status, stock_movement_id, created_by
)
VALUES ($1, $2, $3::uuid, $4::uuid, $5, $6, $7, $8, 'posted', $9::uuid, $10::uuid)
RETURNING id::text, reference, notes, issued_at, status, created_at
`, tenantID, issue.IssueNumber, issue.SalesOrderID, issue.Customer.ID, issue.CustomerName,
			input.Reference, input.Notes, input.IssuedAt, issue.StockMovementID, actorID).Scan(
			&issue.ID, &issue.Reference, &issue.Notes, &issue.IssuedAt, &issue.Status, &issue.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert sales issue: %w", err)
		}

		movementByItem := make(map[string]inventory.StockMovementLine, len(movement.Lines))
		for _, line := range movement.Lines {
			movementByItem[line.ItemID] = line
		}
		issue.Lines = make([]IssueLine, 0, len(input.Lines))
		for _, inputLine := range input.Lines {
			movementLine := movementByItem[inputLine.ItemID]
			var line IssueLine
			if err := tx.QueryRow(ctx, `
INSERT INTO sales.issue_lines (
    tenant_id, issue_id, item_id, location_id, quantity_issued
)
VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5::numeric)
RETURNING id::text, item_id::text, location_id::text, quantity_issued::text
`, tenantID, issue.ID, inputLine.ItemID, inputLine.LocationID, inputLine.QuantityIssued).Scan(
				&line.ID, &line.ItemID, &line.LocationID, &line.QuantityIssued,
			); err != nil {
				return fmt.Errorf("insert sales issue line: %w", err)
			}
			line.ItemSKU = movementLine.ItemSKU
			line.ItemName = movementLine.ItemName
			line.LocationCode = movementLine.LocationCode
			line.LocationName = movementLine.LocationName
			issue.Lines = append(issue.Lines, line)
		}

		var fullyFulfilled bool
		if err := tx.QueryRow(ctx, `
SELECT bool_and(COALESCE(issue_progress.issued_quantity, 0) >= ordered.quantity)
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
`, issue.SalesOrderID).Scan(&fullyFulfilled); err != nil {
			return fmt.Errorf("calculate sales order fulfillment progress: %w", err)
		}

		orderStatus := "partially_fulfilled"
		if fullyFulfilled {
			orderStatus = "fulfilled"
		}
		if _, err := tx.Exec(ctx, `
UPDATE sales.orders SET status = $2, updated_at = now() WHERE id = $1::uuid
`, issue.SalesOrderID, orderStatus); err != nil {
			return fmt.Errorf("update sales order fulfillment status: %w", err)
		}
		if orderStatus != previousOrderStatus {
			order, err := loadSalesOrder(ctx, tx, issue.SalesOrderID)
			if err != nil {
				return err
			}
			if err := insertSalesOrderEventRecords(ctx, tx, tenantID, actorID, order, orderStatus); err != nil {
				return err
			}
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID,
			Action: "sales.issue.post", TargetType: "sales.issue", TargetID: issue.ID,
			Metadata: map[string]any{
				"issue_number": issue.IssueNumber, "sales_order_id": issue.SalesOrderID,
				"customer_id": issue.Customer.ID, "line_count": len(issue.Lines),
				"stock_movement_id": issue.StockMovementID,
			},
		}); err != nil {
			return err
		}
		return outbox.Insert(ctx, tx, outbox.Event{
			TenantID: tenantID, EventType: "sales.issue.posted.v1", EventVersion: 1,
			AggregateType: "sales.issue", AggregateID: issue.ID,
			Payload: map[string]any{
				"issue_id": issue.ID, "issue_number": issue.IssueNumber,
				"sales_order_id": issue.SalesOrderID, "customer_id": issue.Customer.ID,
				"line_count": len(issue.Lines), "stock_movement_id": issue.StockMovementID,
			},
		})
	})
	if err != nil {
		return Issue{}, err
	}
	return issue, nil
}
