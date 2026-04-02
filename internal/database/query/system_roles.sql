-- name: GetSystemRoleByCode :one
SELECT id, code, name, description, is_system, created_at, updated_at
FROM system_roles
WHERE code = $1;

-- name: AssignSystemRoleToUser :exec
INSERT INTO user_system_roles (
  user_id,
  role_id,
  assigned_by
)
VALUES (
  $1,
  $2,
  $3
)
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: GetUserRoleCodes :many
SELECT r.code
FROM system_roles r
JOIN user_system_roles usr ON r.id = usr.role_id
WHERE usr.user_id = $1;
