-- name: GetUserByID :one
SELECT id, email, full_name, created_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, full_name, created_at
FROM users
ORDER BY created_at DESC;
