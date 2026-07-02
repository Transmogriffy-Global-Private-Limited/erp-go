ALTER TABLE inventory.items
DROP CONSTRAINT IF EXISTS fk_inventory_items_base_unit;

DROP INDEX IF EXISTS inventory.idx_inventory_items_tenant_base_unit;

ALTER TABLE inventory.items
DROP COLUMN IF EXISTS base_unit_id;
