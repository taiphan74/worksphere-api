-- name: CreateWorkspace :one
INSERT INTO workspaces (
    id, name, slug, created_at, updated_at
) VALUES (
    $1, $2, $3, NOW(), NOW()
) RETURNING *;

-- name: GetWorkspaceByID :one
SELECT * FROM workspaces
WHERE id = $1;

-- name: GetWorkspaceBySlug :one
SELECT * FROM workspaces
WHERE slug = $1;

-- name: ListWorkspacesByUser :many
SELECT w.* FROM workspaces w
JOIN workspace_members wm ON w.id = wm.workspace_id
WHERE wm.user_id = $1
ORDER BY w.created_at DESC;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET
    name = CASE WHEN sqlc.arg('update_name')::boolean THEN sqlc.arg('name')::varchar ELSE name END,
    slug = CASE WHEN sqlc.arg('update_slug')::boolean THEN sqlc.arg('slug')::varchar ELSE slug END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;

-- name: CheckSlugExists :one
SELECT EXISTS(
    SELECT 1 FROM workspaces
    WHERE slug = $1 AND (sqlc.narg('exclude_id')::uuid IS NULL OR id != sqlc.narg('exclude_id')::uuid)
);
