ALTER TABLE purchase.purchase_order_lines
DROP CONSTRAINT IF EXISTS uq_purchase_order_lines_order_item;

UPDATE purchase.purchase_orders
SET status = 'approved',
    updated_at = now()
WHERE status IN ('partially_received', 'received');

ALTER TABLE purchase.purchase_orders
DROP CONSTRAINT IF EXISTS purchase_orders_status_check;

ALTER TABLE purchase.purchase_orders
ADD CONSTRAINT purchase_orders_status_check
CHECK (status IN ('draft', 'approved', 'cancelled'));
