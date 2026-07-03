DELETE FROM core.permissions
WHERE id IN (
    'purchase.supplier.read',
    'purchase.supplier.write'
);

DROP TABLE IF EXISTS purchase.suppliers CASCADE;
