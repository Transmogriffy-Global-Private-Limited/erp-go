CREATE TABLE IF NOT EXISTS control.platform_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),
    role TEXT NOT NULL
        CHECK (role IN ('superadmin', 'support', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_platform_users_role_status
ON control.platform_users (role, status);

GRANT SELECT, INSERT, UPDATE, DELETE ON control.platform_users TO erp_app;
