package modules

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module is the platform-visible module record.
type Module struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Store reads module registry and tenant module entitlement state.
type Store struct {
	db *pgxpool.Pool
}

// NewStore creates a module store.
func NewStore(db *pgxpool.Pool) Store {
	return Store{
		db: db,
	}
}

// ListAll returns every registered platform module.
func (s Store) ListAll(ctx context.Context) ([]Module, error) {
	rows, err := s.db.Query(ctx, `
SELECT id, name, status
FROM control.modules
ORDER BY id
`)
	if err != nil {
		return nil, fmt.Errorf("query platform modules: %w", err)
	}
	defer rows.Close()

	modules := make([]Module, 0)

	for rows.Next() {
		var module Module
		if err := rows.Scan(&module.ID, &module.Name, &module.Status); err != nil {
			return nil, fmt.Errorf("scan platform module: %w", err)
		}

		modules = append(modules, module)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate platform modules: %w", err)
	}

	return modules, nil
}

// ListEnabledForTenant returns modules enabled for a tenant.
func (s Store) ListEnabledForTenant(ctx context.Context, tenantID string) ([]Module, error) {
	rows, err := s.db.Query(ctx, `
SELECT m.id, m.name, m.status
FROM control.tenant_enabled_modules tem
INNER JOIN control.modules m ON m.id = tem.module_id
WHERE tem.tenant_id = $1
  AND tem.disabled_at IS NULL
ORDER BY m.id
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query enabled tenant modules: %w", err)
	}
	defer rows.Close()

	modules := make([]Module, 0)

	for rows.Next() {
		var module Module
		if err := rows.Scan(&module.ID, &module.Name, &module.Status); err != nil {
			return nil, fmt.Errorf("scan enabled tenant module: %w", err)
		}

		modules = append(modules, module)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled tenant modules: %w", err)
	}

	return modules, nil
}
