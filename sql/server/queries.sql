-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, username, email, created_at;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, username, email
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, username, email, password_hash
FROM users
WHERE username = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING token;

-- name: GetSession :one
SELECT user_id, expires_at
FROM sessions
WHERE token = $1 AND expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: CreateMagicLink :exec
INSERT INTO magic_links (email, token, expires_at)
VALUES ($1, $2, $3);

-- name: GetMagicLink :one
SELECT email, expires_at, used
FROM magic_links
WHERE token = $1;

-- name: MarkMagicLinkUsed :exec
UPDATE magic_links SET used = TRUE WHERE token = $1;

-- name: UpsertProject :one
INSERT INTO projects (user_id, client_id, name, color, encrypted_data, sync_version, deleted, updated_at)
VALUES ($1, $2, '', $3, $4, 
    (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM projects WHERE user_id = $1), 
    $5, NOW())
ON CONFLICT (user_id, client_id) DO UPDATE
SET encrypted_data = EXCLUDED.encrypted_data,
    deleted = EXCLUDED.deleted,
    sync_version = (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM projects WHERE user_id = $1),
    updated_at = NOW()
RETURNING sync_version;

-- name: GetProjectsChanged :many
SELECT client_id, 'project' as type, sync_version, encrypted_data, deleted
FROM projects
WHERE user_id = $1 AND sync_version > $2;

-- name: UpsertTask :one
INSERT INTO tasks (user_id, client_id, project_id, encrypted_data, deleted, sync_version, updated_at)
VALUES ($1, $2, $3, $4, $5, 
    (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM tasks WHERE user_id = $1),
    NOW())
ON CONFLICT (user_id, client_id) DO UPDATE
SET project_id = EXCLUDED.project_id,
    encrypted_data = EXCLUDED.encrypted_data,
    deleted = EXCLUDED.deleted,
    sync_version = (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM tasks WHERE user_id = $1),
    updated_at = NOW()
RETURNING sync_version;

-- name: GetTasksChanged :many
SELECT client_id, project_id, 'task' as type, sync_version, encrypted_data, deleted
FROM tasks
WHERE user_id = $1 AND sync_version > $2;

-- name: ClearTasks :exec
DELETE FROM tasks WHERE user_id = $1;

-- name: ClearProjects :exec
DELETE FROM projects WHERE user_id = $1;
