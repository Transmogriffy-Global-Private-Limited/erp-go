CREATE TABLE IF NOT EXISTS control.platform_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    tenant_id UUID REFERENCES control.tenants(id) ON DELETE SET NULL,
    reason TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_platform_audit_log_created
ON control.platform_audit_log (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_platform_audit_log_tenant_created
ON control.platform_audit_log (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_platform_audit_log_action_created
ON control.platform_audit_log (action, created_at DESC);

GRANT SELECT, INSERT, UPDATE, DELETE ON control.platform_audit_log TO erp_app;
