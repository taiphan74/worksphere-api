DROP INDEX IF EXISTS users_username_active_unique_idx;
DROP INDEX IF EXISTS users_email_active_unique_idx;

ALTER TABLE users
DROP COLUMN IF EXISTS deleted_at,
DROP COLUMN IF EXISTS password_changed_at,
DROP COLUMN IF EXISTS last_login_at,
DROP COLUMN IF EXISTS email_verified_at,
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS job_title,
DROP COLUMN IF EXISTS phone,
DROP COLUMN IF EXISTS avatar_url,
DROP COLUMN IF EXISTS username;

ALTER TABLE users
ALTER COLUMN email TYPE TEXT,
ALTER COLUMN full_name TYPE TEXT,
ALTER COLUMN password_hash TYPE TEXT;

ALTER TABLE users
ADD CONSTRAINT users_email_key UNIQUE (email);
