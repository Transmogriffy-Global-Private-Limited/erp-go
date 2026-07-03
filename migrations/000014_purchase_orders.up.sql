CREATE TABLE IF NOT EXISTS purchase.purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    order_number TEXT NOT NULL,
    supplier_id UUID NOT NULL,
    supplier_reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    currency_code TEXT NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
    ordered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expected_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'approved', 'cancelled')),
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_purchase_orders_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_purchase_orders_tenant_number UNIQUE (tenant_id, order_number),
    CONSTRAINT fk_purchase_orders_supplier
        FOREIGN KEY (tenant_id, supplier_id)
        REFERENCES purchase.suppliers (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS purchase.purchase_order_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    purchase_order_id UUID NOT NULL,
    item_id UUID NOT NULL,
    quantity NUMERIC(18, 3) NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(18, 4) NOT NULL CHECK (unit_price >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_purchase_order_lines_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT fk_purchase_order_lines_order
        FOREIGN KEY (tenant_id, purchase_order_id)
        REFERENCES purchase.purchase_orders (tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_purchase_order_lines_item
        FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_tenant_ordered
ON purchase.purchase_orders (tenant_id, ordered_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_order_lines_tenant_order
ON purchase.purchase_order_lines (tenant_id, purchase_order_id, created_at, id);

ALTER TABLE purchase.purchase_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase.purchase_orders FORCE ROW LEVEL SECURITY;
ALTER TABLE purchase.purchase_order_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase.purchase_order_lines FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_purchase_orders ON purchase.purchase_orders;
DROP POLICY IF EXISTS tenant_isolation_purchase_order_lines ON purchase.purchase_order_lines;

CREATE POLICY tenant_isolation_purchase_orders
ON purchase.purchase_orders
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_purchase_order_lines
ON purchase.purchase_order_lines
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('purchase.order.read', 'purchase', 'Read purchase orders'),
    ('purchase.order.create', 'purchase', 'Create purchase orders'),
    ('purchase.order.approve', 'purchase', 'Approve purchase orders')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON purchase.purchase_orders TO erp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON purchase.purchase_order_lines TO erp_app;
