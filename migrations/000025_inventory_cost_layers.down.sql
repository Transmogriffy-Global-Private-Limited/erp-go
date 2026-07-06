DELETE FROM core.permissions WHERE id IN ('inventory.cost.read', 'inventory.cost.write', 'inventory.report.read');
DROP TABLE IF EXISTS inventory.cost_layers CASCADE;
