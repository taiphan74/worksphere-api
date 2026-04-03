-- name: CreateTask :one
INSERT INTO tasks (
    id,
    workspace_id,
    creator_id,
    assignee_id,
    title,
    description,
    status,
    priority,
    due_at,
    sprint_points,
    parent_id,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    NOW(),
    NOW()
) RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks
WHERE id = $1
  AND workspace_id = $2
  AND deleted_at IS NULL;

-- name: ListTasksByWorkspace :many
SELECT * FROM tasks
WHERE workspace_id = sqlc.arg('workspace_id')
  AND (sqlc.arg('include_deleted')::boolean OR deleted_at IS NULL)
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR assignee_id = sqlc.narg('assignee_id')::uuid)
  AND (sqlc.narg('parent_id')::uuid IS NULL OR parent_id = sqlc.narg('parent_id')::uuid)
  AND (sqlc.narg('due_from')::timestamptz IS NULL OR due_at >= sqlc.narg('due_from')::timestamptz)
  AND (sqlc.narg('due_to')::timestamptz IS NULL OR due_at <= sqlc.narg('due_to')::timestamptz)
ORDER BY created_at DESC;

-- name: UpdateTask :one
UPDATE tasks
SET
    title = CASE
        WHEN sqlc.arg('update_title')::boolean THEN sqlc.arg('title')::varchar
        ELSE title
    END,
    description = CASE
        WHEN sqlc.arg('update_description')::boolean THEN sqlc.narg('description')::text
        ELSE description
    END,
    status = CASE
        WHEN sqlc.arg('update_status')::boolean THEN sqlc.arg('status')::varchar
        ELSE status
    END,
    priority = CASE
        WHEN sqlc.arg('update_priority')::boolean THEN sqlc.arg('priority')::varchar
        ELSE priority
    END,
    due_at = CASE
        WHEN sqlc.arg('update_due_at')::boolean THEN sqlc.narg('due_at')::timestamptz
        ELSE due_at
    END,
    sprint_points = CASE
        WHEN sqlc.arg('update_sprint_points')::boolean THEN sqlc.narg('sprint_points')::int
        ELSE sprint_points
    END,
    assignee_id = CASE
        WHEN sqlc.arg('update_assignee_id')::boolean THEN sqlc.narg('assignee_id')::uuid
        ELSE assignee_id
    END,
    parent_id = CASE
        WHEN sqlc.arg('update_parent_id')::boolean THEN sqlc.narg('parent_id')::uuid
        ELSE parent_id
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
  AND workspace_id = sqlc.arg('workspace_id')
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTask :exec
UPDATE tasks
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1
  AND workspace_id = $2
  AND deleted_at IS NULL;
