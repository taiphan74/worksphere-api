-- name: CreateUser :one
INSERT INTO users (
  id,
  email,
  full_name
)
VALUES (
  $1,
  $2,
  $3
)
RETURNING id, email, full_name, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, full_name, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, full_name, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET
  email = $2,
  full_name = $3,
  updated_at = NOW()
WHERE id = $1
RETURNING id, email, full_name, created_at, updated_at;

-- name: DeleteUser :one
DELETE FROM users
WHERE id = $1
RETURNING id;
