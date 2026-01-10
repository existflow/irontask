CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#4ECDC4',
    archived INTEGER DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    sync_version INTEGER DEFAULT 0
);

CREATE INDEX idx_projects_slug ON projects(slug);

CREATE TABLE tasks (
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
    sync_version INTEGER DEFAULT 0,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX idx_tasks_project ON tasks(project_id);
CREATE INDEX idx_tasks_status ON tasks(status);

CREATE TABLE sync_state (
    key TEXT PRIMARY KEY,
    value TEXT
);
