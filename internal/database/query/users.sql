-- name: CreateUser :one
INSERT INTO users (
  id,
  email,
  password_hash,
  full_name,
  status,
  password_changed_at
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  NOW()
)
RETURNING id, email, full_name, status, password_changed_at, created_at, updated_at, deleted_at;

-- name: GetUserByID :one
SELECT id, email, full_name, status, password_changed_at, created_at, updated_at, deleted_at
FROM users
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT id, email, full_name, status, password_changed_at, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::varchar IS NULL OR status = sqlc.narg(status)::varchar)
  AND (
    sqlc.narg(search)::varchar IS NULL
    OR email ILIKE '%' || sqlc.narg(search)::varchar || '%'
    OR COALESCE(full_name, '') ILIKE '%' || sqlc.narg(search)::varchar || '%'
  )
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET
  full_name = CASE WHEN sqlc.arg(set_full_name)::bool THEN sqlc.narg(full_name)::varchar(150) ELSE full_name END,
  password_hash = CASE WHEN sqlc.arg(set_password_hash)::bool THEN sqlc.arg(password_hash)::varchar(255) ELSE password_hash END,
  password_changed_at = CASE WHEN sqlc.arg(set_password_hash)::bool THEN NOW() ELSE password_changed_at END,
  updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL
RETURNING id, email, full_name, status, password_changed_at, created_at, updated_at, deleted_at;

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
RETURNING id, email, full_name, status, password_changed_at, created_at, updated_at, deleted_at;
