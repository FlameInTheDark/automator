-- +goose Up
CREATE TABLE IF NOT EXISTS kubernetes_clusters (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    kubeconfig TEXT NOT NULL,
    context_name TEXT NOT NULL,
    default_namespace TEXT NOT NULL DEFAULT 'default',
    server TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_kubernetes_clusters_name ON kubernetes_clusters(name);

-- +goose Down
DROP INDEX IF EXISTS idx_kubernetes_clusters_name;
DROP TABLE IF EXISTS kubernetes_clusters;
