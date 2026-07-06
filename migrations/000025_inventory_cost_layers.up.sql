CREATE TABLE inventory.cost_layers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    item_id UUID NOT NULL,
    unit_cost NUMERIC(18, 4) NOT NULL CHECK (unit_cost >= 0),
    currency_code TEXT NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
    source TEXT NOT NULL DEFAULT 'manual',
    effective_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_inventory_cost_layers_current
ON inventory.cost_layers (tenant_id, item_id, effective_at DESC, created_at DESC);

ALTER TABLE inventory.cost_layers ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.cost_layers FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_inventory_cost_layers ON inventory.cost_layers
USING (tenant_id = core.current_tenant_id()) WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description) VALUES
    ('inventory.cost.read', 'inventory', 'Read inventory costs and valuation'),
    ('inventory.cost.write', 'inventory', 'Append inventory cost layers'),
    ('inventory.report.read', 'inventory', 'Read inventory operational reports')
ON CONFLICT (id) DO UPDATE SET module_id = EXCLUDED.module_id, description = EXCLUDED.description;

GRANT SELECT, INSERT ON inventory.cost_layers TO erp_app;
