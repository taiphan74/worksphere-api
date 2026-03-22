-- name: AddWorkspaceMember :one
INSERT INTO workspace_members (
    id, workspace_id, user_id, role, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, NOW(), NOW()
) RETURNING *;

-- name: GetWorkspaceMember :one
SELECT * FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: ListWorkspaceMembersByWorkspace :many
SELECT 
    wm.id, wm.workspace_id, wm.user_id, wm.role, wm.created_at, wm.updated_at,
    u.email, u.full_name, u.avatar_key, u.status AS user_status
FROM workspace_members wm
JOIN users u ON wm.user_id = u.id
WHERE wm.workspace_id = $1
ORDER BY wm.created_at ASC;

-- name: UpdateMemberRole :one
UPDATE workspace_members
SET role = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkspaceMember :exec
DELETE FROM workspace_members
WHERE id = $1;

-- name: CountWorkspaceMembersByRole :one
SELECT COUNT(*) FROM workspace_members
WHERE workspace_id = $1 AND role = $2;
