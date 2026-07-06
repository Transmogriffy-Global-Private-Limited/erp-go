DELETE FROM core.permissions
WHERE id IN (
    'sales.customer.read',
    'sales.customer.write'
);

DROP TABLE IF EXISTS sales.customers CASCADE;
