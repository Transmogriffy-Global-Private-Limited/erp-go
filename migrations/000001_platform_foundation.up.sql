CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS control;
CREATE SCHEMA IF NOT EXISTS core;
CREATE SCHEMA IF NOT EXISTS documents;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS workflow;
CREATE SCHEMA IF NOT EXISTS inventory;
CREATE SCHEMA IF NOT EXISTS sales;
CREATE SCHEMA IF NOT EXISTS purchase;
CREATE SCHEMA IF NOT EXISTS accounting;
CREATE SCHEMA IF NOT EXISTS reports;

CREATE OR REPLACE FUNCTION core.current_tenant_id()
RETURNS UUID
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    tenant_setting TEXT;
BEGIN
    tenant_setting := current_setting('app.tenant_id', true);

    IF tenant_setting IS NULL OR tenant_setting = '' THEN
        RETURN NULL;
    END IF;

    RETURN tenant_setting::UUID;
END;
$$;

CREATE TABLE IF NOT EXISTS control.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    legal_name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('trial', 'active', 'suspended', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS control.modules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('planned', 'active', 'deprecated', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS control.plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS control.plan_modules (
    plan_id TEXT NOT NULL REFERENCES control.plans(id) ON DELETE CASCADE,
    module_id TEXT NOT NULL REFERENCES control.modules(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (plan_id, module_id)
);

CREATE TABLE IF NOT EXISTS control.tenant_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL REFERENCES control.plans(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('trialing', 'active', 'past_due', 'suspended', 'cancelled')),
    starts_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS control.tenant_enabled_modules (
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    module_id TEXT NOT NULL REFERENCES control.modules(id) ON DELETE RESTRICT,
    enabled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    disabled_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, module_id)
);

CREATE TABLE IF NOT EXISTS control.tenant_limits (
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    limit_key TEXT NOT NULL,
    limit_value BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, limit_key)
);

CREATE TABLE IF NOT EXISTS core.tenant_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('invited', 'active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, email)
);

CREATE TABLE IF NOT EXISTS core.roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS core.permissions (
    id TEXT PRIMARY KEY,
    module_id TEXT NOT NULL REFERENCES control.modules(id) ON DELETE RESTRICT,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS core.role_permissions (
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    role_id UUID NOT NULL,
    permission_id TEXT NOT NULL REFERENCES core.permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, role_id, permission_id),
    FOREIGN KEY (tenant_id, role_id) REFERENCES core.roles(tenant_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS core.user_roles (
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id, role_id),
    FOREIGN KEY (tenant_id, user_id) REFERENCES core.tenant_users(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, role_id) REFERENCES core.roles(tenant_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS core.outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES control.tenants(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    event_version INT NOT NULL,
    aggregate_type TEXT NOT NULL DEFAULT '',
    aggregate_id TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'publishing', 'published', 'failed')),
    attempt_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_pending
ON core.outbox_events (status, created_at)
WHERE status IN ('pending', 'failed');

CREATE TABLE IF NOT EXISTS documents.files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    module_id TEXT NOT NULL REFERENCES control.modules(id) ON DELETE RESTRICT,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    object_key TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    sha256_hash TEXT NOT NULL DEFAULT '',
    encryption_key_id TEXT NOT NULL DEFAULT '',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_documents_files_tenant_entity
ON documents.files (tenant_id, module_id, entity_type, entity_id);

CREATE TABLE IF NOT EXISTS audit.audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES control.tenants(id) ON DELETE CASCADE,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_tenant_created
ON audit.audit_log (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS inventory.items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    sku TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('active', 'inactive', 'archived')),
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, sku)
);

CREATE INDEX IF NOT EXISTS idx_inventory_items_tenant_status
ON inventory.items (tenant_id, status);

INSERT INTO control.modules (id, name, status)
VALUES
    ('inventory', 'Inventory', 'planned'),
    ('sales', 'Sales', 'planned'),
    ('purchase', 'Purchase', 'planned'),
    ('accounting', 'Accounting', 'planned')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    status = EXCLUDED.status;

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('inventory.item.read', 'inventory', 'Read inventory items'),
    ('inventory.item.write', 'inventory', 'Create and update inventory items'),
    ('inventory.stock.read', 'inventory', 'Read stock state'),
    ('inventory.stock.adjust', 'inventory', 'Adjust stock'),

    ('sales.order.read', 'sales', 'Read sales orders'),
    ('sales.order.create', 'sales', 'Create sales orders'),
    ('sales.order.approve', 'sales', 'Approve sales orders'),

    ('purchase.order.read', 'purchase', 'Read purchase orders'),
    ('purchase.order.create', 'purchase', 'Create purchase orders'),
    ('purchase.order.approve', 'purchase', 'Approve purchase orders'),

    ('accounting.ledger.read', 'accounting', 'Read ledgers'),
    ('accounting.journal.post', 'accounting', 'Post journals')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

ALTER TABLE core.tenant_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.tenant_users FORCE ROW LEVEL SECURITY;

ALTER TABLE core.roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.roles FORCE ROW LEVEL SECURITY;

ALTER TABLE core.role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.role_permissions FORCE ROW LEVEL SECURITY;

ALTER TABLE core.user_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.user_roles FORCE ROW LEVEL SECURITY;

ALTER TABLE documents.files ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents.files FORCE ROW LEVEL SECURITY;

ALTER TABLE audit.audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit.audit_log FORCE ROW LEVEL SECURITY;

ALTER TABLE inventory.items ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.items FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_core_tenant_users
ON core.tenant_users
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_core_roles
ON core.roles
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_core_role_permissions
ON core.role_permissions
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_core_user_roles
ON core.user_roles
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_documents_files
ON documents.files
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_audit_log
ON audit.audit_log
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_inventory_items
ON inventory.items
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());
