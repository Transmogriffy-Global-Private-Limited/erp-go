package auth

import (
	"context"
	"fmt"

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

func (s Store) IsActiveSuperadmin(ctx context.Context, platformUserID string) (bool, error) {
	var allowed bool

	err := s.db.QueryRow(ctx, `
SELECT EXISTS (
SELECT 1
FROM control.platform_users
WHERE id = $1
  AND status = 'active'
  AND role = 'superadmin'
)
`, platformUserID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check platform superadmin: %w", err)
	}

	return allowed, nil
}
