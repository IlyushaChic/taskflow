DROP INDEX IF EXISTS idx_tasks_active_unique;
DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_deleted_at;

DROP TABLE IF EXISTS tasks;