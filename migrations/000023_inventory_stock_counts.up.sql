CREATE TABLE inventory.stock_counts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    count_number TEXT NOT NULL,
    location_id UUID NOT NULL,
    reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    counted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'posted' CHECK (status = 'posted'),
    stock_movement_id UUID NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, count_number),
    FOREIGN KEY (tenant_id, location_id)
        REFERENCES inventory.locations (tenant_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, stock_movement_id)
        REFERENCES inventory.stock_movements (tenant_id, id) ON DELETE RESTRICT
);

CREATE TABLE inventory.stock_count_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    stock_count_id UUID NOT NULL,
    item_id UUID NOT NULL,
    system_quantity NUMERIC(18, 3) NOT NULL,
    counted_quantity NUMERIC(18, 3) NOT NULL CHECK (counted_quantity >= 0),
    variance_quantity NUMERIC(18, 3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, stock_count_id, item_id),
    FOREIGN KEY (tenant_id, stock_count_id)
        REFERENCES inventory.stock_counts (tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_inventory_stock_counts_tenant_date
ON inventory.stock_counts (tenant_id, counted_at DESC, id DESC);

ALTER TABLE inventory.stock_counts ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.stock_counts FORCE ROW LEVEL SECURITY;
ALTER TABLE inventory.stock_count_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.stock_count_lines FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_inventory_stock_counts ON inventory.stock_counts
USING (tenant_id = core.current_tenant_id()) WITH CHECK (tenant_id = core.current_tenant_id());
CREATE POLICY tenant_isolation_inventory_stock_count_lines ON inventory.stock_count_lines
USING (tenant_id = core.current_tenant_id()) WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description) VALUES
    ('inventory.stock_count.read', 'inventory', 'Read inventory stock counts'),
    ('inventory.stock_count.write', 'inventory', 'Post inventory stock counts')
ON CONFLICT (id) DO UPDATE SET module_id = EXCLUDED.module_id, description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON inventory.stock_counts, inventory.stock_count_lines TO erp_app;
