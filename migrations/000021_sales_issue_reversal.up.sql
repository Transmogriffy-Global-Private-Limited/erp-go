ALTER TABLE inventory.stock_movements
DROP CONSTRAINT IF EXISTS stock_movements_movement_type_check;

ALTER TABLE inventory.stock_movements
ADD CONSTRAINT stock_movements_movement_type_check
CHECK (movement_type IN ('adjustment', 'receipt', 'receipt_reversal', 'issue', 'issue_reversal', 'transfer'));

ALTER TABLE sales.issues
DROP CONSTRAINT IF EXISTS issues_status_check;

ALTER TABLE sales.issues
ADD CONSTRAINT issues_status_check
CHECK (status IN ('posted', 'reversed'));

ALTER TABLE sales.issues
ADD COLUMN IF NOT EXISTS reversal_stock_movement_id UUID,
ADD COLUMN IF NOT EXISTS reversal_reason TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS reversed_by UUID,
ADD COLUMN IF NOT EXISTS reversed_at TIMESTAMPTZ;

ALTER TABLE sales.issues
DROP CONSTRAINT IF EXISTS fk_sales_issues_reversal_stock_movement;

ALTER TABLE sales.issues
ADD CONSTRAINT fk_sales_issues_reversal_stock_movement
FOREIGN KEY (tenant_id, reversal_stock_movement_id)
REFERENCES inventory.stock_movements (tenant_id, id)
ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_sales_issues_tenant_status
ON sales.issues (tenant_id, status, issued_at DESC);

INSERT INTO core.permissions (id, module_id, description)
VALUES ('sales.issue.reverse', 'sales', 'Reverse posted sales issues')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;
