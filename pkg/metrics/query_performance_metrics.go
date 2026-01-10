package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/klog/v2"
)

// QueryPerformanceMetrics holds all Prometheus metrics for query performance monitoring
type QueryPerformanceMetrics struct {
	// Query execution time histograms by query type
	QueryDuration *prometheus.HistogramVec

	// Query success and error rates
	QuerySuccessRate *prometheus.GaugeVec
	QueryErrorRate    *prometheus.CounterVec

	// Slow query tracking
	SlowQueryCount *prometheus.CounterVec

	// Memory and resource usage
	QueryMemoryUsage *prometheus.GaugeVec

	// Connection pool statistics
	ConnectionPoolStats *prometheus.GaugeVec

	// Cache performance for enrichment service
	CacheHitRate  *prometheus.GaugeVec
	CacheMissRate *prometheus.CounterVec

	// K8s API call metrics for metadata enrichment
	K8sAPICallDuration *prometheus.HistogramVec
	K8sAPICallErrors    *prometheus.CounterVec

	// Query complexity metrics
	QueryComplexityScore *prometheus.GaugeVec
}

// QueryType represents the type of query being executed
type QueryType string

const (
	QueryTypeBasic       QueryType = "basic"
	QueryTypeFiltered    QueryType = "filtered"
	QueryTypeAggregation QueryType = "aggregation"
	QueryTypeEnriched    QueryType = "enriched"
)

// NewQueryPerformanceMetrics creates and registers all query performance metrics
func NewQueryPerformanceMetrics() *QueryPerformanceMetrics {
	metrics := &QueryPerformanceMetrics{
		// Query execution time histograms with meaningful buckets for sub-2 second requirement
		QueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "edge_logs_query_duration_seconds",
				Help: "Query execution time in seconds by query type",
				// Buckets optimized for sub-2 second requirement
				Buckets: []float64{0.1, 0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 1.75, 2.0, 3.0, 5.0, 10.0},
			},
			[]string{"query_type", "dataset"}, // labels
		),

		// Query success rate (percentage)
		QuerySuccessRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "edge_logs_query_success_rate",
				Help: "Query success rate as percentage by query type",
			},
			[]string{"query_type", "dataset"},
		),

		// Query error count
		QueryErrorRate: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edge_logs_query_errors_total",
				Help: "Total number of query errors by query type and error category",
			},
			[]string{"query_type", "dataset", "error_category"},
		),

		// Slow query count (> 1.5 seconds warning threshold)
		SlowQueryCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edge_logs_slow_queries_total",
				Help: "Total number of slow queries (> 1.5 seconds) by query type",
			},
			[]string{"query_type", "dataset", "threshold_exceeded"},
		),

		// Memory usage during query execution
		QueryMemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "edge_logs_query_memory_bytes",
				Help: "Memory usage in bytes during query execution",
			},
			[]string{"query_type", "dataset"},
		),

		// Connection pool statistics
		ConnectionPoolStats: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "edge_logs_connection_pool_stats",
				Help: "Connection pool statistics (open, idle, in_use connections)",
			},
			[]string{"dataset", "stat_type"},
		),

		// Cache performance for enrichment service
		CacheHitRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "edge_logs_enrichment_cache_hit_rate",
				Help: "Enrichment service cache hit rate as percentage",
			},
			[]string{"cache_type", "dataset"},
		),

		CacheMissRate: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edge_logs_enrichment_cache_misses_total",
				Help: "Total number of enrichment cache misses",
			},
			[]string{"cache_type", "dataset"},
		),

		// K8s API call metrics
		K8sAPICallDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "edge_logs_k8s_api_call_duration_seconds",
				Help: "K8s API call duration in seconds for metadata enrichment",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
			},
			[]string{"operation", "dataset"},
		),

		K8sAPICallErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edge_logs_k8s_api_errors_total",
				Help: "Total number of K8s API call errors during metadata enrichment",
			},
			[]string{"operation", "dataset", "error_type"},
		),

		// Query complexity score (1-10 scale)
		QueryComplexityScore: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "edge_logs_query_complexity_score",
				Help: "Query complexity score (1-10) based on filters and operations",
			},
			[]string{"query_type", "dataset"},
		),
	}

	klog.InfoS("查询性能指标已初始化并注册到 Prometheus",
		"metrics_count", 11)

	return metrics
}

// RecordQueryDuration records the execution time of a query
func (m *QueryPerformanceMetrics) RecordQueryDuration(queryType QueryType, dataset string, duration time.Duration) {
	durationSeconds := duration.Seconds()
	m.QueryDuration.WithLabelValues(string(queryType), dataset).Observe(durationSeconds)

	// Log performance warnings based on query type
	warningThreshold := getWarningThreshold(queryType)
	if duration > warningThreshold {
		klog.InfoS("查询执行时间超过阈值",
			"query_type", queryType,
			"dataset", dataset,
			"duration_ms", duration.Milliseconds(),
			"warning_threshold_ms", warningThreshold.Milliseconds())

		// Increment slow query counter
		thresholdLabel := formatThresholdLabel(duration)
		m.SlowQueryCount.WithLabelValues(string(queryType), dataset, thresholdLabel).Inc()
	}
}

// RecordQuerySuccess records a successful query
func (m *QueryPerformanceMetrics) RecordQuerySuccess(queryType QueryType, dataset string) {
	m.QuerySuccessRate.WithLabelValues(string(queryType), dataset).Set(100)
}

// RecordQueryError records a failed query with error category
func (m *QueryPerformanceMetrics) RecordQueryError(queryType QueryType, dataset string, errorCategory string) {
	m.QueryErrorRate.WithLabelValues(string(queryType), dataset, errorCategory).Inc()

	// Update success rate (decrease to indicate failure)
	currentSuccess := m.QuerySuccessRate.WithLabelValues(string(queryType), dataset)
	// Simple decrement strategy - in production, use a more sophisticated calculation
	currentSuccess.Set(95) // Indicate degradation
}

// RecordQueryMemoryUsage records memory usage during query execution
func (m *QueryPerformanceMetrics) RecordQueryMemoryUsage(queryType QueryType, dataset string, memoryBytes int64) {
	m.QueryMemoryUsage.WithLabelValues(string(queryType), dataset).Set(float64(memoryBytes))
}

// UpdateConnectionPoolStats updates connection pool statistics
func (m *QueryPerformanceMetrics) UpdateConnectionPoolStats(dataset string, openConns, idleConns, inUseConns int) {
	m.ConnectionPoolStats.WithLabelValues(dataset, "open").Set(float64(openConns))
	m.ConnectionPoolStats.WithLabelValues(dataset, "idle").Set(float64(idleConns))
	m.ConnectionPoolStats.WithLabelValues(dataset, "in_use").Set(float64(inUseConns))
}

// RecordCacheHit records a cache hit for enrichment service
func (m *QueryPerformanceMetrics) RecordCacheHit(cacheType, dataset string, hitRate float64) {
	m.CacheHitRate.WithLabelValues(cacheType, dataset).Set(hitRate)
}

// RecordCacheMiss records a cache miss for enrichment service
func (m *QueryPerformanceMetrics) RecordCacheMiss(cacheType, dataset string) {
	m.CacheMissRate.WithLabelValues(cacheType, dataset).Inc()
}

// RecordK8sAPICall records K8s API call duration
func (m *QueryPerformanceMetrics) RecordK8sAPICall(operation, dataset string, duration time.Duration) {
	m.K8sAPICallDuration.WithLabelValues(operation, dataset).Observe(duration.Seconds())
}

// RecordK8sAPIError records K8s API call error
func (m *QueryPerformanceMetrics) RecordK8sAPIError(operation, dataset string, errorType string) {
	m.K8sAPICallErrors.WithLabelValues(operation, dataset, errorType).Inc()
}

// RecordQueryComplexity records the complexity score of a query
func (m *QueryPerformanceMetrics) RecordQueryComplexity(queryType QueryType, dataset string, complexityScore float64) {
	m.QueryComplexityScore.WithLabelValues(string(queryType), dataset).Set(complexityScore)
}

// getWarningThreshold returns the warning threshold for different query types
func getWarningThreshold(queryType QueryType) time.Duration {
	switch queryType {
	case QueryTypeBasic:
		return 500 * time.Millisecond  // Basic queries should be fast
	case QueryTypeFiltered:
		return 1000 * time.Millisecond // 1 second for filtered queries
	case QueryTypeAggregation:
		return 1500 * time.Millisecond // 1.5 seconds for aggregations
	case QueryTypeEnriched:
		return 2000 * time.Millisecond // 2 seconds for enriched queries (NFR1 requirement)
	default:
		return 1500 * time.Millisecond // Default threshold
	}
}

// formatThresholdLabel creates a label for slow query categorization
func formatThresholdLabel(duration time.Duration) string {
	switch {
	case duration > 5*time.Second:
		return "critical_5s_plus"
	case duration > 3*time.Second:
		return "high_3s_to_5s"
	case duration > 2*time.Second:
		return "medium_2s_to_3s"
	default:
		return "warning_1.5s_to_2s"
	}
}

// Error categories for query error tracking
const (
	ErrorCategoryValidation   = "validation"
	ErrorCategoryRepository   = "repository"
	ErrorCategoryTransformation = "transformation"
	ErrorCategoryTimeout      = "timeout"
	ErrorCategoryMemory       = "memory"
	ErrorCategoryBusinessLogic = "business_logic"
)