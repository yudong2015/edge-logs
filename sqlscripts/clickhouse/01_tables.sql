-- ClickHouse schema for edge-logs
-- Table definitions for log storage and metadata

-- Database creation (optional - may already exist)
CREATE DATABASE IF NOT EXISTS edge_logs;
USE edge_logs;

-- Main log entries table
-- Optimized for high-throughput log ingestion and time-based queries
CREATE TABLE IF NOT EXISTS logs (
    -- Primary fields
    timestamp DateTime64(3, 'UTC') CODEC(Delta, ZSTD),
    dataset LowCardinality(String) CODEC(ZSTD),
    namespace LowCardinality(String) CODEC(ZSTD),
    pod_name String CODEC(ZSTD),
    container_name LowCardinality(String) CODEC(ZSTD),
    log_level LowCardinality(String) CODEC(ZSTD),
    message String CODEC(ZSTD),

    -- Kubernetes metadata
    node_name LowCardinality(String) CODEC(ZSTD),
    cluster_name LowCardinality(String) CODEC(ZSTD),

    -- Additional metadata (JSON for flexibility)
    labels Map(String, String) CODEC(ZSTD),
    annotations Map(String, String) CODEC(ZSTD),

    -- Technical fields
    source_host String CODEC(ZSTD),
    ingestion_time DateTime DEFAULT now() CODEC(Delta, ZSTD),

    -- Log parsing fields
    structured_data Map(String, String) CODEC(ZSTD),

    -- Row identifier
    id UUID DEFAULT generateUUIDv4()

) ENGINE = MergeTree()
PARTITION BY (dataset, toYYYYMM(timestamp))
ORDER BY (dataset, timestamp, namespace, pod_name)
TTL timestamp + INTERVAL 90 DAY DELETE
SETTINGS
    index_granularity = 8192,
    merge_with_ttl_timeout = 3600,
    old_parts_lifetime = 7200;

-- Datasets table for multi-tenancy
CREATE TABLE IF NOT EXISTS datasets (
    name String,
    display_name String,
    description String,
    created_at DateTime DEFAULT now(),
    updated_at DateTime DEFAULT now(),
    retention_days UInt32 DEFAULT 90,
    is_active UInt8 DEFAULT 1

) ENGINE = MergeTree()
ORDER BY name;

-- Kubernetes resources metadata cache
-- Stores enrichment data from K8s API
CREATE TABLE IF NOT EXISTS k8s_resources (
    cluster_name LowCardinality(String),
    namespace LowCardinality(String),
    resource_type LowCardinality(String), -- 'pod', 'deployment', 'service', etc.
    resource_name String,
    labels Map(String, String),
    annotations Map(String, String),
    created_at DateTime,
    updated_at DateTime DEFAULT now(),
    raw_spec String -- JSON representation of full resource

) ENGINE = ReplacingMergeTree(updated_at)
PARTITION BY (cluster_name, resource_type)
ORDER BY (cluster_name, namespace, resource_type, resource_name)
TTL updated_at + INTERVAL 7 DAY DELETE;

-- Query statistics table for monitoring
CREATE TABLE IF NOT EXISTS query_stats (
    query_id String,
    dataset String,
    user_id String DEFAULT '',
    query_type LowCardinality(String), -- 'search', 'aggregation', 'export'
    query_params String, -- JSON
    execution_time_ms UInt32,
    rows_examined UInt64,
    rows_returned UInt64,
    created_at DateTime DEFAULT now()

) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (dataset, created_at)
TTL created_at + INTERVAL 30 DAY DELETE;

-- Insert default dataset
INSERT INTO datasets (name, display_name, description) VALUES
('default', 'Default Dataset', 'Default dataset for all logs')
ON DUPLICATE KEY UPDATE updated_at = now();

-- Create materialized view for log level statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS log_level_stats_mv
TO log_level_stats
AS SELECT
    dataset,
    namespace,
    log_level,
    toStartOfHour(timestamp) as hour,
    count() as count
FROM logs
GROUP BY dataset, namespace, log_level, hour;

-- Supporting table for materialized view
CREATE TABLE IF NOT EXISTS log_level_stats (
    dataset String,
    namespace String,
    log_level String,
    hour DateTime,
    count UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (dataset, namespace, log_level, hour);

-- Performance optimization: Create skipping indexes
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_message_bloom message TYPE bloom_filter GRANULARITY 1;
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_pod_name_set pod_name TYPE set(1000) GRANULARITY 1;
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_container_name_set container_name TYPE set(100) GRANULARITY 1;