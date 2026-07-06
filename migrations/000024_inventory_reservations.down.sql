DELETE FROM core.permissions WHERE id IN ('inventory.reservation.read', 'inventory.reservation.write');
DROP TABLE IF EXISTS inventory.reservations CASCADE;
