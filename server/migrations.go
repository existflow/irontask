package server

import "github.com/existflow/irontask/internal/logger"

// migrate runs database migrations
func (s *Server) migrate() error {
	// Create schema first
	logger.Debug("Creating schema if not exists")
	if _, err := s.db.Exec("CREATE SCHEMA IF NOT EXISTS irontask;"); err != nil {
		return err
	}

	migrationNames := []string{
		"users",
		"sessions",
		"magic_links",
		"projects",
		"tasks",
		"server_side_sync_version",
	}

	migrations := []string{
		migrationUsers,
		migrationSessions,
		migrationMagicLinks,
		migrationProjects,
		migrationTasks,
		migrationServerSideSyncVersion, // v2: Server-side sync versioning
	}

	for i, m := range migrations {
		logger.Debug("Running migration", logger.F("name", migrationNames[i]))
		if _, err := s.db.Exec(m); err != nil {
			logger.Error("Migration failed", logger.F("name", migrationNames[i]), logger.F("error", err))
			return err
		}
	}

	logger.Info("All migrations applied", logger.F("count", len(migrations)))
	return nil
}

const migrationUsers = `
CREATE TABLE IF NOT EXISTS irontask.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
`

const migrationSessions = `
CREATE TABLE IF NOT EXISTS irontask.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES irontask.users(id),
    token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON irontask.sessions(token);
`

const migrationMagicLinks = `
CREATE TABLE IF NOT EXISTS irontask.magic_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);
`

const migrationProjects = `
CREATE TABLE IF NOT EXISTS irontask.projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES irontask.users(id),
    client_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#4ECDC4',
    encrypted_data BYTEA,
    sync_version BIGINT DEFAULT 0,
    deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, client_id),
    UNIQUE(user_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_projects_user ON irontask.projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_sync ON irontask.projects(user_id, sync_version);
CREATE INDEX IF NOT EXISTS idx_projects_slug ON irontask.projects(user_id, slug);
`

const migrationTasks = `
CREATE TABLE IF NOT EXISTS irontask.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES irontask.users(id),
    client_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    encrypted_content BYTEA,
    status TEXT DEFAULT 'process',
    priority INTEGER DEFAULT 4,
    due_date TEXT,
    sync_version BIGINT DEFAULT 0,
    deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, client_id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_user ON irontask.tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_sync ON irontask.tasks(user_id, sync_version);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON irontask.tasks(user_id, status);
`

// migrationServerSideSyncVersion adds server-side sync versioning support:
// - Creates global sync_version_seq sequence
// - Adds client_updated_at column for conflict detection
// - Adds type column to tasks
// - Updates defaults to use sequence
const migrationServerSideSyncVersion = `
-- Create global sync version sequence if not exists
DO $$
DECLARE
    max_project_version BIGINT;
    max_task_version BIGINT;
    start_value BIGINT;
BEGIN
    -- Get max versions from existing tables
    SELECT COALESCE(MAX(sync_version), 0) INTO max_project_version FROM irontask.projects;
    SELECT COALESCE(MAX(sync_version), 0) INTO max_task_version FROM irontask.tasks;

    -- Start sequence from max + 1
    start_value := GREATEST(max_project_version, max_task_version) + 1;

    -- Create or update sequence
    IF NOT EXISTS (SELECT 1 FROM pg_sequences WHERE schemaname = 'irontask' AND sequencename = 'sync_version_seq') THEN
        EXECUTE format('CREATE SEQUENCE irontask.sync_version_seq START %s', start_value);
    END IF;
END $$;

-- Add client_updated_at column to projects if not exists
ALTER TABLE irontask.projects ADD COLUMN IF NOT EXISTS client_updated_at TIMESTAMP;

-- Add client_updated_at column to tasks if not exists
ALTER TABLE irontask.tasks ADD COLUMN IF NOT EXISTS client_updated_at TIMESTAMP;

-- Add type column to tasks if not exists
ALTER TABLE irontask.tasks ADD COLUMN IF NOT EXISTS type TEXT DEFAULT 'task';

-- Update default for sync_version to use sequence (for new rows)
ALTER TABLE irontask.projects ALTER COLUMN sync_version SET DEFAULT nextval('irontask.sync_version_seq');
ALTER TABLE irontask.tasks ALTER COLUMN sync_version SET DEFAULT nextval('irontask.sync_version_seq');

-- Set client_updated_at to updated_at for existing records that don't have it
UPDATE irontask.projects SET client_updated_at = updated_at WHERE client_updated_at IS NULL;
UPDATE irontask.tasks SET client_updated_at = updated_at WHERE client_updated_at IS NULL;
`
