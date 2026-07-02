CREATE TABLE IF NOT EXISTS inventory.locations (
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

CREATE INDEX IF NOT EXISTS idx_inventory_locations_tenant_status
ON inventory.locations (tenant_id, status);

ALTER TABLE inventory.locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.locations FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_inventory_locations ON inventory.locations;

CREATE POLICY tenant_isolation_inventory_locations
ON inventory.locations
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('inventory.location.read', 'inventory', 'Read inventory locations'),
    ('inventory.location.write', 'inventory', 'Create and update inventory locations')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON inventory.locations TO erp_app;
