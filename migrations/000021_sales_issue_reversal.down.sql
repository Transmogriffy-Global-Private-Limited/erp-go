DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM sales.issues WHERE status = 'reversed') THEN
        RAISE EXCEPTION 'cannot roll back sales issue reversal while reversed issues exist';
    END IF;
END
$$;

DELETE FROM core.permissions
WHERE id = 'sales.issue.reverse';

DROP INDEX IF EXISTS sales.idx_sales_issues_tenant_status;

ALTER TABLE sales.issues
DROP CONSTRAINT IF EXISTS fk_sales_issues_reversal_stock_movement;

ALTER TABLE sales.issues
DROP COLUMN IF EXISTS reversal_stock_movement_id,
DROP COLUMN IF EXISTS reversal_reason,
DROP COLUMN IF EXISTS reversed_by,
DROP COLUMN IF EXISTS reversed_at;

ALTER TABLE sales.issues
DROP CONSTRAINT IF EXISTS issues_status_check;

ALTER TABLE sales.issues
ADD CONSTRAINT issues_status_check
CHECK (status IN ('posted'));

ALTER TABLE inventory.stock_movements
DROP CONSTRAINT IF EXISTS stock_movements_movement_type_check;

ALTER TABLE inventory.stock_movements
ADD CONSTRAINT stock_movements_movement_type_check
CHECK (movement_type IN ('adjustment', 'receipt', 'receipt_reversal', 'issue', 'transfer'));
