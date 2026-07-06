CREATE TABLE inventory.transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    transfer_number TEXT NOT NULL,
    reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    transferred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'posted' CHECK (status = 'posted'),
    stock_movement_id UUID NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, transfer_number),
    FOREIGN KEY (tenant_id, stock_movement_id)
        REFERENCES inventory.stock_movements (tenant_id, id) ON DELETE RESTRICT
);

CREATE TABLE inventory.transfer_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    transfer_id UUID NOT NULL,
    item_id UUID NOT NULL,
    source_location_id UUID NOT NULL,
    destination_location_id UUID NOT NULL,
    quantity NUMERIC(18, 3) NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, transfer_id, item_id),
    CHECK (source_location_id <> destination_location_id),
    FOREIGN KEY (tenant_id, transfer_id)
        REFERENCES inventory.transfers (tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, source_location_id)
        REFERENCES inventory.locations (tenant_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, destination_location_id)
        REFERENCES inventory.locations (tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_inventory_transfers_tenant_date
ON inventory.transfers (tenant_id, transferred_at DESC, id DESC);

ALTER TABLE inventory.transfers ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.transfers FORCE ROW LEVEL SECURITY;
ALTER TABLE inventory.transfer_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.transfer_lines FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_inventory_transfers ON inventory.transfers
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_inventory_transfer_lines ON inventory.transfer_lines
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description) VALUES
    ('inventory.transfer.read', 'inventory', 'Read inventory transfers'),
    ('inventory.transfer.write', 'inventory', 'Post inventory transfers')
ON CONFLICT (id) DO UPDATE SET module_id = EXCLUDED.module_id, description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON inventory.transfers, inventory.transfer_lines TO erp_app;
