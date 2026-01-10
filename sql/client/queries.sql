-- name: CreateProject :exec
INSERT INTO projects (id, name, color, archived, created_at, updated_at, sync_version)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetProject :one
SELECT * FROM projects
WHERE id = ? AND deleted_at IS NULL LIMIT 1;

-- name: ListProjects :many
SELECT * FROM projects
WHERE deleted_at IS NULL
ORDER BY name;

-- name: DeleteProject :exec
UPDATE projects 
SET deleted_at = ?, updated_at = ?, sync_version = sync_version + 1
WHERE id = ?;

-- name: CreateTask :exec
INSERT INTO tasks (id, project_id, content, done, priority, due_date, tags, created_at, updated_at, sync_version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

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
  AND (sqlc.narg('include_done') OR done = 0)
ORDER BY priority ASC, due_date ASC NULLS LAST, created_at DESC;

-- name: UpdateTask :exec
UPDATE tasks
SET project_id = ?, content = ?, done = ?, priority = ?, due_date = ?, tags = ?, updated_at = ?, sync_version = sync_version + 1
WHERE id = ?;

-- name: MarkTaskDone :exec
UPDATE tasks
SET done = ?, updated_at = ?, sync_version = sync_version + 1
WHERE id = ?;

-- name: DeleteTask :exec
UPDATE tasks 
SET deleted_at = ?, updated_at = ?, sync_version = sync_version + 1
WHERE id = ?;

-- name: CountTasks :one
SELECT 
    COUNT(*) FILTER (WHERE done = 0),
    COUNT(*)
FROM tasks 
WHERE project_id = ? AND deleted_at IS NULL;

-- name: ClearTasks :exec
DELETE FROM tasks;

-- name: ClearProjects :exec
DELETE FROM projects;

-- name: GetProjectsToSync :many
SELECT * FROM projects
WHERE sync_version > ?
ORDER BY updated_at;

-- name: GetTasksToSync :many
SELECT * FROM tasks
WHERE sync_version > ?
ORDER BY updated_at;

-- name: UpdateProjectSyncVersion :exec
UPDATE projects SET sync_version = ? WHERE id = ?;

-- name: UpdateTaskSyncVersion :exec
UPDATE tasks SET sync_version = ? WHERE id = ?;
