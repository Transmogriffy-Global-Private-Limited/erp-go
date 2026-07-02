DELETE FROM core.permissions
WHERE id IN (
    'purchase.receipt.read',
    'purchase.receipt.write'
);

DROP TABLE IF EXISTS purchase.receipt_lines CASCADE;
DROP TABLE IF EXISTS purchase.receipts CASCADE;
