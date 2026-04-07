-- +goose Up
CREATE TABLE IF NOT EXISTS clusters (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 8006,
    api_token_id TEXT NOT NULL,
    api_token_secret TEXT NOT NULL,
    skip_tls_verify INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS llm_providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    provider_type TEXT NOT NULL,
    api_key TEXT NOT NULL,
    base_url TEXT,
    model TEXT NOT NULL,
    config TEXT,
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pipelines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    nodes TEXT NOT NULL DEFAULT '{}',
    edges TEXT NOT NULL DEFAULT '{}',
    viewport TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cron_jobs (
    id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    schedule TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    enabled INTEGER NOT NULL DEFAULT 1,
    last_run DATETIME,
    next_run DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS executions (
    id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    trigger_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    error TEXT,
    context TEXT
);

CREATE TABLE IF NOT EXISTS node_executions (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    node_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    input TEXT,
    output TEXT,
    error TEXT,
    started_at DATETIME,
    completed_at DATETIME
);

CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    actor_type TEXT NOT NULL,
    actor_id TEXT,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    details TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL,
    pipeline_data TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cron_jobs_pipeline ON cron_jobs(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_executions_pipeline ON executions(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_node_executions_execution ON node_executions(execution_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_log_created;
DROP INDEX IF EXISTS idx_node_executions_execution;
DROP INDEX IF EXISTS idx_executions_pipeline;
DROP INDEX IF EXISTS idx_cron_jobs_pipeline;
DROP TABLE IF EXISTS templates;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS node_executions;
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS cron_jobs;
DROP TABLE IF EXISTS pipelines;
DROP TABLE IF EXISTS llm_providers;
DROP TABLE IF EXISTS clusters;
