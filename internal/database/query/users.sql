-- name: CreateUser :one
INSERT INTO users (
  id,
  email,
  password_hash,
  full_name,
  username,
  avatar_url,
  phone,
  job_title,
  status,
  password_changed_at
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  NOW()
)
RETURNING id, email, full_name, username, avatar_url, phone, job_title, status, email_verified_at, last_login_at, password_changed_at, created_at, updated_at, deleted_at;

-- name: GetUserByID :one
SELECT id, email, full_name, username, avatar_url, phone, job_title, status, email_verified_at, last_login_at, password_changed_at, created_at, updated_at, deleted_at
FROM users
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT id, email, full_name, username, avatar_url, phone, job_title, status, email_verified_at, last_login_at, password_changed_at, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::varchar IS NULL OR status = sqlc.narg(status)::varchar)
  AND (
    sqlc.narg(search)::varchar IS NULL
    OR email ILIKE '%' || sqlc.narg(search)::varchar || '%'
    OR COALESCE(username, '') ILIKE '%' || sqlc.narg(search)::varchar || '%'
  )
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET
  email = CASE WHEN sqlc.arg(set_email)::bool THEN sqlc.arg(email)::varchar(255) ELSE email END,
  full_name = CASE WHEN sqlc.arg(set_full_name)::bool THEN sqlc.arg(full_name)::varchar(150) ELSE full_name END,
  username = CASE WHEN sqlc.arg(set_username)::bool THEN sqlc.narg(username)::varchar(50) ELSE username END,
  avatar_url = CASE WHEN sqlc.arg(set_avatar_url)::bool THEN sqlc.narg(avatar_url)::varchar(500) ELSE avatar_url END,
  phone = CASE WHEN sqlc.arg(set_phone)::bool THEN sqlc.narg(phone)::varchar(20) ELSE phone END,
  job_title = CASE WHEN sqlc.arg(set_job_title)::bool THEN sqlc.narg(job_title)::varchar(100) ELSE job_title END,
  status = CASE WHEN sqlc.arg(set_status)::bool THEN sqlc.arg(status)::varchar(20) ELSE status END,
  password_hash = CASE WHEN sqlc.arg(set_password_hash)::bool THEN sqlc.arg(password_hash)::varchar(255) ELSE password_hash END,
  password_changed_at = CASE WHEN sqlc.arg(set_password_hash)::bool THEN NOW() ELSE password_changed_at END,
  updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL
RETURNING id, email, full_name, username, avatar_url, phone, job_title, status, email_verified_at, last_login_at, password_changed_at, created_at, updated_at, deleted_at;

-- name: DeleteUser :one
UPDATE users
SET
  deleted_at = NOW(),
  updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id;

-- name: RestoreUser :one
UPDATE users
SET
  deleted_at = NULL,
  updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NOT NULL
RETURNING id, email, full_name, username, avatar_url, phone, job_title, status, email_verified_at, last_login_at, password_changed_at, created_at, updated_at, deleted_at;
