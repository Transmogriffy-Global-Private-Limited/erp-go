package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

type PlatformSession struct {
	Token          string    `json:"session_token"`
	PlatformUserID string    `json:"platform_user_id"`
	ExpiresAt      time.Time `json:"expires_at"`
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

func (s Store) LoginSuperadmin(ctx context.Context, email string, password string, ttl time.Duration) (PlatformSession, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return PlatformSession{}, fmt.Errorf("begin platform login transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var platformUserID string

	err = tx.QueryRow(ctx, `
SELECT id::text
FROM control.platform_users
WHERE email = $1
  AND password_hash IS NOT NULL
  AND password_hash = crypt($2, password_hash)
  AND status = 'active'
  AND role = 'superadmin'
`, email, password).Scan(&platformUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PlatformSession{}, ErrInvalidPlatformLogin
		}

		return PlatformSession{}, fmt.Errorf("query platform login: %w", err)
	}

	token, err := randomToken()
	if err != nil {
		return PlatformSession{}, err
	}

	tokenHash := hashToken(token)
	expiresAt := time.Now().UTC().Add(ttl)

	var sessionID string

	err = tx.QueryRow(ctx, `
INSERT INTO control.platform_sessions (
platform_user_id,
token_hash,
expires_at
)
VALUES (
$1,
$2,
$3
)
RETURNING id::text
`, platformUserID, tokenHash, expiresAt).Scan(&sessionID)
	if err != nil {
		return PlatformSession{}, fmt.Errorf("insert platform session: %w", err)
	}

	if err := audit.InsertPlatform(ctx, tx, audit.PlatformEntry{
		PlatformActorID: platformUserID,
		Action:          "control.platform_auth.login",
		TargetType:      "control.platform_session",
		TargetID:        sessionID,
		Metadata: map[string]any{
			"email":      email,
			"expires_at": expiresAt,
		},
	}); err != nil {
		return PlatformSession{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return PlatformSession{}, fmt.Errorf("commit platform login transaction: %w", err)
	}

	return PlatformSession{
		Token:          token,
		PlatformUserID: platformUserID,
		ExpiresAt:      expiresAt,
	}, nil
}

func (s Store) SuperadminFromSession(ctx context.Context, token string) (string, bool, error) {
	if token == "" {
		return "", false, nil
	}

	tokenHash := hashToken(token)

	var platformUserID string

	err := s.db.QueryRow(ctx, `
UPDATE control.platform_sessions s
SET last_seen_at = now()
FROM control.platform_users u
WHERE s.platform_user_id = u.id
  AND s.token_hash = $1
  AND s.status = 'active'
  AND s.revoked_at IS NULL
  AND s.expires_at > now()
  AND u.status = 'active'
  AND u.role = 'superadmin'
RETURNING u.id::text
`, tokenHash).Scan(&platformUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("check platform session: %w", err)
	}

	return platformUserID, true, nil
}

func (s Store) RevokeSession(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}

	tokenHash := hashToken(token)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin platform logout transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var sessionID string
	var platformUserID string

	err = tx.QueryRow(ctx, `
UPDATE control.platform_sessions
SET status = 'revoked',
    revoked_at = now()
WHERE token_hash = $1
  AND status = 'active'
  AND revoked_at IS NULL
RETURNING id::text, platform_user_id::text
`, tokenHash).Scan(&sessionID, &platformUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("revoke platform session: %w", err)
	}

	if err := audit.InsertPlatform(ctx, tx, audit.PlatformEntry{
		PlatformActorID: platformUserID,
		Action:          "control.platform_auth.logout",
		TargetType:      "control.platform_session",
		TargetID:        sessionID,
		Metadata: map[string]any{
			"revoked": true,
		},
	}); err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit platform logout transaction: %w", err)
	}

	return true, nil
}

var ErrInvalidPlatformLogin = errors.New("invalid platform login")

func randomToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate platform session token: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
