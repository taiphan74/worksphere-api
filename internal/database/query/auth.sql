-- name: CreateUserWithPassword :one
INSERT INTO users (
  id,
  email,
  password_hash,
  full_name
)
VALUES (
  $1,
  $2,
  $3,
  NULL
)
RETURNING id, email, full_name, username, avatar_url, phone, job_title, status, password_changed_at, created_at, updated_at, deleted_at;

-- name: GetUserByEmail :one
SELECT
  id,
  email,
  full_name,
  username,
  avatar_url,
  phone,
  job_title,
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
SELECT id, email, full_name, username, avatar_url, phone, job_title, status, password_changed_at, created_at, updated_at, deleted_at
FROM users
WHERE id = $1
  AND deleted_at IS NULL;
