-- name: CreateProject :exec
-- sync_version is NULL for new items, will be set after successful push
INSERT INTO projects (id, slug, name, color, archived, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetProject :one
SELECT * FROM projects
WHERE id = ? AND deleted_at IS NULL LIMIT 1;

-- name: ListProjects :many
SELECT * FROM projects
WHERE deleted_at IS NULL
ORDER BY name;

-- name: DeleteProject :exec
-- Set sync_version to NULL to mark as "needs push". Server will assign new version.
UPDATE projects
SET deleted_at = ?, updated_at = ?, sync_version = NULL
WHERE id = ?;

-- name: UpdateProject :exec
-- Set sync_version to NULL to mark as "needs push". Server will assign new version.
UPDATE projects
SET slug = ?, name = ?, color = ?, updated_at = ?, sync_version = NULL
WHERE id = ?;

-- name: OverwriteProject :exec
UPDATE projects
SET slug = ?, name = ?, color = ?, updated_at = ?, sync_version = ?
WHERE id = ?;

-- name: OverwriteTask :exec
UPDATE tasks
SET project_id = ?, content = ?, status = ?, priority = ?, due_date = ?, tags = ?, updated_at = ?, sync_version = ?
WHERE id = ?;


-- name: CreateTask :exec
-- sync_version is NULL for new items, will be set after successful push
INSERT INTO tasks (id, project_id, content, status, priority, due_date, tags, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = ? AND deleted_at IS NULL LIMIT 1;

-- name: GetTaskPartial :one
SELECT * FROM tasks 
WHERE id LIKE ? || '%' AND deleted_at IS NULL LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE deleted_at IS NULL
  AND (sqlc.narg('project_id') IS NULL OR project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('show_all') OR status != 'done')
ORDER BY priority ASC, due_date ASC NULLS LAST, created_at DESC;

-- name: UpdateTask :exec
-- Set sync_version to NULL to mark as "needs push". Server will assign new version.
UPDATE tasks
SET project_id = ?, content = ?, status = ?, priority = ?, due_date = ?, tags = ?, updated_at = ?, sync_version = NULL
WHERE id = ?;

-- name: UpdateTaskStatus :exec
-- Set sync_version to NULL to mark as "needs push". Server will assign new version.
UPDATE tasks
SET status = ?, updated_at = ?, sync_version = NULL
WHERE id = ?;

-- name: DeleteTask :exec
-- Set sync_version to NULL to mark as "needs push". Server will assign new version.
UPDATE tasks
SET deleted_at = ?, updated_at = ?, sync_version = NULL
WHERE id = ?;


-- name: CountTasks :one
SELECT 
    COUNT(*) FILTER (WHERE status = 'process'),
    COUNT(*)
FROM tasks 
WHERE project_id = ? AND deleted_at IS NULL;

-- name: ClearTasks :exec
DELETE FROM tasks;

-- name: ClearProjects :exec
DELETE FROM projects;

-- name: GetProjectsToSync :many
-- Get projects that need to be pushed (sync_version is NULL means "dirty")
SELECT * FROM projects
WHERE sync_version IS NULL
ORDER BY updated_at;

-- name: GetTasksToSync :many
-- Get tasks that need to be pushed (sync_version is NULL means "dirty")
SELECT * FROM tasks
WHERE sync_version IS NULL
ORDER BY updated_at;

-- name: UpdateProjectSyncVersion :exec
UPDATE projects SET sync_version = ? WHERE id = ?;

-- name: UpdateTaskSyncVersion :exec
UPDATE tasks SET sync_version = ? WHERE id = ?;
