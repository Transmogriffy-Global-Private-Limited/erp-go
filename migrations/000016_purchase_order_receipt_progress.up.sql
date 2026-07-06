ALTER TABLE purchase.purchase_orders
DROP CONSTRAINT IF EXISTS purchase_orders_status_check;

ALTER TABLE purchase.purchase_orders
ADD CONSTRAINT purchase_orders_status_check
CHECK (status IN ('draft', 'approved', 'partially_received', 'received', 'cancelled'));

ALTER TABLE purchase.purchase_order_lines
ADD CONSTRAINT uq_purchase_order_lines_order_item
UNIQUE (tenant_id, purchase_order_id, item_id);
