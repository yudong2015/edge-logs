# Performance Monitoring and Alerting Guide

## Overview

This guide provides comprehensive monitoring and alerting configuration for edge-logs to ensure optimal performance and early detection of performance issues.

## Prometheus Metrics Setup

### Metrics Endpoint Configuration

**Edge-Logs Metrics Endpoint:**
- **Endpoint**: `/metrics`
- **Port**: `8080` (default)
- **Format**: Prometheus text format

### Key Metrics

#### Query Performance Metrics

```promql
# Query duration by type (histogram)
edge_logs_query_duration_seconds_bucket{query_type="basic|filtered|aggregation|enriched",le="0.1|0.25|0.5|..."}
edge_logs_query_duration_seconds_sum{query_type="basic|filtered|aggregation|enriched"}
edge_logs_query_duration_seconds_count{query_type="basic|filtered|aggregation|enriched"}

# Query success rate (gauge)
edge_logs_query_success_rate{query_type="basic|filtered|aggregation|enriched",dataset="dataset-name"}

# Query error count (counter)
edge_logs_query_errors_total{query_type="basic|filtered|aggregation|enriched",dataset="dataset-name",error_category="validation|repository|transformation|timeout|memory"}

# Slow query count (counter)
edge_logs_slow_queries_total{query_type="basic|filtered|aggregation|enriched",dataset="dataset-name",threshold_exceeded="warning_1.5s_to_2s|medium_2s_to_3s|high_3s_to_5s|critical_5s_plus"}
```

#### Resource Usage Metrics

```promql
# Memory usage during queries (gauge)
edge_logs_query_memory_bytes{query_type="basic|filtered|aggregation|enriched",dataset="dataset-name"}

# Connection pool statistics (gauge)
edge_logs_connection_pool_stats{dataset="dataset-name",stat_type="open|idle|in_use"}
```

#### Enrichment Service Metrics

```promql
# Cache performance (gauge)
edge_logs_enrichment_cache_hit_rate{cache_type="memory|distributed",dataset="dataset-name"}

# Cache misses (counter)
edge_logs_enrichment_cache_misses_total{cache_type="memory|distributed",dataset="dataset-name"}

# K8s API call performance (histogram)
edge_logs_k8s_api_call_duration_seconds_bucket{operation="get_pod|list_pods|get_namespace",le="0.01|0.05|..."}
edge_logs_k8s_api_call_duration_seconds_sum{operation="get_pod|list_pods|get_namespace"}
edge_logs_k8s_api_call_duration_seconds_count{operation="get_pod|list_pods|get_namespace"}

# K8s API errors (counter)
edge_logs_k8s_api_errors_total{operation="get_pod|list_pods|get_namespace",dataset="dataset-name",error_type="timeout|not_found|forbidden"}
```

#### Query Complexity Metrics

```promql
# Query complexity score (gauge)
edge_logs_query_complexity_score{query_type="basic|filtered|aggregation|enriched",dataset="dataset-name"}
```

## Prometheus Configuration

### Scrape Configuration

**prometheus.yml:**
```yaml
scrape_configs:
  - job_name: 'edge-logs'
    static_configs:
      - targets: ['edge-logs:8080']
    scrape_interval: 15s
    scrape_timeout: 10s
    metrics_path: /metrics
    honor_labels: true
```

### Recording Rules

**recording_rules.yml:**
```yaml
groups:
  - name: edge_logs_performance
    interval: 30s
    rules:
      # Average query duration by type
      - record: job:edge_logs_query_duration_seconds:avg5m
        expr: avg(rate(edge_logs_query_duration_seconds_sum[5m]) / rate(edge_logs_query_duration_seconds_count[5m])) by (query_type)

      # P95 query duration by type
      - record: job:edge_logs_query_duration_seconds:p95_5m
        expr: histogram_quantile(0.95, sum(rate(edge_logs_query_duration_seconds_bucket[5m])) by (le, query_type))

      # P99 query duration by type
      - record: job:edge_logs_query_duration_seconds:p99_5m
        expr: histogram_quantile(0.99, sum(rate(edge_logs_query_duration_seconds_bucket[5m])) by (le, query_type))

      # Query success rate by type
      - record: job:edge_logs_query_success_rate:avg5m
        expr: avg(edge_logs_query_success_rate) by (query_type)

      # Slow query rate by type
      - record: job:edge_logs_slow_queries:rate5m
        expr: sum(rate(edge_logs_slow_queries_total[5m])) by (query_type, threshold_exceeded)

      # Connection pool utilization
      - record: job:edge_logs_connection_pool_utilization
        expr: edge_logs_connection_pool_stats{stat_type="in_use"} / edge_logs_connection_pool_stats{stat_type="open"}

      # Cache hit rate
      - record: job:edge_logs_cache_hit_rate:avg5m
        expr: avg(edge_logs_enrichment_cache_hit_rate) by (cache_type)

      # K8s API call duration
      - record: job:edge_logs_k8s_api_duration:avg5m
        expr: avg(rate(edge_logs_k8s_api_call_duration_seconds_sum[5m]) / rate(edge_logs_k8s_api_call_duration_seconds_count[5m])) by (operation)
```

## Alerting Rules

### Critical Alerts

**alerting_rules.yml:**
```yaml
groups:
  - name: edge_logs_critical
    interval: 30s
    rules:
      # NFR1 violation: Queries exceeding 2 seconds
      - alert: EdgeLogsNFR1Violation
        expr: job:edge_logs_query_duration_seconds:p95_5m{query_type="enriched"} > 2.0
        for: 5m
        labels:
          severity: critical
          component: query_performance
        annotations:
          summary: "NFR1 violation: Queries exceeding 2 seconds"
          description: "P95 enriched query duration is {{ $value }}s, exceeding NFR1 requirement of 2s"
          runbook_url: "https://docs.example.com/runbooks/nfr1-violation"

      # High error rate
      - alert: EdgeLogsHighErrorRate
        expr: sum(rate(edge_logs_query_errors_total[5m])) by (dataset) / sum(rate(edge_logs_query_duration_seconds_count[5m])) by (dataset) > 0.05
        for: 5m
        labels:
          severity: critical
          component: query_reliability
        annotations:
          summary: "High query error rate (>5%)"
          description: "Dataset {{ $labels.dataset }} has error rate of {{ $value | humanizePercentage }}"
```

### Warning Alerts

```yaml
  - name: edge_logs_warning
    interval: 30s
    rules:
      # Slow queries detected
      - alert: EdgeLogsSlowQueries
        expr: sum(rate(edge_logs_slow_queries_total[5m])) by (dataset, threshold_exceeded) > 1.0
        for: 5m
        labels:
          severity: warning
          component: query_performance
        annotations:
          summary: "Slow queries detected"
          description: "Dataset {{ $labels.dataset }} has {{ $value | humanize }} slow queries/min (threshold: {{ $labels.threshold_exceeded }})"

      # Connection pool exhaustion
      - alert: EdgeLogsConnectionPoolExhausted
        expr: job:edge_logs_connection_pool_utilization > 0.8
        for: 5m
        labels:
          severity: warning
          component: resource_management
        annotations:
          summary: "Connection pool utilization >80%"
          description: "Dataset {{ $labels.dataset }} connection pool is at {{ $value | humanizePercentage }} capacity"

      # High memory usage
      - alert: EdgeLogsHighMemoryUsage
        expr: avg(edge_logs_query_memory_bytes) by (dataset) > 100*1024*1024
        for: 5m
        labels:
          severity: warning
          component: resource_usage
        annotations:
          summary: "High memory usage during queries"
          description: "Dataset {{ $labels.dataset }} using {{ $value | humanize }} memory per query"

      # Low cache hit rate
      - alert: EdgeLogsLowCacheHitRate
        expr: job:edge_logs_cache_hit_rate:avg5m < 0.7
        for: 10m
        labels:
          severity: warning
          component: cache_performance
        annotations:
          summary: "Low enrichment cache hit rate (<70%)"
          description: "{{ $labels.cache_type }} cache hit rate is {{ $value | humanizePercentage }}"

      # High K8s API latency
      - alert: EdgeLogsHighK8sAPILatency
        expr: job:edge_logs_k8s_api_duration:p95_5m > 1.0
        for: 5m
        labels:
          severity: warning
          component: k8s_api_performance
        annotations:
          summary: "High K8s API call latency"
          description: "{{ $labels.operation }} P95 latency is {{ $value }}s"
```

### Info Alerts

```yaml
  - name: edge_logs_info
    interval: 1m
    rules:
      # Performance degradation
      - alert: EdgeLogsPerformanceDegradation
        expr: job:edge_logs_query_duration_seconds:p95_5m > (job:edge_logs_query_duration_seconds:p95_5m offset 1h) * 1.5
        for: 10m
        labels:
          severity: info
          component: performance_trend
        annotations:
          summary: "Query performance degraded by >50%"
          description: "{{ $labels.query_type }} P95 latency increased 50% in the last hour"

      # Query complexity increase
      - alert: EdgeLogsHighQueryComplexity
        expr: avg(edge_logs_query_complexity_score) by (dataset) > 7.0
        for: 5m
        labels:
          severity: info
          component: query_complexity
        annotations:
          summary: "High average query complexity detected"
          description: "Dataset {{ $labels.dataset }} average complexity score is {{ $value }}"
```

## Grafana Dashboard Configuration

### Dashboard JSON

**Recommended Dashboard Panels:**

1. **Query Performance Overview**
   - P50, P95, P99 query duration by type
   - Query success rate
   - Error rate breakdown

2. **Resource Utilization**
   - Connection pool statistics
   - Memory usage during queries
   - Query complexity trends

3. **Enrichment Service Performance**
   - Cache hit rates
   - K8s API call latency
   - API error rates

4. **NFR1 Compliance**
   - Real-time NFR1 compliance status
   - Slow query analysis
   - Performance trends

### Sample Dashboard Queries

**Query Duration Panel:**
```promql
# P95 Query Duration by Type
histogram_quantile(0.95,
  sum(rate(edge_logs_query_duration_seconds_bucket[5m])) by (le, query_type)
)
```

**Success Rate Panel:**
```promql
# Query Success Rate
sum(rate(edge_logs_query_duration_seconds_count[5m])) -
sum(rate(edge_logs_query_errors_total[5m])) /
sum(rate(edge_logs_query_duration_seconds_count[5m]))
```

**Connection Pool Panel:**
```promql
# Connection Pool Utilization
edge_logs_connection_pool_stats{stat_type="in_use"} /
edge_logs_connection_pool_stats{stat_type="open"}
```

## Monitoring Best Practices

### 1. Alert Threshold Tuning

- **Start with conservative thresholds** to reduce alert fatigue
- **Adjust based on baseline metrics** from normal operation
- **Use different thresholds** for different environments (dev, staging, prod)
- **Consider seasonal patterns** in usage

### 2. Dashboard Organization

- **Group related panels** for logical navigation
- **Use consistent color schemes** for status indicators
- **Include context** with annotations and descriptions
- **Set appropriate refresh intervals** (30s for critical, 1m for normal)

### 3. Performance Baseline

- **Establish baseline metrics** during normal operation
- **Document expected ranges** for each metric
- **Track trends over time** to detect gradual degradation
- **Compare across environments** for validation

### 4. Runbook Integration

- **Link alerts to runbooks** for automated responses
- **Include troubleshooting steps** in alert descriptions
- **Provide escalation paths** for different severity levels
- **Document recovery procedures** for common issues

## Troubleshooting Guide

### Performance Issues

**Investigate Query Performance:**
1. Check query duration metrics by type
2. Analyze slow query logs
3. Review connection pool utilization
4. Examine database performance

**Investigate Enrichment Issues:**
1. Check cache hit rates
2. Review K8s API call latency
3. Analyze API error rates
4. Validate K8s cluster health

### Resource Issues

**Investigate Memory Issues:**
1. Check query memory usage metrics
2. Analyze result set sizes
3. Review page size settings
4. Validate memory limits

**Investigate Connection Issues:**
1. Check connection pool statistics
2. Review connection health checks
3. Analyze connection lifecycle
4. Validate network connectivity

## Conclusion

Effective monitoring and alerting are essential for maintaining edge-logs performance and meeting NFR1 requirements. Regular review of metrics, alerts, and dashboards will help ensure optimal service quality and early detection of performance issues.

For specific metric interpretation or advanced monitoring scenarios, refer to the Prometheus documentation and internal runbooks.