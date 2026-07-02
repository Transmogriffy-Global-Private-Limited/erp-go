CREATE SCHEMA IF NOT EXISTS purchase;

CREATE TABLE IF NOT EXISTS purchase.receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    receipt_number TEXT NOT NULL,
    supplier_name TEXT NOT NULL DEFAULT '',
    reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'posted'
        CHECK (status IN ('posted')),
    stock_movement_id UUID NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, receipt_number),
    CONSTRAINT fk_purchase_receipts_stock_movement
        FOREIGN KEY (tenant_id, stock_movement_id)
        REFERENCES inventory.stock_movements (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS purchase.receipt_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES control.tenants(id) ON DELETE CASCADE,
    receipt_id UUID NOT NULL,
    item_id UUID NOT NULL,
    location_id UUID NOT NULL,
    quantity_received NUMERIC(18, 3) NOT NULL CHECK (quantity_received > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_purchase_receipt_lines_receipt
        FOREIGN KEY (tenant_id, receipt_id)
        REFERENCES purchase.receipts (tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_purchase_receipt_lines_item
        FOREIGN KEY (tenant_id, item_id)
        REFERENCES inventory.items (tenant_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_purchase_receipt_lines_location
        FOREIGN KEY (tenant_id, location_id)
        REFERENCES inventory.locations (tenant_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_purchase_receipts_tenant_received
ON purchase.receipts (tenant_id, received_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_receipt_lines_tenant_item_location
ON purchase.receipt_lines (tenant_id, item_id, location_id);

ALTER TABLE purchase.receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase.receipts FORCE ROW LEVEL SECURITY;

ALTER TABLE purchase.receipt_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase.receipt_lines FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_purchase_receipts ON purchase.receipts;
DROP POLICY IF EXISTS tenant_isolation_purchase_receipt_lines ON purchase.receipt_lines;

CREATE POLICY tenant_isolation_purchase_receipts
ON purchase.receipts
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

CREATE POLICY tenant_isolation_purchase_receipt_lines
ON purchase.receipt_lines
USING (tenant_id = core.current_tenant_id())
WITH CHECK (tenant_id = core.current_tenant_id());

INSERT INTO core.permissions (id, module_id, description)
VALUES
    ('purchase.receipt.read', 'purchase', 'Read purchase receipts'),
    ('purchase.receipt.write', 'purchase', 'Create purchase receipts')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;

GRANT USAGE ON SCHEMA purchase TO erp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON purchase.receipts TO erp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON purchase.receipt_lines TO erp_app;
