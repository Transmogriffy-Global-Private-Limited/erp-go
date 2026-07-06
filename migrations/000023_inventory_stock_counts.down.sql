DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM inventory.stock_counts) THEN
        RAISE EXCEPTION 'cannot roll back inventory stock counts while count rows exist';
    END IF;
END $$;
DELETE FROM core.permissions WHERE id IN ('inventory.stock_count.read', 'inventory.stock_count.write');
DROP TABLE IF EXISTS inventory.stock_count_lines CASCADE;
DROP TABLE IF EXISTS inventory.stock_counts CASCADE;
