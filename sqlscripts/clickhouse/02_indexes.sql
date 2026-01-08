-- Additional indexes and performance optimizations for edge-logs ClickHouse schema
-- These indexes improve query performance for common access patterns

USE edge_logs;

-- Projection for namespace-based queries
-- Significantly improves performance for namespace filtering
ALTER TABLE logs ADD PROJECTION IF NOT EXISTS proj_namespace (
    SELECT *
    ORDER BY (namespace, timestamp, pod_name)
);

-- Projection for pod-based queries
-- Optimizes queries filtering by specific pods
ALTER TABLE logs ADD PROJECTION IF NOT EXISTS proj_pod (
    SELECT *
    ORDER BY (pod_name, timestamp, namespace)
);

-- Projection for log level aggregations
-- Speeds up log level analytics and alerting queries
ALTER TABLE logs ADD PROJECTION IF NOT EXISTS proj_log_level (
    SELECT
        dataset,
        namespace,
        log_level,
        toStartOfMinute(timestamp) as minute,
        count() as count
    GROUP BY dataset, namespace, log_level, minute
    ORDER BY (dataset, log_level, minute)
);

-- Bloom filter index on message content for full-text search
-- Enables fast text search across log messages
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_message_bloom_large
    message TYPE bloom_filter(0.01) GRANULARITY 1;

-- Set index for container names (limited cardinality)
-- Improves container-specific filtering
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_container_set
    container_name TYPE set(500) GRANULARITY 1;

-- Set index for node names (limited cardinality)
-- Optimizes node-based queries and troubleshooting
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_node_set
    node_name TYPE set(1000) GRANULARITY 1;

-- MinMax index for timestamp range queries
-- Provides faster timestamp boundary checks
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_timestamp_minmax
    timestamp TYPE minmax GRANULARITY 1;

-- Tokenbf_v1 index for pod name pattern matching
-- Enables efficient pod name pattern searches
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_pod_pattern
    pod_name TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1;

-- Set index for cluster names
-- Improves multi-cluster query performance
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_cluster_set
    cluster_name TYPE set(100) GRANULARITY 1;

-- Ngrambf_v1 index for partial message matching
-- Enables efficient substring search in log messages
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_message_ngram
    message TYPE ngrambf_v1(3, 256, 2, 0) GRANULARITY 1;

-- For k8s_resources table - optimize resource lookups
ALTER TABLE k8s_resources ADD INDEX IF NOT EXISTS idx_resource_bloom
    resource_name TYPE bloom_filter GRANULARITY 1;

ALTER TABLE k8s_resources ADD INDEX IF NOT EXISTS idx_labels_map
    labels TYPE bloom_filter GRANULARITY 1;

-- Build projections (this may take some time on large datasets)
-- Uncomment the following lines after initial data load for better performance:

-- ALTER TABLE logs MATERIALIZE PROJECTION proj_namespace;
-- ALTER TABLE logs MATERIALIZE PROJECTION proj_pod;
-- ALTER TABLE logs MATERIALIZE PROJECTION proj_log_level;

-- Performance tuning: Optimize merge tree settings for write-heavy workloads
ALTER TABLE logs MODIFY SETTING
    max_parts_in_total = 10000,
    max_bytes_to_merge_at_max_space_in_pool = 161061273600,
    merge_with_ttl_timeout = 14400,
    max_replicated_merges_in_queue = 100;

-- Create dictionary for dataset metadata (if needed for very frequent lookups)
-- CREATE DICTIONARY IF NOT EXISTS dataset_dict (
--     name String,
--     display_name String,
--     retention_days UInt32
-- )
-- PRIMARY KEY name
-- SOURCE(CLICKHOUSE(TABLE 'datasets'))
-- LAYOUT(HASHED())
-- LIFETIME(MIN 300 MAX 3600);

-- Optimize table after index creation
OPTIMIZE TABLE logs;