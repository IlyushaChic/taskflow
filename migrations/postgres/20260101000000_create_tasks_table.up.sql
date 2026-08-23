CREATE TABLE IF NOT EXISTS tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'in_progress', 'done', 'cancelled')),
    assignee    TEXT,
    due_date    TIMESTAMP WITH TIME ZONE,
    version     INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_tasks_active_unique ON tasks (title, assignee) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_status ON tasks (status);
CREATE INDEX idx_tasks_deleted_at ON tasks (deleted_at);