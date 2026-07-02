DELETE FROM core.permissions
WHERE id IN (
    'inventory.location.read',
    'inventory.location.write'
);

DROP TABLE IF EXISTS inventory.locations CASCADE;
