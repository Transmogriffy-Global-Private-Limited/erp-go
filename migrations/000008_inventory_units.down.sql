DELETE FROM core.permissions
WHERE id IN (
    'inventory.unit.read',
    'inventory.unit.write'
);

DROP TABLE IF EXISTS inventory.units CASCADE;
