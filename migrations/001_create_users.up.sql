CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  full_name VARCHAR(150),
  username VARCHAR(50),
  avatar_url VARCHAR(500),
  phone VARCHAR(20),
  job_title VARCHAR(100),
  status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  password_changed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX users_email_active_unique_idx ON users (LOWER(email)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX users_username_active_unique_idx ON users (LOWER(username)) WHERE deleted_at IS NULL AND username IS NOT NULL;
