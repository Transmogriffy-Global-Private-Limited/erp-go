package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantSessionStore struct {
	db *pgxpool.Pool
}

type TenantSession struct {
	Token     string    `json:"session_token"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

type TenantSessionUser struct {
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

var ErrInvalidTenantLogin = errors.New("invalid tenant login")

func NewTenantSessionStore(db *pgxpool.Pool) TenantSessionStore {
	return TenantSessionStore{
		db: db,
	}
}

func (s TenantSessionStore) Login(ctx context.Context, tenantID string, email string, password string, ttl time.Duration) (TenantSession, error) {
	var session TenantSession

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var userID string
		var status string

		err := tx.QueryRow(ctx, `
SELECT id::text, status
FROM core.tenant_users
WHERE tenant_id = $1
  AND email = $2
  AND password_hash IS NOT NULL
  AND password_hash = crypt($3, password_hash)
`, tenantID, email, password).Scan(&userID, &status)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidTenantLogin
			}

			return fmt.Errorf("query tenant login: %w", err)
		}

		if status != "active" {
			return ErrInvalidTenantLogin
		}

		token, err := randomTenantToken()
		if err != nil {
			return err
		}

		expiresAt := time.Now().UTC().Add(ttl)
		tokenHash := hashTenantToken(token)

		_, err = tx.Exec(ctx, `
INSERT INTO core.tenant_user_sessions (
tenant_id,
user_id,
token_hash,
expires_at
)
VALUES (
$1,
$2::uuid,
$3,
$4
)
`, tenantID, userID, tokenHash, expiresAt)
		if err != nil {
			return fmt.Errorf("insert tenant user session: %w", err)
		}

		session = TenantSession{
			Token:     token,
			TenantID:  tenantID,
			UserID:    userID,
			Email:     email,
			ExpiresAt: expiresAt,
		}

		return nil
	})
	if err != nil {
		return TenantSession{}, err
	}

	return session, nil
}

func (s TenantSessionStore) UserFromSession(ctx context.Context, tenantID string, token string) (TenantSessionUser, bool, error) {
	if token == "" {
		return TenantSessionUser{}, false, nil
	}

	var user TenantSessionUser
	tokenHash := hashTenantToken(token)

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
UPDATE core.tenant_user_sessions s
SET last_seen_at = now()
FROM core.tenant_users u
WHERE s.tenant_id = u.tenant_id
  AND s.user_id = u.id
  AND s.tenant_id = $1
  AND s.token_hash = $2
  AND s.status = 'active'
  AND s.revoked_at IS NULL
  AND s.expires_at > now()
  AND u.status = 'active'
RETURNING
  u.tenant_id::text,
  u.id::text,
  u.email,
  u.display_name,
  u.status
`, tenantID, tokenHash).Scan(
			&user.TenantID,
			&user.UserID,
			&user.Email,
			&user.DisplayName,
			&user.Status,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrTenantSessionNotFound
			}

			return fmt.Errorf("check tenant user session: %w", err)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, ErrTenantSessionNotFound) {
			return TenantSessionUser{}, false, nil
		}

		return TenantSessionUser{}, false, err
	}

	return user, true, nil
}

func (s TenantSessionStore) RevokeSession(ctx context.Context, tenantID string, token string) (bool, error) {
	if token == "" {
		return false, nil
	}

	tokenHash := hashTenantToken(token)
	revoked := false

	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
UPDATE core.tenant_user_sessions
SET status = 'revoked',
    revoked_at = now()
WHERE tenant_id = $1
  AND token_hash = $2
  AND status = 'active'
  AND revoked_at IS NULL
`, tenantID, tokenHash)
		if err != nil {
			return fmt.Errorf("revoke tenant user session: %w", err)
		}

		revoked = tag.RowsAffected() > 0
		return nil
	})
	if err != nil {
		return false, err
	}

	return revoked, nil
}

var ErrTenantSessionNotFound = errors.New("tenant session not found")

func randomTenantToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate tenant session token: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

func hashTenantToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
