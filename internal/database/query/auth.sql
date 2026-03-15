-- name: CreateUserWithPassword :one
INSERT INTO users (
  id,
  email,
  password_hash,
  full_name,
  is_verified,
  status
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6
)
RETURNING id, email, full_name, avatar_key, phone, job_title, is_verified, status, password_changed_at, created_at, updated_at, deleted_at;

-- name: ResetUserPassword :one
UPDATE users
SET
  password_hash = $2,
  password_changed_at = NOW(),
  updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, email, full_name, avatar_key, phone, job_title, is_verified, status, password_changed_at, created_at, updated_at, deleted_at;

-- name: GetUserByEmail :one
SELECT
  id,
  email,
  full_name,
  avatar_key,
  phone,
  job_title,
  is_verified,
  status,
  COALESCE(password_hash, '') AS password_hash,
  password_changed_at,
  created_at,
  updated_at,
  deleted_at
FROM users
WHERE email = $1
  AND deleted_at IS NULL;

-- name: GetUserByIDForAuthProfile :one
SELECT id, email, full_name, avatar_key, phone, job_title, is_verified, status, password_changed_at, created_at, updated_at, deleted_at
FROM users
WHERE id = $1
  AND deleted_at IS NULL;

-- name: MarkUserEmailVerified :one
UPDATE users
SET
  is_verified = true,
  updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, email, full_name, avatar_key, phone, job_title, is_verified, status, password_changed_at, created_at, updated_at, deleted_at;
