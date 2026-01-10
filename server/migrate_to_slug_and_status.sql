-- Migration: Add slug to projects and status/priority/due_date to tasks
-- Run this on your production database before deploying the new version

BEGIN;

-- 1. Add slug column to projects (with default based on client_id for existing data)
ALTER TABLE irontask.projects 
ADD COLUMN IF NOT EXISTS slug TEXT;

-- Populate slug from client_id for existing projects (temporary)
UPDATE irontask.projects 
SET slug = client_id 
WHERE slug IS NULL;

-- Now make it NOT NULL and add constraints
ALTER TABLE irontask.projects 
ALTER COLUMN slug SET NOT NULL;

ALTER TABLE irontask.projects 
ADD CONSTRAINT projects_user_slug_unique UNIQUE(user_id, slug);

CREATE INDEX IF NOT EXISTS idx_projects_slug ON irontask.projects(user_id, slug);

-- 2. Add status, priority, due_date columns to tasks
ALTER TABLE irontask.tasks 
ADD COLUMN IF NOT EXISTS encrypted_content BYTEA;

ALTER TABLE irontask.tasks 
ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'process';

ALTER TABLE irontask.tasks 
ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 4;

ALTER TABLE irontask.tasks 
ADD COLUMN IF NOT EXISTS due_date TEXT;

-- Copy encrypted_data to encrypted_content for existing tasks
UPDATE irontask.tasks 
SET encrypted_content = encrypted_data 
WHERE encrypted_content IS NULL AND encrypted_data IS NOT NULL;

-- Create index on status
CREATE INDEX IF NOT EXISTS idx_tasks_status ON irontask.tasks(user_id, status);

-- 3. Optional: Drop old encrypted_data column after verifying migration
-- ALTER TABLE irontask.tasks DROP COLUMN encrypted_data;

COMMIT;

-- Verify migration
SELECT 
  'projects' as table_name,
  COUNT(*) as total_rows,
  COUNT(slug) as rows_with_slug
FROM irontask.projects
UNION ALL
SELECT 
  'tasks' as table_name,
  COUNT(*) as total_rows,
  COUNT(encrypted_content) as rows_with_encrypted_content
FROM irontask.tasks;
