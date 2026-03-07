-- name: CreateUserWithPassword :one
INSERT INTO users (
  id,
  email,
  full_name,
  password_hash
)
VALUES (
  $1,
  $2,
  $3,
  $4
)
RETURNING id, email, full_name, password_hash, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, full_name, password_hash, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByIDForAuthProfile :one
SELECT id, email, full_name, password_hash, created_at, updated_at
FROM users
WHERE id = $1;
