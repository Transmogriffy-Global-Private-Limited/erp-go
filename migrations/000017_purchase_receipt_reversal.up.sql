ALTER TABLE inventory.stock_movements
DROP CONSTRAINT IF EXISTS stock_movements_movement_type_check;

ALTER TABLE inventory.stock_movements
ADD CONSTRAINT stock_movements_movement_type_check
CHECK (movement_type IN ('adjustment', 'receipt', 'receipt_reversal', 'issue', 'transfer'));

ALTER TABLE purchase.receipts
DROP CONSTRAINT IF EXISTS receipts_status_check;

ALTER TABLE purchase.receipts
ADD CONSTRAINT receipts_status_check
CHECK (status IN ('posted', 'reversed'));

ALTER TABLE purchase.receipts
ADD COLUMN IF NOT EXISTS reversal_stock_movement_id UUID,
ADD COLUMN IF NOT EXISTS reversal_reason TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS reversed_by UUID,
ADD COLUMN IF NOT EXISTS reversed_at TIMESTAMPTZ;

ALTER TABLE purchase.receipts
DROP CONSTRAINT IF EXISTS fk_purchase_receipts_reversal_stock_movement;

ALTER TABLE purchase.receipts
ADD CONSTRAINT fk_purchase_receipts_reversal_stock_movement
FOREIGN KEY (tenant_id, reversal_stock_movement_id)
REFERENCES inventory.stock_movements (tenant_id, id)
ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_purchase_receipts_tenant_status
ON purchase.receipts (tenant_id, status, received_at DESC);

INSERT INTO core.permissions (id, module_id, description)
VALUES ('purchase.receipt.reverse', 'purchase', 'Reverse posted purchase receipts')
ON CONFLICT (id) DO UPDATE
SET module_id = EXCLUDED.module_id,
    description = EXCLUDED.description;
