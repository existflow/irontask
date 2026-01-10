package server

// migrate runs database migrations
func (s *Server) migrate() error {
	// Create schema first
	if _, err := s.db.Exec("CREATE SCHEMA IF NOT EXISTS irontask;"); err != nil {
		return err
	}

	migrations := []string{
		migrationUsers,
		migrationSessions,
		migrationMagicLinks,
		migrationProjects,
		migrationTasks,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return err
		}
	}

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
    name TEXT NOT NULL,
    color TEXT DEFAULT '#4ECDC4',
    encrypted_data BYTEA,
    sync_version BIGINT DEFAULT 0,
    deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, client_id)
);

CREATE INDEX IF NOT EXISTS idx_projects_user ON irontask.projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_sync ON irontask.projects(user_id, sync_version);
`

const migrationTasks = `
CREATE TABLE IF NOT EXISTS irontask.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES irontask.users(id),
    client_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    encrypted_data BYTEA,
    sync_version BIGINT DEFAULT 0,
    deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, client_id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_user ON irontask.tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_sync ON irontask.tasks(user_id, sync_version);
`
