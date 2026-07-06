CREATE TABLE IF NOT EXISTS sales.orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    order_number TEXT NOT NULL,
    customer_id UUID NOT NULL,
    customer_reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    currency_code TEXT NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
    ordered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    requested_delivery_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'confirmed', 'cancelled')),
    confirmed_by UUID,
    confirmed_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_sales_orders_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_sales_orders_tenant_number UNIQUE (tenant_id, order_number),
    CONSTRAINT fk_sales_orders_customer
        FOREIGN KEY (tenant_id, customer_id)
        REFERENCES sales.customers (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS sales.order_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    sales_order_id UUID NOT NULL,
    item_id UUID NOT NULL,
    quantity NUMERIC(18, 3) NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(18, 4) NOT NULL CHECK (unit_price >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_sales_order_lines_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_sales_order_lines_order_item UNIQUE (tenant_id, sales_order_id, item_id),
    CONSTRAINT fk_sales_order_lines_order
        FOREIGN KEY (tenant_id, sales_order_id)
        REFERENCES sales.orders (tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_sales_order_lines_item
        FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_sales_orders_tenant_ordered
ON sales.orders (tenant_id, ordered_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_sales_order_lines_tenant_order
ON sales.order_lines (tenant_id, sales_order_id, created_at, id);

ALTER TABLE sales.orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales.orders FORCE ROW LEVEL SECURITY;
ALTER TABLE sales.order_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales.order_lines FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_sales_orders ON sales.orders;
DROP POLICY IF EXISTS tenant_isolation_sales_order_lines ON sales.order_lines;

CREATE POLICY tenant_isolation_sales_orders
ON sales.orders
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_sales_order_lines
ON sales.order_lines
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('sales.order.read', 'sales', 'Read sales orders'),
    ('sales.order.create', 'sales', 'Create sales orders'),
    ('sales.order.approve', 'sales', 'Confirm sales orders')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON sales.orders TO erp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sales.order_lines TO erp_app;
