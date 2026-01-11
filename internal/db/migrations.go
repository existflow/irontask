package db

import "fmt"

// migrate runs all database migrations
func (db *DB) migrate() error {
	migrations := []string{
		migrationCreateProjects,
		migrationCreateTasks,
		migrationCreateSyncState,
		migrationServerSideSyncVersion, // v2: Server-side sync versioning
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
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#4ECDC4',
    archived INTEGER DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    sync_version INTEGER  -- NULL means "needs sync"
);

CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug);

-- Create default inbox project
INSERT OR IGNORE INTO projects (id, slug, name, color, created_at, updated_at)
VALUES ('inbox', 'inbox', 'Inbox', '#6C757D', datetime('now'), datetime('now'));
`

const migrationCreateTasks = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT DEFAULT 'process',
    priority INTEGER DEFAULT 4,
    due_date TEXT,
    tags TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    sync_version INTEGER,  -- NULL means "needs sync"
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
`

const migrationCreateSyncState = `
CREATE TABLE IF NOT EXISTS sync_state (
    key TEXT PRIMARY KEY,
    value TEXT
);
`

// migrationServerSideSyncVersion updates existing databases for server-side sync versioning.
// - Sets sync_version to NULL for all items to force re-sync
// - NULL means "needs to be pushed", server assigns version after push
// - Creates inbox project if it doesn't exist (for existing databases)
const migrationServerSideSyncVersion = `
-- Reset sync_version to NULL for items that haven't been synced properly
-- Items with sync_version = 0 were never synced (old default)
-- After this migration, sync_version IS NULL means "dirty/needs push"
UPDATE projects SET sync_version = NULL WHERE sync_version = 0 OR sync_version IS NULL;
UPDATE tasks SET sync_version = NULL WHERE sync_version = 0 OR sync_version IS NULL;

-- Ensure inbox project exists for existing databases
INSERT OR IGNORE INTO projects (id, slug, name, color, created_at, updated_at)
VALUES ('inbox', 'inbox', 'Inbox', '#6C757D', datetime('now'), datetime('now'));
`
