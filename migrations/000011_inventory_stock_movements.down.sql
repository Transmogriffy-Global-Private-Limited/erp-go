DELETE FROM core.permissions
WHERE id IN (
    'inventory.stock_movement.read',
    'inventory.stock_movement.write',
    'inventory.stock_balance.read'
);

DROP TABLE IF EXISTS inventory.stock_movement_lines CASCADE;
DROP TABLE IF EXISTS inventory.stock_movements CASCADE;
