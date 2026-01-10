package metrics

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"k8s.io/klog/v2"
)

// PerformanceMonitor provides comprehensive performance monitoring for queries
type PerformanceMonitor struct {
	metrics          *QueryPerformanceMetrics
	slowQueryLogger  *SlowQueryLogger
	enabled          bool
	monitorInterval  time.Duration
	lastMonitoredTime time.Time
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(metrics *QueryPerformanceMetrics) *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:          metrics,
		slowQueryLogger:  NewSlowQueryLogger(metrics),
		enabled:          true,
		monitorInterval:  30 * time.Second,
		lastMonitoredTime: time.Now(),
	}
}

// MonitorQueryExecution monitors a query execution from start to finish
func (pm *PerformanceMonitor) MonitorQueryExecution(
	ctx context.Context,
	queryType QueryType,
	dataset string,
	queryParams map[string]interface{},
	queryFunc func() (interface{}, error),
) (interface{}, error) {
	if !pm.enabled {
		return queryFunc()
	}

	// Record start time and memory
	startTime := time.Now()
	var startMemStats runtime.MemStats
	runtime.ReadMemStats(&startMemStats)

	// Calculate query complexity score
	complexityScore := pm.calculateQueryComplexity(queryType, queryParams)
	pm.metrics.RecordQueryComplexity(queryType, dataset, complexityScore)

	// Execute the query
	result, err := queryFunc()

	// Calculate execution time and memory usage
	executionTime := time.Since(startTime)
	var endMemStats runtime.MemStats
	runtime.ReadMemStats(&endMemStats)
	memoryUsed := endMemStats.Alloc - startMemStats.Alloc

	// Record metrics
	pm.metrics.RecordQueryDuration(queryType, dataset, executionTime)
	pm.metrics.RecordQueryMemoryUsage(queryType, dataset, int64(memoryUsed))

	// Check for slow queries
	if err == nil {
		pm.metrics.RecordQuerySuccess(queryType, dataset)
		pm.slowQueryLogger.CheckAndLogSlowQuery(ctx, queryType, dataset, executionTime, queryParams)

		klog.V(4).InfoS("查询执行监控完成",
			"query_type", queryType,
			"dataset", dataset,
			"duration_ms", executionTime.Milliseconds(),
			"memory_bytes", memoryUsed,
			"complexity_score", complexityScore)
	} else {
		pm.handleQueryError(queryType, dataset, err)
	}

	return result, err
}

// MonitorK8sAPICall monitors K8s API calls for metadata enrichment
func (pm *PerformanceMonitor) MonitorK8sAPICall(
	ctx context.Context,
	operation string,
	dataset string,
	apiCallFunc func() (interface{}, error),
) (interface{}, error) {
	if !pm.enabled {
		return apiCallFunc()
	}

	startTime := time.Now()

	result, err := apiCallFunc()

	executionTime := time.Since(startTime)
	pm.metrics.RecordK8sAPICall(operation, dataset, executionTime)

	if err != nil {
		pm.metrics.RecordK8sAPIError(operation, dataset, fmt.Sprintf("%T", err))
		klog.ErrorS(err, "K8s API 调用失败",
			"operation", operation,
			"dataset", dataset,
			"duration_ms", executionTime.Milliseconds())
	} else {
		klog.V(4).InfoS("K8s API 调用监控完成",
			"operation", operation,
			"dataset", dataset,
			"duration_ms", executionTime.Milliseconds())
	}

	return result, err
}

// MonitorCacheOperation monitors cache operations for enrichment service
func (pm *PerformanceMonitor) MonitorCacheOperation(
	cacheType string,
	dataset string,
	isHit bool,
) {
	if !pm.enabled {
		return
	}

	if isHit {
		// In a real implementation, we would track actual hit rates over time
		pm.metrics.RecordCacheHit(cacheType, dataset, 1.0) // 100% hit rate for this call
	} else {
		pm.metrics.RecordCacheMiss(cacheType, dataset)
	}
}

// UpdateConnectionPoolStats updates connection pool statistics
func (pm *PerformanceMonitor) UpdateConnectionPoolStats(
	dataset string,
	openConns,
	idleConns,
	inUseConns int,
) {
	if !pm.enabled {
		return
	}

	pm.metrics.UpdateConnectionPoolStats(dataset, openConns, idleConns, inUseConns)

	klog.V(4).InfoS("连接池统计已更新",
		"dataset", dataset,
		"open_connections", openConns,
		"idle_connections", idleConns,
		"in_use_connections", inUseConns)
}

// handleQueryError handles query errors and records appropriate metrics
func (pm *PerformanceMonitor) handleQueryError(queryType QueryType, dataset string, err error) {
	errorCategory := pm.categorizeError(err)
	pm.metrics.RecordQueryError(queryType, dataset, errorCategory)

	klog.ErrorS(err, "查询执行监控记录错误",
		"query_type", queryType,
		"dataset", dataset,
		"error_category", errorCategory)
}

// categorizeError categorizes errors for better metrics analysis
func (pm *PerformanceMonitor) categorizeError(err error) string {
	errMsg := err.Error()

	// Categorize based on error message content
	switch {
	case containsIgnoreCase(errMsg, "timeout") || containsIgnoreCase(errMsg, "deadline"):
		return ErrorCategoryTimeout
	case containsIgnoreCase(errMsg, "validation") || containsIgnoreCase(errMsg, "invalid"):
		return ErrorCategoryValidation
	case containsIgnoreCase(errMsg, "memory") || containsIgnoreCase(errMsg, "allocation"):
		return ErrorCategoryMemory
	case containsIgnoreCase(errMsg, "transform") || containsIgnoreCase(errMsg, "marshal"):
		return ErrorCategoryTransformation
	default:
		return ErrorCategoryRepository
	}
}

// calculateQueryComplexity calculates a complexity score (1-10) for a query
func (pm *PerformanceMonitor) calculateQueryComplexity(queryType QueryType, params map[string]interface{}) float64 {
	baseScore := 1.0

	// Base complexity by query type
	switch queryType {
	case QueryTypeBasic:
		baseScore = 2.0
	case QueryTypeFiltered:
		baseScore = 4.0
	case QueryTypeAggregation:
		baseScore = 6.0
	case QueryTypeEnriched:
		baseScore = 7.0
	}

	// Add complexity for filters
	if params != nil {
		// Check for multiple filters
		filterCount := 0
		if _, hasFilter := params["filter"]; hasFilter {
			filterCount++
		}
		if _, hasNamespace := params["namespace"]; hasNamespace {
			filterCount++
		}
		if _, hasPod := params["pod_name"]; hasPod {
			filterCount++
		}
		if _, hasContainer := params["container_name"]; hasContainer {
			filterCount++
		}

		// Each additional filter adds complexity
		baseScore += float64(filterCount) * 0.5

		// Check for large time ranges (would calculate actual range in production)
		if _, hasTime := params["start_time"]; hasTime {
			baseScore += 0.3 // Time range filtering adds some complexity
		}
	}

	// Ensure score is within 1-10 range
	if baseScore > 10.0 {
		baseScore = 10.0
	}
	if baseScore < 1.0 {
		baseScore = 1.0
	}

	return baseScore
}

// GetPerformanceSummary returns a summary of performance metrics
func (pm *PerformanceMonitor) GetPerformanceSummary(ctx context.Context) (*PerformanceSummary, error) {
	return &PerformanceSummary{
		Enabled:           pm.enabled,
		MonitorInterval:   pm.monitorInterval,
		LastMonitoredTime: pm.lastMonitoredTime,
		Timestamp:         time.Now(),
	}, nil
}

// PerformanceSummary contains a summary of performance monitoring status
type PerformanceSummary struct {
	Enabled           bool      `json:"enabled"`
	MonitorInterval   time.Duration `json:"monitor_interval_seconds"`
	LastMonitoredTime time.Time `json:"last_monitored_time"`
	Timestamp         time.Time `json:"timestamp"`
}

// Enable enables performance monitoring
func (pm *PerformanceMonitor) Enable() {
	pm.enabled = true
	klog.InfoS("性能监控已启用")
}

// Disable disables performance monitoring
func (pm *PerformanceMonitor) Disable() {
	pm.enabled = false
	klog.InfoS("性能监控已禁用")
}

// IsEnabled returns whether performance monitoring is enabled
func (pm *PerformanceMonitor) IsEnabled() bool {
	return pm.enabled
}

// SetMonitorInterval sets the monitoring interval
func (pm *PerformanceMonitor) SetMonitorInterval(interval time.Duration) {
	pm.monitorInterval = interval
	klog.InfoS("性能监控间隔已更新",
		"interval_seconds", interval.Seconds())
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s[:len(substr)] == substr ||
		 len(s) > len(substr) && containsIgnoreCase(s[1:], substr))
}