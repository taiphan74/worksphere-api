-- name: CreateWorkspace :one
INSERT INTO workspaces (
    id, name, slug, description, owner_user_id, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, NOW(), NOW()
) RETURNING id, name, slug, description, owner_user_id, created_at, updated_at;

-- name: GetWorkspaceByID :one
SELECT id, name, slug, description, owner_user_id, created_at, updated_at
FROM workspaces
WHERE id = $1;

-- name: GetWorkspaceBySlug :one
SELECT id, name, slug, description, owner_user_id, created_at, updated_at
FROM workspaces
WHERE slug = $1;

-- name: ListWorkspacesByOwner :many
SELECT id, name, slug, description, owner_user_id, created_at, updated_at
FROM workspaces
WHERE owner_user_id = $1
ORDER BY created_at DESC;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET
    name = CASE WHEN sqlc.arg('update_name')::boolean THEN sqlc.arg('name')::varchar ELSE name END,
    slug = CASE WHEN sqlc.arg('update_slug')::boolean THEN sqlc.arg('slug')::varchar ELSE slug END,
    description = CASE WHEN sqlc.arg('update_description')::boolean THEN sqlc.narg('description') ELSE description END,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, slug, description, owner_user_id, created_at, updated_at;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;

-- name: CheckSlugExists :one
SELECT EXISTS(
    SELECT 1 FROM workspaces 
    WHERE slug = $1 AND (sqlc.narg('exclude_id')::uuid IS NULL OR id != sqlc.narg('exclude_id')::uuid)
);
