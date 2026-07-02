package inventory

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

type Item struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	BaseUnitID  string    `json:"base_unit_id"`
	BaseUnit    *UnitRef  `json:"base_unit,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UnitRef struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

var ErrBaseUnitNotFound = errors.New("base unit not found or inactive")

type CreateItemInput struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	BaseUnitID  string `json:"base_unit_id"`
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return Store{
		db: db,
	}
}

func (s Store) ListItems(ctx context.Context, tenantID string) ([]Item, error) {
	items := make([]Item, 0)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT
    i.id::text,
    i.sku,
    i.name,
    i.description,
    i.status,
    COALESCE(i.base_unit_id::text, ''),
    COALESCE(u.id::text, ''),
    COALESCE(u.code, ''),
    COALESCE(u.name, ''),
    i.created_at,
    i.updated_at
FROM inventory.items i
LEFT JOIN inventory.units u
    ON u.tenant_id = i.tenant_id
   AND u.id = i.base_unit_id
ORDER BY i.created_at DESC, i.id DESC
`)
		if err != nil {
			return fmt.Errorf("query inventory items: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item Item
			var unit UnitRef
			if err := rows.Scan(
				&item.ID,
				&item.SKU,
				&item.Name,
				&item.Description,
				&item.Status,
				&item.BaseUnitID,
				&unit.ID,
				&unit.Code,
				&unit.Name,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan inventory item: %w", err)
			}

			if unit.ID != "" {
				item.BaseUnit = &unit
			}

			items = append(items, item)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate inventory items: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s Store) CreateItem(ctx context.Context, tenantID string, actorID string, input CreateItemInput) (Item, error) {
	input.SKU = strings.TrimSpace(input.SKU)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.TrimSpace(input.Status)
	input.BaseUnitID = strings.TrimSpace(input.BaseUnitID)

	if input.Status == "" {
		input.Status = "active"
	}

	var item Item

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var unit UnitRef
		if err := tx.QueryRow(ctx, `
SELECT id::text, code, name
FROM inventory.units
WHERE tenant_id = $1::uuid
  AND id = $2::uuid
  AND status = 'active'
`, tenantID, input.BaseUnitID).Scan(
			&unit.ID,
			&unit.Code,
			&unit.Name,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrBaseUnitNotFound
			}

			return fmt.Errorf("query inventory base unit: %w", err)
		}

		err := tx.QueryRow(ctx, `
INSERT INTO inventory.items (
tenant_id,
sku,
name,
description,
status,
base_unit_id,
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
$7::uuid,
$7::uuid
)
RETURNING id::text, sku, name, description, status, base_unit_id::text, created_at, updated_at
`, tenantID, input.SKU, input.Name, input.Description, input.Status, input.BaseUnitID, actorID).Scan(
			&item.ID,
			&item.SKU,
			&item.Name,
			&item.Description,
			&item.Status,
			&item.BaseUnitID,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert inventory item: %w", err)
		}

		item.BaseUnit = &unit

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID:   tenantID,
			ActorType:  "tenant_user",
			ActorID:    actorID,
			Action:     "inventory.item.create",
			TargetType: "inventory.item",
			TargetID:   item.ID,
			Metadata: map[string]any{
				"sku":          item.SKU,
				"name":         item.Name,
				"status":       item.Status,
				"base_unit_id": item.BaseUnitID,
				"base_unit":    unit.Code,
			},
		}); err != nil {
			return err
		}

		if err := outbox.Insert(ctx, tx, outbox.Event{
			TenantID:      tenantID,
			EventType:     "inventory.item.created.v1",
			EventVersion:  1,
			AggregateType: "inventory.item",
			AggregateID:   item.ID,
			Payload: map[string]any{
				"item_id":      item.ID,
				"sku":          item.SKU,
				"name":         item.Name,
				"status":       item.Status,
				"base_unit_id": item.BaseUnitID,
				"base_unit":    unit.Code,
			},
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return Item{}, err
	}

	return item, nil
}
