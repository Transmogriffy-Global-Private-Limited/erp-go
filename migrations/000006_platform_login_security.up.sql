ALTER TABLE control.platform_users
ADD COLUMN IF NOT EXISTS failed_login_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE control.platform_users
ADD COLUMN IF NOT EXISTS last_failed_login_at TIMESTAMPTZ;

ALTER TABLE control.platform_users
ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_platform_users_locked_until
ON control.platform_users (locked_until)
WHERE locked_until IS NOT NULL;
