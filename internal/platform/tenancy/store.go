package tenancy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Tenant struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	LegalName   string    `json:"legal_name"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTenantInput struct {
	Slug        string `json:"slug"`
	LegalName   string `json:"legal_name"`
	DisplayName string `json:"display_name"`
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

func (s Store) ListTenants(ctx context.Context) ([]Tenant, error) {
	rows, err := s.db.Query(ctx, `
SELECT id::text, slug, legal_name, display_name, status, created_at, updated_at
FROM control.tenants
ORDER BY created_at DESC, id DESC
`)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()

	tenants := make([]Tenant, 0)

	for rows.Next() {
		var tenant Tenant
		if err := rows.Scan(
			&tenant.ID,
			&tenant.Slug,
			&tenant.LegalName,
			&tenant.DisplayName,
			&tenant.Status,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}

		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenants: %w", err)
	}

	return tenants, nil
}

func (s Store) CreateTenant(ctx context.Context, input CreateTenantInput) (Tenant, error) {
	input.Slug = strings.TrimSpace(input.Slug)
	input.LegalName = strings.TrimSpace(input.LegalName)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Status = strings.TrimSpace(input.Status)

	if input.Status == "" {
		input.Status = "trial"
	}

	var tenant Tenant

	err := s.db.QueryRow(ctx, `
INSERT INTO control.tenants (
slug,
legal_name,
display_name,
status
)
VALUES (
$1,
$2,
$3,
$4
)
RETURNING id::text, slug, legal_name, display_name, status, created_at, updated_at
`, input.Slug, input.LegalName, input.DisplayName, input.Status).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.LegalName,
		&tenant.DisplayName,
		&tenant.Status,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)
	if err != nil {
		return Tenant{}, fmt.Errorf("insert tenant: %w", err)
	}

	return tenant, nil
}
