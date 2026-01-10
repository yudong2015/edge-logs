# Performance Tuning Guide for edge-logs

## Overview

This guide provides comprehensive performance optimization strategies for edge-logs to ensure queries complete in under 2 seconds (NFR1 requirement) and maintain optimal performance as data volumes grow.

## Performance Targets (NFR1)

### Query Response Time Requirements

- **Basic Queries**: < 500ms (simple time + dataset queries)
- **Filtered Queries**: < 1s (queries with multiple filters)
- **Aggregation Queries**: < 1.5s (complex multi-dimensional aggregations)
- **Metadata-Enriched Queries**: < 2s (queries with K8s API enrichment)

## Architecture Optimization

### 1. Database Layer Optimization

#### ClickHouse Query Optimization

**Schema Design:**
```sql
-- Optimize table partitioning for time-based queries
ALTER TABLE logs MODIFY PARTITION BY toDateTime(timestamp);

-- Create proper indexes for common query patterns
ALTER TABLE logs ADD INDEX idx_namespace (namespace) TYPE minmax GRANULARITY 4;
ALTER TABLE logs ADD INDEX idx_severity (severity) TYPE set(100) GRANULARITY 4;
```

**Query Optimization Techniques:**
```sql
-- Use PREWHERE for early filtering (more efficient than WHERE)
SELECT timestamp, namespace, pod_name, message
FROM logs
PREWHERE timestamp >= now() - INTERVAL 1 HOUR
WHERE namespace = 'default'

-- Avoid SELECT * - specify only required columns
SELECT timestamp, namespace, pod_name, message, severity
FROM logs
WHERE timestamp >= now() - INTERVAL 1 HOUR

-- Add LIMIT to prevent large result sets
SELECT timestamp, namespace, pod_name, message
FROM logs
WHERE timestamp >= now() - INTERVAL 1 HOUR
LIMIT 10000
```

**Connection Pool Configuration:**
```go
// Optimal connection pool settings
config := &PoolConfig{
    MaxOpenConns:    25,    // Total connections
    MaxIdleConns:    10,    // Idle connections
    ConnMaxLifetime: 1 * time.Hour,
    ConnMaxIdleTime: 10 * time.Minute,
    HealthCheckFreq: 30 * time.Second,
}
```

### 2. Application Layer Optimization

#### Query Optimization Service

**Enable Query Optimizer:**
```go
optimizer := optimization.NewQueryOptimizer()

// Configure optimizer settings
optimizer.SetMaxResultRows(100000)  // Limit result sets
optimizer.EnablePrewhere(true)      // Enable PREWHERE optimization
optimizer.EnableColumnPruning(true) // Avoid SELECT *
```

**Pagination Configuration:**
```go
paginationMgr := optimization.NewPaginationManager()

// Set memory limits
paginationMgr.SetMaxResultSize(100 * 1024 * 1024) // 100MB limit

// Configure page sizes
paginationMgr.SetDefaultPageSize(100)
paginationMgr.SetMaxPageSize(10000)
```

#### Performance Monitoring

**Initialize Metrics:**
```go
// Create performance metrics
metrics := metrics.NewQueryPerformanceMetrics()

// Create performance monitor
performanceMon := metrics.NewPerformanceMonitor(metrics)

// Enable monitoring
performanceMon.Enable()
```

**Monitor Query Execution:**
```go
// Wrap query execution with monitoring
result, err := performanceMon.MonitorQueryExecution(
    ctx,
    metrics.QueryTypeFiltered,
    "dataset-name",
    queryParams,
    func() (interface{}, error) {
        return queryService.QueryLogs(ctx, req)
    },
)
```

### 3. Metadata Enrichment Optimization

#### Caching Strategy

**Enable Metadata Caching:**
```go
config := enrichment.DefaultEnrichmentConfig()

// Optimize cache settings
config.CacheTTL = 5 * time.Minute  // Cache for 5 minutes
config.EnableInformer = true       // Enable K8s informers

// Create enrichment optimizer
optimizer := enrichment.NewEnrichmentOptimizer(service, metrics, config)
```

**Batch Processing:**
```go
// Enable batch processing for multiple requests
optimizerConfig := &enrichment.OptimizerConfig{
    EnableBatching:        true,
    BatchSize:             25,
    EnableParallelism:      true,
    MaxConcurrentAPICalls: 10,
}
```

#### K8s API Optimization

**Connection Pooling:**
```go
// Use client-go connection pooling
clientConfig := &rest.Config{
    QPS:   50,   // Queries per second
    Burst: 100,  // Burst capacity
}
```

**API Call Batching:**
```go
// Batch multiple metadata requests
batchRequest := &enrichment.EnrichmentRequest{
    PodUIDs: []string{"uid1", "uid2", "uid3", ...},
    IncludeLabels: true,
}

result, err := optimizer.OptimizeEnrichment(ctx, batchRequest)
```

## Performance Monitoring

### Prometheus Metrics

**Key Metrics to Monitor:**

```promql
# Query duration by type
rate(edge_logs_query_duration_seconds_sum[5m]) by (query_type)
/
rate(edge_logs_query_duration_seconds_count[5m]) by (query_type)

# Query success rate
edge_logs_query_success_rate

# Slow query count
rate(edge_logs_slow_queries_total[5m])

# Connection pool utilization
edge_logs_connection_pool_stats{stat_type="in_use"}

# Cache hit rate
edge_logs_enrichment_cache_hit_rate
```

### Performance Alerts

**Recommended Alerting Rules:**

```yaml
# Query performance alerts
- alert: SlowQueryDetected
  expr: edge_logs_query_duration_seconds{quantile="0.95"} > 1.5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Slow queries detected (p95 > 1.5s)"

- alert: HighErrorRate
  expr: rate(edge_logs_query_errors_total[5m]) > 0.05
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "High query error rate (>5%)"

- alert: ConnectionPoolExhausted
  expr: edge_logs_connection_pool_stats{stat_type="in_use"} / edge_logs_connection_pool_stats{stat_type="open"} > 0.8
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Connection pool utilization >80%"

- alert: CacheMissRateHigh
  expr: rate(edge_logs_enrichment_cache_misses_total[5m]) > 10
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High enrichment cache miss rate"
```

## Performance Tuning Scenarios

### Scenario 1: High Query Load

**Symptoms:**
- Increased query response times
- Connection pool exhaustion
- High CPU/memory usage

**Solutions:**
1. **Increase Connection Pool Size:**
   ```go
   config.MaxOpenConns = 50  // Increase from 25
   ```

2. **Enable Query Caching:**
   ```go
   config.CacheTTL = 10 * time.Minute  // Increase cache TTL
   ```

3. **Optimize Database Queries:**
   ```sql
   -- Add materialized views for common queries
   CREATE MATERIALIZED VIEW logs_hourly_stats
   ENGINE = SummingMergeTree()
   ORDER BY (namespace, hour)
   AS SELECT
       toStartOfHour(timestamp) as hour,
       namespace,
       count() as log_count
   FROM logs
   GROUP BY namespace, hour;
   ```

### Scenario 2: Large Dataset Queries

**Symptoms:**
- Memory pressure
- Slow query response times
- Query timeouts

**Solutions:**
1. **Implement Result Pagination:**
   ```go
   req.PageSize = 100  // Reduce page size
   ```

2. **Add Time Range Limits:**
   ```sql
   -- Enforce maximum time range
   WHERE timestamp >= now() - INTERVAL 24 HOUR
   ```

3. **Enable Result Streaming:**
   ```go
   paginationMgr.EnableStreaming(true)
   paginationMgr.SetStreamingChunkSize(1000)
   ```

### Scenario 3: Metadata Enrichment Bottleneck

**Symptoms:**
- Slow metadata-enriched queries
- High K8s API call rate
- Cache misses

**Solutions:**
1. **Optimize Cache Strategy:**
   ```go
   config.CacheTTL = 15 * time.Minute  // Increase cache duration
   config.EnableInformer = true         // Enable informers
   ```

2. **Enable Batch Processing:**
   ```go
   optimizerConfig.EnableBatching = true
   optimizerConfig.BatchSize = 50
   ```

3. **Implement Parallel Processing:**
   ```go
   optimizerConfig.EnableParallelism = true
   optimizerConfig.MaxConcurrentAPICalls = 20
   ```

## Performance Testing

### Load Testing

**Concurrent Query Test:**
```go
// Simulate concurrent query load
func TestConcurrentQueryLoad(t *testing.T) {
    concurrency := 50
    queries := make(chan int, concurrency)

    startTime := time.Now()

    for i := 0; i < concurrency; i++ {
        go func(queryID int) {
            // Execute query
            queries <- queryID
        }(i)
    }

    // Wait for completion
    for i := 0; i < concurrency; i++ {
        <-queries
    }

    totalTime := time.Since(startTime)
    avgTime := totalTime / time.Duration(concurrency)

    assert.True(t, avgTime < 2*time.Second,
        "Average query time must be <2s under load")
}
```

### Performance Benchmarking

**Run Performance Tests:**
```bash
# Run NFR1 compliance tests
go test ./test/performance/... -v -run TestNFR1Compliance

# Run concurrent load tests
go test ./test/performance/... -v -run TestConcurrentQueryLoad

# Run benchmarks
go test ./test/performance/... -bench=. -benchmem
```

## Troubleshooting

### Common Performance Issues

#### Issue 1: Query Timeout

**Diagnosis:**
```bash
# Check slow query logs
kubectl logs -f deployment/edge-logs | grep "慢查询"
```

**Solutions:**
1. Optimize query with PREWHERE
2. Add database indexes
3. Reduce time range
4. Increase query timeout

#### Issue 2: High Memory Usage

**Diagnosis:**
```bash
# Check memory metrics
curl http://edge-logs:8080/metrics | grep memory
```

**Solutions:**
1. Reduce page size
2. Enable result streaming
3. Add memory limits
4. Optimize data structures

#### Issue 3: Connection Pool Exhaustion

**Diagnosis:**
```bash
# Check connection pool stats
curl http://edge-logs:8080/metrics | grep connection_pool
```

**Solutions:**
1. Increase pool size
2. Reduce connection lifetime
3. Fix connection leaks
4. Enable health checks

## Best Practices

### 1. Query Design

- **Always use time range filtering** to leverage partitioning
- **Avoid SELECT *** - specify only required columns
- **Use pagination** for large result sets
- **Enable caching** for repeated queries

### 2. Database Configuration

- **Monitor connection pool** utilization
- **Use appropriate indexes** for query patterns
- **Partition tables by time** for efficient queries
- **Compress data** to reduce I/O

### 3. Application Architecture

- **Implement request validation** to prevent expensive queries
- **Use connection pooling** for database connections
- **Enable performance monitoring** for all queries
- **Implement graceful degradation** for enrichment failures

### 4. Operational Excellence

- **Set up performance alerts** for proactive monitoring
- **Run regular performance tests** to detect regressions
- **Monitor resource usage** to prevent bottlenecks
- **Document performance baselines** for comparison

## Conclusion

Following this performance tuning guide will help ensure that edge-logs meets the NFR1 requirement of sub-2 second query response times while maintaining optimal performance as data volumes and query complexity grow. Regular performance monitoring and testing are essential for maintaining service quality.

For specific performance issues or additional optimization guidance, refer to the performance monitoring dashboard and query logs.