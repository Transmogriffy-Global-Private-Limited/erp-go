package inventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

func PostIssueReversalMovement(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	actorID string,
	reference string,
	reason string,
	inputLines []PostIssueMovementLine,
) (StockMovement, error) {
	var movement StockMovement
	if err := tx.QueryRow(ctx, `
INSERT INTO inventory.stock_movements (
    tenant_id, movement_number, movement_type, reference, notes,
    occurred_at, status, created_by
)
VALUES (
    $1, 'SM-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12)),
    'issue_reversal', $2, $3, now(), 'posted', $4::uuid
)
RETURNING id::text, movement_number, movement_type, reference, notes,
          occurred_at, status, created_at
`, tenantID, strings.TrimSpace(reference), strings.TrimSpace(reason), actorID).Scan(
		&movement.ID, &movement.MovementNumber, &movement.MovementType,
		&movement.Reference, &movement.Notes, &movement.OccurredAt,
		&movement.Status, &movement.CreatedAt,
	); err != nil {
		return StockMovement{}, fmt.Errorf("insert issue reversal stock movement: %w", err)
	}

	movement.Lines = make([]StockMovementLine, 0, len(inputLines))
	for _, inputLine := range inputLines {
		lockKey := tenantID + ":" + inputLine.ItemID + ":" + inputLine.LocationID
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
			return StockMovement{}, fmt.Errorf("lock issue reversal stock balance: %w", err)
		}

		var line StockMovementLine
		if err := tx.QueryRow(ctx, `
INSERT INTO inventory.stock_movement_lines (
    tenant_id, movement_id, item_id, location_id, quantity_delta
)
VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5::numeric)
RETURNING id::text, item_id::text, location_id::text, quantity_delta::text
`, tenantID, movement.ID, inputLine.ItemID, inputLine.LocationID, inputLine.Quantity).Scan(
			&line.ID, &line.ItemID, &line.LocationID, &line.QuantityDelta,
		); err != nil {
			return StockMovement{}, fmt.Errorf("insert issue reversal stock movement line: %w", err)
		}
		if err := tx.QueryRow(ctx, `
SELECT i.sku, i.name, loc.code, loc.name
FROM inventory.items i
JOIN inventory.locations loc ON loc.tenant_id = i.tenant_id
WHERE i.id = $1::uuid AND loc.id = $2::uuid
`, line.ItemID, line.LocationID).Scan(
			&line.ItemSKU, &line.ItemName, &line.LocationCode, &line.LocationName,
		); err != nil {
			return StockMovement{}, fmt.Errorf("load issue reversal stock references: %w", err)
		}
		movement.Lines = append(movement.Lines, line)
	}

	if err := audit.Insert(ctx, tx, audit.Entry{
		TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID,
		Action: "inventory.stock_movement.create", TargetType: "inventory.stock_movement", TargetID: movement.ID,
		Metadata: map[string]any{
			"movement_number": movement.MovementNumber, "movement_type": movement.MovementType,
			"reference": movement.Reference, "line_count": len(movement.Lines),
		},
	}); err != nil {
		return StockMovement{}, err
	}
	if err := outbox.Insert(ctx, tx, outbox.Event{
		TenantID: tenantID, EventType: "inventory.stock_movement.created.v1", EventVersion: 1,
		AggregateType: "inventory.stock_movement", AggregateID: movement.ID,
		Payload: map[string]any{
			"movement_id": movement.ID, "movement_number": movement.MovementNumber,
			"movement_type": movement.MovementType, "reference": movement.Reference,
			"line_count": len(movement.Lines),
		},
	}); err != nil {
		return StockMovement{}, err
	}

	return movement, nil
}
