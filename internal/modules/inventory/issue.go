package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

var (
	ErrIssueItemOrLocationNotFound = errors.New("issue item or location not found or inactive")
	ErrIssueInsufficientStock      = errors.New("insufficient stock for issue")
)

type PostIssueMovementLine struct {
	ItemID     string
	LocationID string
	Quantity   string
}

func PostIssueMovement(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	actorID string,
	reference string,
	notes string,
	occurredAt time.Time,
	inputLines []PostIssueMovementLine,
) (StockMovement, error) {
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	var movement StockMovement
	if err := tx.QueryRow(ctx, `
INSERT INTO inventory.stock_movements (
    tenant_id, movement_number, movement_type, reference, notes,
    occurred_at, status, created_by
)
VALUES (
    $1, 'SM-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 12)),
    'issue', $2, $3, $4, 'posted', $5::uuid
)
RETURNING id::text, movement_number, movement_type, reference, notes,
          occurred_at, status, created_at
`, tenantID, strings.TrimSpace(reference), strings.TrimSpace(notes), occurredAt, actorID).Scan(
		&movement.ID, &movement.MovementNumber, &movement.MovementType,
		&movement.Reference, &movement.Notes, &movement.OccurredAt,
		&movement.Status, &movement.CreatedAt,
	); err != nil {
		return StockMovement{}, fmt.Errorf("insert issue stock movement: %w", err)
	}

	movement.Lines = make([]StockMovementLine, 0, len(inputLines))
	for _, inputLine := range inputLines {
		lockKey := tenantID + ":" + inputLine.ItemID + ":" + inputLine.LocationID
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
			return StockMovement{}, fmt.Errorf("lock issue stock balance: %w", err)
		}

		var line StockMovementLine
		if err := tx.QueryRow(ctx, `
SELECT i.id::text, i.sku, i.name, loc.id::text, loc.code, loc.name
FROM inventory.items i
JOIN inventory.locations loc ON loc.tenant_id = i.tenant_id
WHERE i.id = $1::uuid
  AND i.status = 'active'
  AND loc.id = $2::uuid
  AND loc.status = 'active'
`, inputLine.ItemID, inputLine.LocationID).Scan(
			&line.ItemID, &line.ItemSKU, &line.ItemName,
			&line.LocationID, &line.LocationCode, &line.LocationName,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return StockMovement{}, ErrIssueItemOrLocationNotFound
			}
			return StockMovement{}, fmt.Errorf("load issue stock references: %w", err)
		}

		var available bool
		if err := tx.QueryRow(ctx, `
SELECT $3::numeric <= (
  COALESCE((SELECT SUM(quantity_delta) FROM inventory.stock_movement_lines WHERE item_id=$1::uuid AND location_id=$2::uuid), 0)
  - COALESCE((SELECT SUM(quantity) FROM inventory.reservations WHERE item_id=$1::uuid AND location_id=$2::uuid AND status='active' AND (expires_at IS NULL OR expires_at > now())), 0)
)
`, inputLine.ItemID, inputLine.LocationID, inputLine.Quantity).Scan(&available); err != nil {
			return StockMovement{}, fmt.Errorf("check issue stock availability: %w", err)
		}
		if !available {
			return StockMovement{}, ErrIssueInsufficientStock
		}

		if err := tx.QueryRow(ctx, `
INSERT INTO inventory.stock_movement_lines (
    tenant_id, movement_id, item_id, location_id, quantity_delta
)
VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5::numeric * -1)
RETURNING id::text, quantity_delta::text
`, tenantID, movement.ID, line.ItemID, line.LocationID, inputLine.Quantity).Scan(
			&line.ID, &line.QuantityDelta,
		); err != nil {
			return StockMovement{}, fmt.Errorf("insert issue stock movement line: %w", err)
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
