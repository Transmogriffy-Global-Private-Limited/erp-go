DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM purchase.receipts WHERE status = 'reversed') THEN
        RAISE EXCEPTION 'cannot roll back purchase receipt reversal while reversed receipts exist';
    END IF;
END
$$;

DELETE FROM core.permissions
WHERE id = 'purchase.receipt.reverse';

DROP INDEX IF EXISTS purchase.idx_purchase_receipts_tenant_status;

ALTER TABLE purchase.receipts
DROP CONSTRAINT IF EXISTS fk_purchase_receipts_reversal_stock_movement;

ALTER TABLE purchase.receipts
DROP COLUMN IF EXISTS reversal_stock_movement_id,
DROP COLUMN IF EXISTS reversal_reason,
DROP COLUMN IF EXISTS reversed_by,
DROP COLUMN IF EXISTS reversed_at;

ALTER TABLE purchase.receipts
DROP CONSTRAINT IF EXISTS receipts_status_check;

ALTER TABLE purchase.receipts
ADD CONSTRAINT receipts_status_check
CHECK (status IN ('posted'));

ALTER TABLE inventory.stock_movements
DROP CONSTRAINT IF EXISTS stock_movements_movement_type_check;

ALTER TABLE inventory.stock_movements
ADD CONSTRAINT stock_movements_movement_type_check
CHECK (movement_type IN ('adjustment', 'receipt', 'issue', 'transfer'));
