package db

import "fmt"

// migrate runs all database migrations
func (db *DB) migrate() error {
	migrations := []string{
		migrationCreateProjects,
		migrationCreateTasks,
		migrationCreateSyncState,
		migrationInsertInbox,
	}

	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
}

const migrationCreateProjects = `
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#4ECDC4',
    archived INTEGER DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    sync_version INTEGER DEFAULT 0
);
`

const migrationCreateTasks = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL DEFAULT 'inbox',
    content TEXT NOT NULL,
    done INTEGER DEFAULT 0,
    priority INTEGER DEFAULT 4,
    due_date TEXT,
    tags TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    sync_version INTEGER DEFAULT 0,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_done ON tasks(done);
`

const migrationCreateSyncState = `
CREATE TABLE IF NOT EXISTS sync_state (
    key TEXT PRIMARY KEY,
    value TEXT
);
`

const migrationInsertInbox = `
INSERT OR IGNORE INTO projects (id, name, color, created_at, updated_at)
VALUES ('inbox', 'Inbox', '#6C757D', datetime('now'), datetime('now'));
`
