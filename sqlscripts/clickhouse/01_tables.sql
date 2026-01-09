-- ClickHouse schema for edge-logs (APO Production Patterns)
-- Optimized for high-throughput iLogtail ingestion and edge computing workloads
-- Based on architecture.md APO design patterns

-- Database creation (optional - may already exist)
CREATE DATABASE IF NOT EXISTS edge_logs;
USE edge_logs;

-- Main logs table (iLogtail direct writes)
-- CRITICAL: APO Platform Patterns with proven production optimizations
CREATE TABLE IF NOT EXISTS logs (
    -- Time and data isolation
    timestamp          DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    dataset            LowCardinality(String) CODEC(ZSTD(1)),

    -- Log content
    content            String CODEC(ZSTD(1)),
    severity           LowCardinality(String) CODEC(ZSTD(1)),

    -- Container information
    container_id       String CODEC(ZSTD(1)),
    container_name     LowCardinality(String) CODEC(ZSTD(1)),
    pid                String CODEC(ZSTD(1)),

    -- Host information
    host_ip            LowCardinality(String) CODEC(ZSTD(1)),
    host_name          LowCardinality(String) CODEC(ZSTD(1)),

    -- K8s metadata
    k8s_namespace_name LowCardinality(String) CODEC(ZSTD(1)),
    k8s_pod_name       LowCardinality(String) CODEC(ZSTD(1)),
    k8s_pod_uid        String CODEC(ZSTD(1)),
    k8s_node_name      LowCardinality(String) CODEC(ZSTD(1)),

    -- Tags for analysis dimensions (cluster, region, etc.)
    tags               Map(String, String) CODEC(ZSTD(1)),

    -- Full-text search index
    INDEX idx_content content TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1,
    -- Tags index for cluster/region queries
    INDEX idx_tags tags TYPE bloom_filter GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY (dataset, toDate(timestamp))
ORDER BY (dataset, host_ip, timestamp)
TTL timestamp + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

-- Distributed table configuration for ClickHouse clustering (Future-ready)
-- This enables horizontal scaling across multiple ClickHouse nodes
-- Uncomment and configure when deploying in clustered environment
/*
CREATE TABLE IF NOT EXISTS logs_distributed AS logs
ENGINE = Distributed(edge_logs_cluster, edge_logs, logs, rand());
*/

-- Datasets table for multi-tenancy and data isolation management
CREATE TABLE IF NOT EXISTS datasets (
    name String,
    display_name String,
    description String,
    created_at DateTime DEFAULT now(),
    updated_at DateTime DEFAULT now(),
    retention_days UInt32 DEFAULT 30,
    is_active UInt8 DEFAULT 1

) ENGINE = MergeTree()
ORDER BY name;

-- Query statistics table for monitoring and performance analysis
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

-- Create materialized view for severity level statistics (aligned with new schema)
CREATE MATERIALIZED VIEW IF NOT EXISTS severity_stats_mv
TO severity_stats
AS SELECT
    dataset,
    k8s_namespace_name,
    severity,
    toStartOfHour(timestamp) as hour,
    count() as count
FROM logs
GROUP BY dataset, k8s_namespace_name, severity, hour;

-- Supporting table for materialized view
CREATE TABLE IF NOT EXISTS severity_stats (
    dataset String,
    k8s_namespace_name String,
    severity String,
    hour DateTime,
    count UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (dataset, k8s_namespace_name, severity, hour);