CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE control.platform_users
ADD COLUMN IF NOT EXISTS password_hash TEXT;

CREATE TABLE IF NOT EXISTS control.platform_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id UUID NOT NULL REFERENCES control.platform_users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'revoked')),
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_platform_sessions_user_created
ON control.platform_sessions (platform_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_platform_sessions_active_expiry
ON control.platform_sessions (status, expires_at);

GRANT SELECT, INSERT, UPDATE, DELETE ON control.platform_users TO erp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON control.platform_sessions TO erp_app;
