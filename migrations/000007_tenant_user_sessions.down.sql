DROP TABLE IF EXISTS core.tenant_user_sessions CASCADE;

ALTER TABLE core.tenant_users
DROP COLUMN IF EXISTS password_hash;
