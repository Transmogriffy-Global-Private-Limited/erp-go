package sales

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
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCustomerCodeExists = errors.New("sales customer code already exists")

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return Store{db: db}
}

type Customer struct {
	ID              string    `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	BillingAddress  string    `json:"billing_address"`
	ShippingAddress string    `json:"shipping_address"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateCustomerInput struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	BillingAddress  string `json:"billing_address"`
	ShippingAddress string `json:"shipping_address"`
	Status          string `json:"status"`
}

func (s Store) ListCustomers(ctx context.Context, tenantID string) ([]Customer, error) {
	customers := make([]Customer, 0)
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id::text, code, name, email, phone, billing_address, shipping_address,
       status, created_at, updated_at
FROM sales.customers
ORDER BY code ASC, created_at DESC, id DESC
`)
		if err != nil {
			return fmt.Errorf("query sales customers: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var customer Customer
			if err := rows.Scan(
				&customer.ID,
				&customer.Code,
				&customer.Name,
				&customer.Email,
				&customer.Phone,
				&customer.BillingAddress,
				&customer.ShippingAddress,
				&customer.Status,
				&customer.CreatedAt,
				&customer.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan sales customer: %w", err)
			}
			customers = append(customers, customer)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return customers, nil
}

func (s Store) CreateCustomer(ctx context.Context, tenantID string, actorID string, input CreateCustomerInput) (Customer, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	input.BillingAddress = strings.TrimSpace(input.BillingAddress)
	input.ShippingAddress = strings.TrimSpace(input.ShippingAddress)
	input.Status = strings.TrimSpace(input.Status)
	if input.Status == "" {
		input.Status = "active"
	}

	var customer Customer
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
INSERT INTO sales.customers (
    tenant_id, code, name, email, phone, billing_address, shipping_address,
    status, created_by, updated_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::uuid, $9::uuid)
RETURNING id::text, code, name, email, phone, billing_address, shipping_address,
          status, created_at, updated_at
`, tenantID, input.Code, input.Name, input.Email, input.Phone, input.BillingAddress, input.ShippingAddress, input.Status, actorID).Scan(
			&customer.ID,
			&customer.Code,
			&customer.Name,
			&customer.Email,
			&customer.Phone,
			&customer.BillingAddress,
			&customer.ShippingAddress,
			&customer.Status,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_sales_customers_tenant_code" {
				return ErrCustomerCodeExists
			}
			return fmt.Errorf("insert sales customer: %w", err)
		}

		if err := audit.Insert(ctx, tx, audit.Entry{
			TenantID:   tenantID,
			ActorType:  "tenant_user",
			ActorID:    actorID,
			Action:     "sales.customer.create",
			TargetType: "sales.customer",
			TargetID:   customer.ID,
			Metadata: map[string]any{
				"code":   customer.Code,
				"name":   customer.Name,
				"status": customer.Status,
			},
		}); err != nil {
			return err
		}

		return outbox.Insert(ctx, tx, outbox.Event{
			TenantID:      tenantID,
			EventType:     "sales.customer.created.v1",
			EventVersion:  1,
			AggregateType: "sales.customer",
			AggregateID:   customer.ID,
			Payload: map[string]any{
				"customer_id": customer.ID,
				"code":        customer.Code,
				"name":        customer.Name,
				"status":      customer.Status,
			},
		})
	})
	if err != nil {
		return Customer{}, err
	}

	return customer, nil
}
