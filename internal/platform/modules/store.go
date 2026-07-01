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

// IsEnabledForTenant reports whether a module is enabled for a tenant.
func (s Store) IsEnabledForTenant(ctx context.Context, tenantID string, moduleID string) (bool, error) {
	var enabled bool

	err := s.db.QueryRow(ctx, `
SELECT EXISTS (
SELECT 1
FROM control.tenant_enabled_modules tem
INNER JOIN control.modules m ON m.id = tem.module_id
WHERE tem.tenant_id = $1
  AND tem.module_id = $2
  AND tem.disabled_at IS NULL
  AND m.status <> 'disabled'
)
`, tenantID, moduleID).Scan(&enabled)
	if err != nil {
		return false, fmt.Errorf("check tenant module entitlement: %w", err)
	}

	return enabled, nil
}

// EnableForTenant enables a module for a tenant and returns the module record.
func (s Store) EnableForTenant(ctx context.Context, tenantID string, moduleID string) (Module, error) {
	var module Module

	err := s.db.QueryRow(ctx, `
WITH enabled AS (
INSERT INTO control.tenant_enabled_modules (
tenant_id,
module_id,
enabled_at,
disabled_at
)
VALUES (
$1,
$2,
now(),
NULL
)
ON CONFLICT (tenant_id, module_id) DO UPDATE
SET enabled_at = now(),
    disabled_at = NULL
RETURNING module_id
)
SELECT m.id, m.name, m.status
FROM enabled e
INNER JOIN control.modules m ON m.id = e.module_id
`, tenantID, moduleID).Scan(&module.ID, &module.Name, &module.Status)
	if err != nil {
		return Module{}, fmt.Errorf("enable tenant module: %w", err)
	}

	return module, nil
}

// DisableForTenant disables a module for a tenant and returns the module record.
func (s Store) DisableForTenant(ctx context.Context, tenantID string, moduleID string) (Module, error) {
	var module Module

	err := s.db.QueryRow(ctx, `
WITH disabled AS (
UPDATE control.tenant_enabled_modules
SET disabled_at = now()
WHERE tenant_id = $1
  AND module_id = $2
RETURNING module_id
)
SELECT m.id, m.name, m.status
FROM disabled d
INNER JOIN control.modules m ON m.id = d.module_id
`, tenantID, moduleID).Scan(&module.ID, &module.Name, &module.Status)
	if err != nil {
		return Module{}, fmt.Errorf("disable tenant module: %w", err)
	}

	return module, nil
}
