package inventory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

type Unit struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateUnitInput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (s Store) ListUnits(ctx context.Context, tenantID string) ([]Unit, error) {
	units := make([]Unit, 0)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id::text, code, name, description, status, created_at, updated_at
FROM inventory.units
ORDER BY code ASC, created_at DESC, id DESC
`)
		if err != nil {
			return fmt.Errorf("query inventory units: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var unit Unit
			if err := rows.Scan(
				&unit.ID,
				&unit.Code,
				&unit.Name,
				&unit.Description,
				&unit.Status,
				&unit.CreatedAt,
				&unit.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan inventory unit: %w", err)
			}

			units = append(units, unit)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate inventory units: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return units, nil
}

func (s Store) CreateUnit(ctx context.Context, tenantID string, actorID string, input CreateUnitInput) (Unit, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.TrimSpace(input.Status)

	if input.Status == "" {
		input.Status = "active"
	}

	var unit Unit

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
INSERT INTO inventory.units (
    tenant_id,
    code,
    name,
    description,
    status,
    created_by,
    updated_by
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6::uuid,
    $6::uuid
)
RETURNING id::text, code, name, description, status, created_at, updated_at
`, tenantID, input.Code, input.Name, input.Description, input.Status, actorID).Scan(
			&unit.ID,
			&unit.Code,
			&unit.Name,
			&unit.Description,
			&unit.Status,
			&unit.CreatedAt,
			&unit.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert inventory unit: %w", err)
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID:   tenantID,
			ActorType:  "tenant_user",
			ActorID:    actorID,
			Action:     "inventory.unit.create",
			TargetType: "inventory.unit",
			TargetID:   unit.ID,
			Metadata: map[string]any{
				"code":   unit.Code,
				"name":   unit.Name,
				"status": unit.Status,
			},
		}); err != nil {
			return err
		}

		if err := outbox.Insert(ctx, tx, outbox.Event{
			TenantID:      tenantID,
			EventType:     "inventory.unit.created.v1",
			EventVersion:  1,
			AggregateType: "inventory.unit",
			AggregateID:   unit.ID,
			Payload: map[string]any{
				"unit_id": unit.ID,
				"code":    unit.Code,
				"name":    unit.Name,
				"status":  unit.Status,
			},
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return Unit{}, err
	}

	return unit, nil
}
