CREATE TABLE IF NOT EXISTS inventory.units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_inventory_units_tenant_status
ON inventory.units (tenant_id, status);

ALTER TABLE inventory.units ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.units FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_inventory_units ON inventory.units;

CREATE POLICY tenant_isolation_inventory_units
ON inventory.units
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('inventory.unit.read', 'inventory', 'Read inventory units of measure'),
    ('inventory.unit.write', 'inventory', 'Create and update inventory units of measure')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON inventory.units TO erp_app;
