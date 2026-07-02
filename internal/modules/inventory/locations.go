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

type Location struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateLocationInput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (s Store) ListLocations(ctx context.Context, tenantID string) ([]Location, error) {
	locations := make([]Location, 0)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id::text, code, name, description, status, created_at, updated_at
FROM inventory.locations
ORDER BY code ASC, created_at DESC, id DESC
`)
		if err != nil {
			return fmt.Errorf("query inventory locations: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var location Location
			if err := rows.Scan(
				&location.ID,
				&location.Code,
				&location.Name,
				&location.Description,
				&location.Status,
				&location.CreatedAt,
				&location.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan inventory location: %w", err)
			}

			locations = append(locations, location)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate inventory locations: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return locations, nil
}

func (s Store) CreateLocation(ctx context.Context, tenantID string, actorID string, input CreateLocationInput) (Location, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.TrimSpace(input.Status)

	if input.Status == "" {
		input.Status = "active"
	}

	var location Location

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
INSERT INTO inventory.locations (
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
			&location.ID,
			&location.Code,
			&location.Name,
			&location.Description,
			&location.Status,
			&location.CreatedAt,
			&location.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert inventory location: %w", err)
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID:   tenantID,
			ActorType:  "tenant_user",
			ActorID:    actorID,
			Action:     "inventory.location.create",
			TargetType: "inventory.location",
			TargetID:   location.ID,
			Metadata: map[string]any{
				"code":   location.Code,
				"name":   location.Name,
				"status": location.Status,
			},
		}); err != nil {
			return err
		}

		if err := outbox.Insert(ctx, tx, outbox.Event{
			TenantID:      tenantID,
			EventType:     "inventory.location.created.v1",
			EventVersion:  1,
			AggregateType: "inventory.location",
			AggregateID:   location.ID,
			Payload: map[string]any{
				"location_id": location.ID,
				"code":        location.Code,
				"name":        location.Name,
				"status":      location.Status,
			},
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return Location{}, err
	}

	return location, nil
}
