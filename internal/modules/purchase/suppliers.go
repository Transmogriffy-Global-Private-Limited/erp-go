package purchase

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
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrSupplierCodeExists = errors.New("purchase supplier code already exists")

type Supplier struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateSupplierInput struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Status  string `json:"status"`
}

func (s Store) ListSuppliers(ctx context.Context, tenantID string) ([]Supplier, error) {
	suppliers := make([]Supplier, 0)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id::text, code, name, email, phone, address, status, created_at, updated_at
FROM purchase.suppliers
ORDER BY code ASC, created_at DESC, id DESC
`)
		if err != nil {
			return fmt.Errorf("query purchase suppliers: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var supplier Supplier
			if err := rows.Scan(
				&supplier.ID,
				&supplier.Code,
				&supplier.Name,
				&supplier.Email,
				&supplier.Phone,
				&supplier.Address,
				&supplier.Status,
				&supplier.CreatedAt,
				&supplier.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan purchase supplier: %w", err)
			}

			suppliers = append(suppliers, supplier)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate purchase suppliers: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return suppliers, nil
}

func (s Store) CreateSupplier(ctx context.Context, tenantID string, actorID string, input CreateSupplierInput) (Supplier, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Address = strings.TrimSpace(input.Address)
	input.Status = strings.TrimSpace(input.Status)

	if input.Status == "" {
		input.Status = "active"
	}

	var supplier Supplier

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
INSERT INTO purchase.suppliers (
    tenant_id,
    code,
    name,
    email,
    phone,
    address,
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
    $6,
    $7,
    $8::uuid,
    $8::uuid
)
RETURNING id::text, code, name, email, phone, address, status, created_at, updated_at
`, tenantID, input.Code, input.Name, input.Email, input.Phone, input.Address, input.Status, actorID).Scan(
			&supplier.ID,
			&supplier.Code,
			&supplier.Name,
			&supplier.Email,
			&supplier.Phone,
			&supplier.Address,
			&supplier.Status,
			&supplier.CreatedAt,
			&supplier.UpdatedAt,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_purchase_suppliers_tenant_code" {
				return ErrSupplierCodeExists
			}

			return fmt.Errorf("insert purchase supplier: %w", err)
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID:   tenantID,
			ActorType:  "tenant_user",
			ActorID:    actorID,
			Action:     "purchase.supplier.create",
			TargetType: "purchase.supplier",
			TargetID:   supplier.ID,
			Metadata: map[string]any{
				"code":   supplier.Code,
				"name":   supplier.Name,
				"status": supplier.Status,
			},
		}); err != nil {
			return err
		}

		if err := outbox.Insert(ctx, tx, outbox.Event{
			TenantID:      tenantID,
			EventType:     "purchase.supplier.created.v1",
			EventVersion:  1,
			AggregateType: "purchase.supplier",
			AggregateID:   supplier.ID,
			Payload: map[string]any{
				"supplier_id": supplier.ID,
				"code":        supplier.Code,
				"name":        supplier.Name,
				"status":      supplier.Status,
			},
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return Supplier{}, err
	}

	return supplier, nil
}
