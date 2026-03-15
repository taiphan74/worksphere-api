-- name: GetProfile :one
SELECT 
    id, email, full_name, avatar_key, phone, job_title, 
    is_verified, status, created_at, updated_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateProfile :one
UPDATE users
SET
    full_name = CASE WHEN sqlc.arg('update_full_name')::boolean THEN sqlc.narg('full_name') ELSE full_name END,
    phone = CASE WHEN sqlc.arg('update_phone')::boolean THEN sqlc.narg('phone') ELSE phone END,
    job_title = CASE WHEN sqlc.arg('update_job_title')::boolean THEN sqlc.narg('job_title') ELSE job_title END,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, email, full_name, avatar_key, phone, job_title, is_verified, status, created_at, updated_at;

-- name: UpdateAvatarKey :exec
UPDATE users
SET
    avatar_key = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserPasswordHash :one
SELECT password_hash
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: ChangePassword :exec
UPDATE users
SET
    password_hash = $2,
    password_changed_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
