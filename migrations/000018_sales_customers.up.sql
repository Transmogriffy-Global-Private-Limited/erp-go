CREATE TABLE IF NOT EXISTS sales.customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    billing_address TEXT NOT NULL DEFAULT '',
    shipping_address TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_sales_customers_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_sales_customers_tenant_code UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_sales_customers_tenant_status
ON sales.customers (tenant_id, status);

ALTER TABLE sales.customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales.customers FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_sales_customers ON sales.customers;

CREATE POLICY tenant_isolation_sales_customers
ON sales.customers
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('sales.customer.read', 'sales', 'Read sales customers'),
    ('sales.customer.write', 'sales', 'Create and update sales customers')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT SELECT, INSERT, UPDATE, DELETE ON sales.customers TO erp_app;
