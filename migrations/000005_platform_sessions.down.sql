DROP TABLE IF EXISTS control.platform_sessions CASCADE;

ALTER TABLE control.platform_users
DROP COLUMN IF EXISTS password_hash;
