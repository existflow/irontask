-- name: CreateUser :one
INSERT INTO irontask.users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, username, email, created_at;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash
FROM irontask.users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, username, email
FROM irontask.users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, username, email, password_hash
FROM irontask.users
WHERE username = $1;

-- name: CreateSession :one
INSERT INTO irontask.sessions (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING token;

-- name: GetSession :one
SELECT user_id, expires_at
FROM irontask.sessions
WHERE token = $1 AND expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM irontask.sessions WHERE token = $1;

-- name: CreateMagicLink :exec
INSERT INTO irontask.magic_links (email, token, expires_at)
VALUES ($1, $2, $3);

-- name: GetMagicLink :one
SELECT email, expires_at, used
FROM irontask.magic_links
WHERE token = $1;

-- name: MarkMagicLinkUsed :exec
UPDATE irontask.magic_links SET used = TRUE WHERE token = $1;

-- name: UpsertProject :one
INSERT INTO irontask.projects (user_id, client_id, slug, name, color, encrypted_data, sync_version, deleted, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, 
    (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM irontask.projects WHERE user_id = $1), 
    $7, NOW())
ON CONFLICT (user_id, client_id) DO UPDATE
SET slug = EXCLUDED.slug,
    name = EXCLUDED.name,
    encrypted_data = EXCLUDED.encrypted_data,
    deleted = EXCLUDED.deleted,
    sync_version = (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM irontask.projects WHERE user_id = $1),
    updated_at = NOW()
RETURNING sync_version;

-- name: GetProjectsChanged :many
SELECT client_id, slug, name, 'project' as type, sync_version, encrypted_data, deleted
FROM irontask.projects
WHERE user_id = $1 AND sync_version > $2;

-- name: UpsertTask :one
INSERT INTO irontask.tasks (user_id, client_id, project_id, encrypted_content, status, priority, due_date, deleted, sync_version, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 
    (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM irontask.tasks WHERE user_id = $1),
    NOW())
ON CONFLICT (user_id, client_id) DO UPDATE
SET project_id = EXCLUDED.project_id,
    encrypted_content = EXCLUDED.encrypted_content,
    status = EXCLUDED.status,
    priority = EXCLUDED.priority,
    due_date = EXCLUDED.due_date,
    deleted = EXCLUDED.deleted,
    sync_version = (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM irontask.tasks WHERE user_id = $1),
    updated_at = NOW()
RETURNING sync_version;

-- name: GetTasksChanged :many
SELECT client_id, project_id, 'task' as type, sync_version, encrypted_content, status, priority, due_date, deleted
FROM irontask.tasks
WHERE user_id = $1 AND sync_version > $2;

-- name: ClearTasks :exec
DELETE FROM irontask.tasks WHERE user_id = $1;

-- name: ClearProjects :exec
DELETE FROM irontask.projects WHERE user_id = $1;
