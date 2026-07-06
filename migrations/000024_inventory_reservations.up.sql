CREATE TABLE inventory.reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    reservation_number TEXT NOT NULL,
    item_id UUID NOT NULL,
    location_id UUID NOT NULL,
    quantity NUMERIC(18, 3) NOT NULL CHECK (quantity > 0),
    reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'released')),
    expires_at TIMESTAMPTZ,
    released_by UUID,
    released_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, reservation_number),
    FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, location_id)
        REFERENCES inventory.locations (tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_inventory_reservations_active
ON inventory.reservations (tenant_id, item_id, location_id, status);

ALTER TABLE inventory.reservations ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.reservations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_inventory_reservations ON inventory.reservations
USING (tenant_id = core.current_tenant_id()) WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description) VALUES
    ('inventory.reservation.read', 'inventory', 'Read stock reservations and availability'),
    ('inventory.reservation.write', 'inventory', 'Create and release stock reservations')
ON CONFLICT (id) DO UPDATE SET module_id = EXCLUDED.module_id, description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON inventory.reservations TO erp_app;
