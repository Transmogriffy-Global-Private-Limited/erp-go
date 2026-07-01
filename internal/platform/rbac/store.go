package rbac

import (
	"context"
	"fmt"

	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return Store{
		db: db,
	}
}

// HasPermission reports whether a tenant user has a permission.
//
// This query touches tenant-owned RBAC tables and therefore must run inside
// a tenant-scoped transaction so PostgreSQL RLS sees app.tenant_id.
func (s Store) HasPermission(ctx context.Context, tenantID string, userID string, permissionID string) (bool, error) {
	var allowed bool

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
SELECT EXISTS (
SELECT 1
FROM core.user_roles ur
INNER JOIN core.role_permissions rp
ON rp.tenant_id = ur.tenant_id
   AND rp.role_id = ur.role_id
INNER JOIN core.tenant_users tu
ON tu.tenant_id = ur.tenant_id
   AND tu.id = ur.user_id
WHERE ur.tenant_id = $1
  AND ur.user_id = $2
  AND rp.permission_id = $3
  AND tu.status = 'active'
)
`, tenantID, userID, permissionID).Scan(&allowed)
		if err != nil {
			return fmt.Errorf("query permission: %w", err)
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return allowed, nil
}
