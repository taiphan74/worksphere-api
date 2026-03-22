-- name: CreateWorkspaceInvitation :one
INSERT INTO workspace_invitations (
    id,
    workspace_id,
    email,
    token,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    LOWER($3),
    $4,
    NOW(),
    NOW()
) RETURNING *;

-- name: GetWorkspaceInvitationByID :one
SELECT * FROM workspace_invitations
WHERE id = $1;

-- name: GetWorkspaceInvitationByToken :one
SELECT * FROM workspace_invitations
WHERE token = $1;

-- name: GetWorkspaceInvitationByEmailAndWorkspace :one
SELECT * FROM workspace_invitations
WHERE workspace_id = $2 AND LOWER(email) = LOWER($1)
LIMIT 1;

-- name: ListWorkspaceInvitationsByWorkspace :many
SELECT * FROM workspace_invitations
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: DeleteWorkspaceInvitation :exec
DELETE FROM workspace_invitations
WHERE id = $1;

-- name: CountPendingInvitationsByEmail :one
SELECT COUNT(*) FROM workspace_invitations
WHERE LOWER(email) = LOWER($1);
