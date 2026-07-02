ALTER TABLE inventory.items
ADD COLUMN IF NOT EXISTS base_unit_id UUID;

CREATE INDEX IF NOT EXISTS idx_inventory_items_tenant_base_unit
ON inventory.items (tenant_id, base_unit_id)
WHERE base_unit_id IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_inventory_items_base_unit'
          AND conrelid = 'inventory.items'::regclass
    ) THEN
        ALTER TABLE inventory.items
        ADD CONSTRAINT fk_inventory_items_base_unit
        FOREIGN KEY (tenant_id, base_unit_id)
        REFERENCES inventory.units (tenant_id, id)
        ON DELETE RESTRICT;
    END IF;
END;
$$;
