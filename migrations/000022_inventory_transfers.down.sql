DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM inventory.transfers) THEN
        RAISE EXCEPTION 'cannot roll back inventory transfers while transfer rows exist';
    END IF;
END $$;
DELETE FROM core.permissions WHERE id IN ('inventory.transfer.read', 'inventory.transfer.write');
DROP TABLE IF EXISTS inventory.transfer_lines CASCADE;
DROP TABLE IF EXISTS inventory.transfers CASCADE;
