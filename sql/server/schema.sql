-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS irontask;

-- Set search path for this session
SET search_path TO irontask, public;

-- Global sync version sequence (shared across all users and entities)
CREATE SEQUENCE IF NOT EXISTS irontask.sync_version_seq START 1;

CREATE TABLE IF NOT EXISTS irontask.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS irontask.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES irontask.users(id),
    token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON irontask.sessions(token);

CREATE TABLE IF NOT EXISTS irontask.magic_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS irontask.projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES irontask.users(id),
    client_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#4ECDC4',
    encrypted_data BYTEA,
    sync_version BIGINT DEFAULT nextval('irontask.sync_version_seq'),
    deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    client_updated_at TIMESTAMP,  -- Timestamp from client for conflict detection
    UNIQUE(user_id, client_id),
    UNIQUE(user_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_projects_user ON irontask.projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_sync ON irontask.projects(user_id, sync_version);
CREATE INDEX IF NOT EXISTS idx_projects_slug ON irontask.projects(user_id, slug);

CREATE TABLE IF NOT EXISTS irontask.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES irontask.users(id),
    client_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    type TEXT DEFAULT 'task',  -- For sync item type identification
    encrypted_content BYTEA,
    status TEXT DEFAULT 'process',
    priority INTEGER DEFAULT 4,
    due_date TEXT,
    sync_version BIGINT DEFAULT nextval('irontask.sync_version_seq'),
    deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    client_updated_at TIMESTAMP,  -- Timestamp from client for conflict detection
    UNIQUE(user_id, client_id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_user ON irontask.tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_sync ON irontask.tasks(user_id, sync_version);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON irontask.tasks(user_id, status);

