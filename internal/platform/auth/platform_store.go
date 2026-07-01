package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	platformLoginLockThreshold = 5
	platformLoginLockDuration  = 15 * time.Minute
	unknownPlatformActorID     = "unknown"
)

type Store struct {
	db *pgxpool.Pool
}

type PlatformSession struct {
	Token          string    `json:"session_token"`
	PlatformUserID string    `json:"platform_user_id"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type PlatformSessionRecord struct {
	ID             string     `json:"id"`
	PlatformUserID string     `json:"platform_user_id"`
	Status         string     `json:"status"`
	ExpiresAt      time.Time  `json:"expires_at"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
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
	var status string
	var role string
	var passwordOK bool
	var lockedUntil sql.NullTime

	err = tx.QueryRow(ctx, `
SELECT
id::text,
status,
role,
(password_hash IS NOT NULL AND password_hash = crypt($2, password_hash)) AS password_ok,
locked_until
FROM control.platform_users
WHERE email = $1
`, email, password).Scan(
		&platformUserID,
		&status,
		&role,
		&passwordOK,
		&lockedUntil,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if auditErr := audit.InsertPlatform(ctx, tx, audit.PlatformEntry{
				PlatformActorID: unknownPlatformActorID,
				Action:          "control.platform_auth.login_failed",
				TargetType:      "control.platform_user",
				TargetID:        email,
				Reason:          "unknown_email",
				Metadata: map[string]any{
					"email": email,
				},
			}); auditErr != nil {
				return PlatformSession{}, auditErr
			}

			if commitErr := tx.Commit(ctx); commitErr != nil {
				return PlatformSession{}, fmt.Errorf("commit unknown platform login failure audit: %w", commitErr)
			}

			return PlatformSession{}, ErrInvalidPlatformLogin
		}

		return PlatformSession{}, fmt.Errorf("query platform login: %w", err)
	}

	if lockedUntil.Valid && time.Now().UTC().Before(lockedUntil.Time.UTC()) {
		if auditErr := audit.InsertPlatform(ctx, tx, audit.PlatformEntry{
			PlatformActorID: platformUserID,
			Action:          "control.platform_auth.login_failed",
			TargetType:      "control.platform_user",
			TargetID:        platformUserID,
			Reason:          "locked",
			Metadata: map[string]any{
				"email":        email,
				"locked_until": lockedUntil.Time.UTC(),
			},
		}); auditErr != nil {
			return PlatformSession{}, auditErr
		}

		if commitErr := tx.Commit(ctx); commitErr != nil {
			return PlatformSession{}, fmt.Errorf("commit locked platform login audit: %w", commitErr)
		}

		return PlatformSession{}, ErrPlatformLoginLocked
	}

	if status != "active" || role != "superadmin" || !passwordOK {
		var failedLoginCount int
		var newLockedUntil sql.NullTime

		err = tx.QueryRow(ctx, `
UPDATE control.platform_users
SET failed_login_count = failed_login_count + 1,
    last_failed_login_at = now(),
    locked_until = CASE
        WHEN failed_login_count + 1 >= $2 THEN now() + ($3::text)::interval
        ELSE locked_until
    END,
    updated_at = now()
WHERE id = $1
RETURNING failed_login_count, locked_until
`, platformUserID, platformLoginLockThreshold, platformLoginLockDuration.String()).Scan(
			&failedLoginCount,
			&newLockedUntil,
		)
		if err != nil {
			return PlatformSession{}, fmt.Errorf("record failed platform login: %w", err)
		}

		reason := "invalid_credentials"
		if status != "active" {
			reason = "inactive_user"
		} else if role != "superadmin" {
			reason = "not_superadmin"
		}

		if auditErr := audit.InsertPlatform(ctx, tx, audit.PlatformEntry{
			PlatformActorID: platformUserID,
			Action:          "control.platform_auth.login_failed",
			TargetType:      "control.platform_user",
			TargetID:        platformUserID,
			Reason:          reason,
			Metadata: map[string]any{
				"email":              email,
				"failed_login_count": failedLoginCount,
				"locked":             newLockedUntil.Valid,
			},
		}); auditErr != nil {
			return PlatformSession{}, auditErr
		}

		if commitErr := tx.Commit(ctx); commitErr != nil {
			return PlatformSession{}, fmt.Errorf("commit platform login failure transaction: %w", commitErr)
		}

		if newLockedUntil.Valid && failedLoginCount >= platformLoginLockThreshold {
			return PlatformSession{}, ErrPlatformLoginLocked
		}

		return PlatformSession{}, ErrInvalidPlatformLogin
	}

	token, err := randomToken()
	if err != nil {
		return PlatformSession{}, err
	}

	tokenHash := hashToken(token)
	expiresAt := time.Now().UTC().Add(ttl)

	_, err = tx.Exec(ctx, `
UPDATE control.platform_users
SET failed_login_count = 0,
    last_failed_login_at = NULL,
    locked_until = NULL,
    updated_at = now()
WHERE id = $1
`, platformUserID)
	if err != nil {
		return PlatformSession{}, fmt.Errorf("reset platform login failure state: %w", err)
	}

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

func (s Store) ListActiveSessions(ctx context.Context, platformUserID string) ([]PlatformSessionRecord, error) {
	rows, err := s.db.Query(ctx, `
SELECT
id::text,
platform_user_id::text,
status,
expires_at,
last_seen_at,
revoked_at,
created_at
FROM control.platform_sessions
WHERE platform_user_id = $1
  AND status = 'active'
  AND revoked_at IS NULL
  AND expires_at > now()
ORDER BY created_at DESC, id DESC
`, platformUserID)
	if err != nil {
		return nil, fmt.Errorf("query active platform sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]PlatformSessionRecord, 0)

	for rows.Next() {
		var session PlatformSessionRecord
		var lastSeenAt sql.NullTime
		var revokedAt sql.NullTime

		if err := rows.Scan(
			&session.ID,
			&session.PlatformUserID,
			&session.Status,
			&session.ExpiresAt,
			&lastSeenAt,
			&revokedAt,
			&session.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan platform session: %w", err)
		}

		if lastSeenAt.Valid {
			value := lastSeenAt.Time
			session.LastSeenAt = &value
		}

		if revokedAt.Valid {
			value := revokedAt.Time
			session.RevokedAt = &value
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate platform sessions: %w", err)
	}

	return sessions, nil
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

func (s Store) RevokeAllSessions(ctx context.Context, platformUserID string) (int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin platform revoke-all transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(ctx, `
UPDATE control.platform_sessions
SET status = 'revoked',
    revoked_at = now()
WHERE platform_user_id = $1
  AND status = 'active'
  AND revoked_at IS NULL
`, platformUserID)
	if err != nil {
		return 0, fmt.Errorf("revoke all platform sessions: %w", err)
	}

	revokedCount := tag.RowsAffected()

	if err := audit.InsertPlatform(ctx, tx, audit.PlatformEntry{
		PlatformActorID: platformUserID,
		Action:          "control.platform_auth.revoke_all_sessions",
		TargetType:      "control.platform_user",
		TargetID:        platformUserID,
		Metadata: map[string]any{
			"revoked_count": revokedCount,
		},
	}); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit platform revoke-all transaction: %w", err)
	}

	return revokedCount, nil
}

func (s Store) CleanupExpiredSessions(ctx context.Context, platformActorID string) (int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin platform session cleanup transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(ctx, `
UPDATE control.platform_sessions
SET status = 'revoked',
    revoked_at = now()
WHERE status = 'active'
  AND revoked_at IS NULL
  AND expires_at <= now()
`)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired platform sessions: %w", err)
	}

	revokedCount := tag.RowsAffected()

	if err := audit.InsertPlatform(ctx, tx, audit.PlatformEntry{
		PlatformActorID: platformActorID,
		Action:          "control.platform_auth.cleanup_expired_sessions",
		TargetType:      "control.platform_sessions",
		TargetID:        "expired",
		Metadata: map[string]any{
			"revoked_count": revokedCount,
		},
	}); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit platform session cleanup transaction: %w", err)
	}

	return revokedCount, nil
}

var ErrInvalidPlatformLogin = errors.New("invalid platform login")
var ErrPlatformLoginLocked = errors.New("platform login locked")

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
