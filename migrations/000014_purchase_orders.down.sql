DELETE FROM core.permissions
WHERE id IN (
    'purchase.order.read',
    'purchase.order.create',
    'purchase.order.approve'
);

DROP TABLE IF EXISTS purchase.purchase_order_lines CASCADE;
DROP TABLE IF EXISTS purchase.purchase_orders CASCADE;
