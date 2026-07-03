ALTER TABLE purchase.receipts
ADD COLUMN IF NOT EXISTS supplier_id UUID,
ADD COLUMN IF NOT EXISTS purchase_order_id UUID;

ALTER TABLE purchase.receipts
DROP CONSTRAINT IF EXISTS fk_purchase_receipts_supplier;

ALTER TABLE purchase.receipts
ADD CONSTRAINT fk_purchase_receipts_supplier
FOREIGN KEY (tenant_id, supplier_id)
REFERENCES purchase.suppliers (tenant_id, id)
ON DELETE RESTRICT;

ALTER TABLE purchase.receipts
DROP CONSTRAINT IF EXISTS fk_purchase_receipts_purchase_order;

ALTER TABLE purchase.receipts
ADD CONSTRAINT fk_purchase_receipts_purchase_order
FOREIGN KEY (tenant_id, purchase_order_id)
REFERENCES purchase.purchase_orders (tenant_id, id)
ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_purchase_receipts_tenant_order
ON purchase.receipts (tenant_id, purchase_order_id)
WHERE purchase_order_id IS NOT NULL;
