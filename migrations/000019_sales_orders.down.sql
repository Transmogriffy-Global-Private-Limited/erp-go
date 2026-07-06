UPDATE core.permissions
SET description = 'Approve sales orders'
WHERE id = 'sales.order.approve';

DROP TABLE IF EXISTS sales.order_lines CASCADE;
DROP TABLE IF EXISTS sales.orders CASCADE;
