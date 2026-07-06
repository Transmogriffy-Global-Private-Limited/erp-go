ALTER TABLE sales.orders
DROP CONSTRAINT IF EXISTS orders_status_check;

ALTER TABLE sales.orders
DROP CONSTRAINT IF EXISTS sales_orders_status_check;

ALTER TABLE sales.orders
ADD CONSTRAINT sales_orders_status_check
CHECK (status IN ('draft', 'confirmed', 'partially_fulfilled', 'fulfilled', 'cancelled'));

CREATE TABLE IF NOT EXISTS sales.issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    issue_number TEXT NOT NULL,
    sales_order_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    customer_name TEXT NOT NULL,
    reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'posted' CHECK (status IN ('posted')),
    stock_movement_id UUID NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_sales_issues_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_sales_issues_tenant_number UNIQUE (tenant_id, issue_number),
    CONSTRAINT fk_sales_issues_order
        FOREIGN KEY (tenant_id, sales_order_id)
        REFERENCES sales.orders (tenant_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_sales_issues_customer
        FOREIGN KEY (tenant_id, customer_id)
        REFERENCES sales.customers (tenant_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_sales_issues_stock_movement
        FOREIGN KEY (tenant_id, stock_movement_id)
        REFERENCES inventory.stock_movements (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS sales.issue_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    issue_id UUID NOT NULL,
    item_id UUID NOT NULL,
    location_id UUID NOT NULL,
    quantity_issued NUMERIC(18, 3) NOT NULL CHECK (quantity_issued > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_sales_issue_lines_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_sales_issue_lines_issue_item UNIQUE (tenant_id, issue_id, item_id),
    CONSTRAINT fk_sales_issue_lines_issue
        FOREIGN KEY (tenant_id, issue_id)
        REFERENCES sales.issues (tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_sales_issue_lines_item
        FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_sales_issue_lines_location
        FOREIGN KEY (tenant_id, location_id)
        REFERENCES inventory.locations (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_sales_issues_tenant_issued
ON sales.issues (tenant_id, issued_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_sales_issues_tenant_order
ON sales.issues (tenant_id, sales_order_id, status);

CREATE INDEX IF NOT EXISTS idx_sales_issue_lines_tenant_item_location
ON sales.issue_lines (tenant_id, item_id, location_id);

ALTER TABLE sales.issues ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales.issues FORCE ROW LEVEL SECURITY;
ALTER TABLE sales.issue_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales.issue_lines FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_sales_issues ON sales.issues;
DROP POLICY IF EXISTS tenant_isolation_sales_issue_lines ON sales.issue_lines;

CREATE POLICY tenant_isolation_sales_issues
ON sales.issues
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_sales_issue_lines
ON sales.issue_lines
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('sales.issue.read', 'sales', 'Read posted sales issues'),
    ('sales.issue.write', 'sales', 'Post sales issues')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON sales.issues TO erp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sales.issue_lines TO erp_app;
