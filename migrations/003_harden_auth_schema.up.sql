DO $$
BEGIN
  UPDATE users
  SET password_hash = NULL
  WHERE password_hash IS NOT NULL AND btrim(password_hash) = '';
END $$;

ALTER TABLE users
ALTER COLUMN password_hash DROP DEFAULT,
ALTER COLUMN password_hash DROP NOT NULL,
ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
ALTER COLUMN created_at SET DEFAULT NOW(),
ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC',
ALTER COLUMN updated_at SET DEFAULT NOW();
