DROP INDEX IF EXISTS control.idx_platform_users_locked_until;

ALTER TABLE control.platform_users
DROP COLUMN IF EXISTS locked_until;

ALTER TABLE control.platform_users
DROP COLUMN IF EXISTS last_failed_login_at;

ALTER TABLE control.platform_users
DROP COLUMN IF EXISTS failed_login_count;
