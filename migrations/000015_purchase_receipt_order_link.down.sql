DROP INDEX IF EXISTS purchase.idx_purchase_receipts_tenant_order;

ALTER TABLE purchase.receipts
DROP CONSTRAINT IF EXISTS fk_purchase_receipts_purchase_order;

ALTER TABLE purchase.receipts
DROP CONSTRAINT IF EXISTS fk_purchase_receipts_supplier;

ALTER TABLE purchase.receipts
DROP COLUMN IF EXISTS purchase_order_id,
DROP COLUMN IF EXISTS supplier_id;
