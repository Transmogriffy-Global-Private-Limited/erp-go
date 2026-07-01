CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE core.tenant_users
ADD COLUMN IF NOT EXISTS password_hash TEXT;

CREATE TABLE IF NOT EXISTS core.tenant_user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'revoked')),
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_tenant_user_sessions_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES core.tenant_users (tenant_id, id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tenant_user_sessions_user_created
ON core.tenant_user_sessions (tenant_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tenant_user_sessions_active_expiry
ON core.tenant_user_sessions (status, expires_at);

ALTER TABLE core.tenant_user_sessions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_user_sessions_isolation
ON core.tenant_user_sessions
USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON core.tenant_user_sessions TO erp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON core.tenant_users TO erp_app;
