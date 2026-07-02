CREATE TABLE IF NOT EXISTS inventory.stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    movement_number TEXT NOT NULL,
    movement_type TEXT NOT NULL
        CHECK (movement_type IN ('adjustment', 'receipt', 'issue', 'transfer')),
    reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'posted'
        CHECK (status IN ('posted')),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, movement_number)
);

CREATE TABLE IF NOT EXISTS inventory.stock_movement_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    movement_id UUID NOT NULL,
    item_id UUID NOT NULL,
    location_id UUID NOT NULL,
    quantity_delta NUMERIC(18, 3) NOT NULL CHECK (quantity_delta <> 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_inventory_stock_lines_movement
        FOREIGN KEY (tenant_id, movement_id)
        REFERENCES inventory.stock_movements (tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_inventory_stock_lines_item
        FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_inventory_stock_lines_location
        FOREIGN KEY (tenant_id, location_id)
        REFERENCES inventory.locations (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_inventory_stock_movements_tenant_occurred
ON inventory.stock_movements (tenant_id, occurred_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_inventory_stock_lines_tenant_item_location
ON inventory.stock_movement_lines (tenant_id, item_id, location_id);

ALTER TABLE inventory.stock_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.stock_movements FORCE ROW LEVEL SECURITY;

ALTER TABLE inventory.stock_movement_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.stock_movement_lines FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_inventory_stock_movements ON inventory.stock_movements;
DROP POLICY IF EXISTS tenant_isolation_inventory_stock_movement_lines ON inventory.stock_movement_lines;

CREATE POLICY tenant_isolation_inventory_stock_movements
ON inventory.stock_movements
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_inventory_stock_movement_lines
ON inventory.stock_movement_lines
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('inventory.stock_movement.read', 'inventory', 'Read inventory stock movements'),
    ('inventory.stock_movement.write', 'inventory', 'Create inventory stock movements'),
    ('inventory.stock_balance.read', 'inventory', 'Read inventory stock balances')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON inventory.stock_movements TO erp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON inventory.stock_movement_lines TO erp_app;
