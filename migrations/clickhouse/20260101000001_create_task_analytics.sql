CREATE TABLE IF NOT EXISTS task_analytics (
    task_id UUID,
    event_type LowCardinality(String),  -- 'created', 'updated', 'deleted'
    status LowCardinality(String),      -- 'new', 'in_progress', 'done', 'cancelled'
    assignee String,
    timestamp DateTime64(3, 'UTC') DEFAULT now64()
) ENGINE = MergeTree()
ORDER BY (timestamp, task_id);