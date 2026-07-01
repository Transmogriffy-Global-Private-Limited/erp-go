package inventory

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateItemInput struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
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
SELECT id::text, sku, name, description, status, created_at, updated_at
FROM inventory.items
ORDER BY created_at DESC, id DESC
`)
		if err != nil {
			return fmt.Errorf("query inventory items: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item Item
			if err := rows.Scan(
				&item.ID,
				&item.SKU,
				&item.Name,
				&item.Description,
				&item.Status,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan inventory item: %w", err)
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

func (s Store) CreateItem(ctx context.Context, tenantID string, input CreateItemInput) (Item, error) {
	input.SKU = strings.TrimSpace(input.SKU)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.TrimSpace(input.Status)

	if input.Status == "" {
		input.Status = "active"
	}

	var item Item

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
INSERT INTO inventory.items (
tenant_id,
sku,
name,
description,
status
)
VALUES (
$1,
$2,
$3,
$4,
$5
)
RETURNING id::text, sku, name, description, status, created_at, updated_at
`, tenantID, input.SKU, input.Name, input.Description, input.Status).Scan(
			&item.ID,
			&item.SKU,
			&item.Name,
			&item.Description,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert inventory item: %w", err)
		}

		return nil
	})
	if err != nil {
		return Item{}, err
	}

	return item, nil
}
