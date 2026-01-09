-- Performance indexes and optimizations for edge-logs ClickHouse schema
-- APO production patterns for high-performance edge computing workloads
-- Aligned with new schema field names and iLogtail integration requirements

USE edge_logs;

-- Projection for namespace-based queries (aligned with k8s_namespace_name)
-- Significantly improves performance for K8s namespace filtering
ALTER TABLE logs ADD PROJECTION IF NOT EXISTS proj_k8s_namespace (
    SELECT *
    ORDER BY (k8s_namespace_name, timestamp, k8s_pod_name)
);

-- Projection for pod-based queries (aligned with k8s_pod_name)
-- Optimizes queries filtering by specific pods
ALTER TABLE logs ADD PROJECTION IF NOT EXISTS proj_k8s_pod (
    SELECT *
    ORDER BY (k8s_pod_name, timestamp, k8s_namespace_name)
);

-- Projection for severity level aggregations (aligned with severity field)
-- Speeds up log level analytics and alerting queries
ALTER TABLE logs ADD PROJECTION IF NOT EXISTS proj_severity (
    SELECT
        dataset,
        k8s_namespace_name,
        severity,
        toStartOfMinute(timestamp) as minute,
        count() as count
    GROUP BY dataset, k8s_namespace_name, severity, minute
    ORDER BY (dataset, severity, minute)
);

-- Set index for container names (limited cardinality)
-- Improves container-specific filtering
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_container_name_set
    container_name TYPE set(500) GRANULARITY 1;

-- Set index for node names (limited cardinality)
-- Optimizes node-based queries and troubleshooting
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_k8s_node_set
    k8s_node_name TYPE set(1000) GRANULARITY 1;

-- Set index for host IP addresses (limited cardinality in edge environments)
-- Optimizes host-specific log queries
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_host_ip_set
    host_ip TYPE set(1000) GRANULARITY 1;

-- MinMax index for timestamp range queries
-- Provides faster timestamp boundary checks with millisecond precision
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_timestamp_minmax
    timestamp TYPE minmax GRANULARITY 1;

-- Tokenbf_v1 index for pod name pattern matching
-- Enables efficient K8s pod name pattern searches
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_k8s_pod_pattern
    k8s_pod_name TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1;

-- Additional bloom_filter index for tags map keys/values
-- Optimizes cluster and region-based queries via tags
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_tags_bloom
    tags TYPE bloom_filter GRANULARITY 1;

-- Ngrambf_v1 index for partial content matching
-- Enables efficient substring search in log content
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_content_ngram
    content TYPE ngrambf_v1(3, 256, 2, 0) GRANULARITY 1;

-- Set index for severity levels (very limited cardinality)
-- Optimizes severity-based filtering
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_severity_set
    severity TYPE set(20) GRANULARITY 1;

-- Build projections (materialize after initial data load for better performance)
-- Uncomment the following lines after initial data load:

-- ALTER TABLE logs MATERIALIZE PROJECTION proj_k8s_namespace;
-- ALTER TABLE logs MATERIALIZE PROJECTION proj_k8s_pod;
-- ALTER TABLE logs MATERIALIZE PROJECTION proj_severity;

-- Performance tuning: Optimize merge tree settings for edge computing workloads
-- Configured for high-frequency small writes from edge nodes
ALTER TABLE logs MODIFY SETTING
    max_parts_in_total = 10000,                              -- Maximum number of parts before merge
    max_bytes_to_merge_at_max_space_in_pool = 161061273600,  -- 150GB max merge size
    merge_with_ttl_timeout = 14400,                          -- TTL merge timeout (4 hours)
    max_replicated_merges_in_queue = 100,                    -- Max concurrent merges
    -- Edge-specific optimizations for high-frequency small writes
    parts_to_throw_insert = 3000,                            -- Reject inserts when too many parts
    parts_to_delay_insert = 1000,                            -- Start delaying inserts threshold
    max_insert_delayed_streams_for_parallel_write = 1000;    -- Parallel write optimization

-- Create dictionary for dataset metadata (for very frequent dataset lookups)
-- Uncomment when needed for high-performance dataset validation:
-- CREATE DICTIONARY IF NOT EXISTS dataset_dict (
--     name String,
--     display_name String,
--     retention_days UInt32
-- )
-- PRIMARY KEY name
-- SOURCE(CLICKHOUSE(TABLE 'datasets'))
-- LAYOUT(HASHED())
-- LIFETIME(MIN 300 MAX 3600);

-- Final optimization pass after index creation
-- This may take time on large datasets but is essential for performance
OPTIMIZE TABLE logs;