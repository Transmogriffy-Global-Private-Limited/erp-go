DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM sales.issues) THEN
        RAISE EXCEPTION 'cannot roll back sales issues while posted issues exist';
    END IF;
END
$$;

DELETE FROM core.permissions
WHERE id IN ('sales.issue.read', 'sales.issue.write');

DROP TABLE IF EXISTS sales.issue_lines CASCADE;
DROP TABLE IF EXISTS sales.issues CASCADE;

ALTER TABLE sales.orders
DROP CONSTRAINT IF EXISTS sales_orders_status_check;

ALTER TABLE sales.orders
ADD CONSTRAINT orders_status_check
CHECK (status IN ('draft', 'confirmed', 'cancelled'));
