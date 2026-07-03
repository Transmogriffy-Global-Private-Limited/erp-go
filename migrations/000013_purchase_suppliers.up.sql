CREATE TABLE IF NOT EXISTS purchase.suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_purchase_suppliers_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_purchase_suppliers_tenant_code UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_purchase_suppliers_tenant_status
ON purchase.suppliers (tenant_id, status);

ALTER TABLE purchase.suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase.suppliers FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_purchase_suppliers ON purchase.suppliers;

CREATE POLICY tenant_isolation_purchase_suppliers
ON purchase.suppliers
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('purchase.supplier.read', 'purchase', 'Read purchase suppliers'),
    ('purchase.supplier.write', 'purchase', 'Create and update purchase suppliers')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON purchase.suppliers TO erp_app;
