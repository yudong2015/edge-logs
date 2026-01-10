package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/klog/v2"
)

// SlowQueryLogger detects and logs slow queries for performance analysis
type SlowQueryLogger struct {
	metrics               *QueryPerformanceMetrics
	warningThreshold      time.Duration
	criticalThreshold     time.Duration
	enableDetailedLogging bool
}

// SlowQueryInfo contains detailed information about a slow query
type SlowQueryInfo struct {
	QueryID           string                 `json:"query_id"`
	QueryType         QueryType              `json:"query_type"`
	Dataset           string                 `json:"dataset"`
	ExecutionTime     time.Duration          `json:"execution_time_ms"`
	ThresholdExceeded string                 `json:"threshold_exceeded"`
	Timestamp         time.Time              `json:"timestamp"`
	QueryParameters   map[string]interface{} `json:"query_parameters,omitempty"`
	PerformanceImpact string                 `json:"performance_impact"`
	OptimizationHints []string               `json:"optimization_hints,omitempty"`
}

// NewSlowQueryLogger creates a new slow query logger
func NewSlowQueryLogger(metrics *QueryPerformanceMetrics) *SlowQueryLogger {
	return &SlowQueryLogger{
		metrics:               metrics,
		warningThreshold:      1500 * time.Millisecond, // 1.5 seconds warning
		criticalThreshold:     3000 * time.Millisecond, // 3 seconds critical
		enableDetailedLogging: true,
	}
}

// CheckAndLogSlowQuery checks if a query is slow and logs detailed information
func (sql *SlowQueryLogger) CheckAndLogSlowQuery(
	ctx context.Context,
	queryType QueryType,
	dataset string,
	executionTime time.Duration,
	queryParams map[string]interface{},
) {
	if executionTime < sql.warningThreshold {
		return // Query is within acceptable limits
	}

	// Determine severity level
	severity := sql.getSeverityLevel(executionTime)
	thresholdLabel := formatThresholdLabel(executionTime)

	// Create slow query info
	slowQueryInfo := &SlowQueryInfo{
		QueryID:           generateQueryID(),
		QueryType:         queryType,
		Dataset:           dataset,
		ExecutionTime:     executionTime,
		ThresholdExceeded: thresholdLabel,
		Timestamp:         time.Now(),
		QueryParameters:   queryParams,
		PerformanceImpact: sql.assessPerformanceImpact(executionTime),
		OptimizationHints: sql.generateOptimizationHints(queryType, executionTime, queryParams),
	}

	// Log slow query
	sql.logSlowQuery(slowQueryInfo, severity)

	// Update metrics
	sql.metrics.SlowQueryCount.WithLabelValues(
		string(queryType),
		dataset,
		thresholdLabel,
	).Inc()

	// Log additional details if enabled
	if sql.enableDetailedLogging {
		sql.logDetailedSlowQueryInfo(slowQueryInfo)
	}
}

// getSeverityLevel determines the severity level based on execution time
func (sql *SlowQueryLogger) getSeverityLevel(executionTime time.Duration) string {
	if executionTime >= sql.criticalThreshold {
		return "CRITICAL"
	}
	if executionTime >= sql.warningThreshold {
		return "WARNING"
	}
	return "INFO"
}

// assessPerformanceImpact provides a human-readable impact assessment
func (sql *SlowQueryLogger) assessPerformanceImpact(executionTime time.Duration) string {
	switch {
	case executionTime > 10*time.Second:
		return "SEVERE - Major impact on user experience and system capacity"
	case executionTime > 5*time.Second:
		return "HIGH - Significant impact on user experience"
	case executionTime > 3*time.Second:
		return "MEDIUM - Noticeable impact on user experience"
	case executionTime > 2*time.Second:
		return "LOW - Approaching NFR1 threshold (2 seconds)"
	default:
		return "MINIMAL - Within acceptable range but needs monitoring"
	}
}

// generateOptimizationHints provides actionable optimization suggestions
func (sql *SlowQueryLogger) generateOptimizationHints(
	queryType QueryType,
	executionTime time.Duration,
	queryParams map[string]interface{},
) []string {
	hints := []string{}

	// Time range optimization hints
	if _, hasStartTime := queryParams["start_time"]; hasStartTime {
		if _, hasEndTime := queryParams["end_time"]; hasEndTime {
			// Check if time range is very large
			timeRange := "unknown" // Would calculate actual range in production
			hints = append(hints, fmt.Sprintf("Consider reducing time range (current: %s)", timeRange))
		}
	}

	// Query type specific hints
	switch queryType {
	case QueryTypeAggregation:
		hints = append(hints, "Aggregation query: Consider reducing number of GROUP BY dimensions")
		hints = append(hints, "Consider using pre-aggregated materialized views for complex aggregations")

	case QueryTypeEnriched:
		hints = append(hints, "Enriched query: Metadata enrichment adds latency")
		hints = append(hints, "Consider enabling caching for K8s metadata")
		hints = append(hints, "Evaluate if all enrichment fields are necessary")

	case QueryTypeFiltered:
		hints = append(hints, "Filtered query: Multiple filters increase complexity")
		hints = append(hints, "Consider adding dataset-specific indexes")
	}

	// General performance hints
	if executionTime > 5*time.Second {
		hints = append(hints, "Query execution time critical: Review database table partitions")
		hints = append(hints, "Check ClickHouse system.query_log for detailed query analysis")
	}

	if executionTime > 2*time.Second {
		hints = append(hints, "Query exceeds NFR1 requirement: Target under 2 seconds for typical queries")
		hints = append(hints, "Review query complexity and result set size")
	}

	return hints
}

// logSlowQuery logs the slow query with appropriate severity
func (sql *SlowQueryLogger) logSlowQuery(info *SlowQueryInfo, severity string) {
	switch severity {
	case "CRITICAL":
		klog.ErrorS(nil, "检测到严重慢查询",
			"query_id", info.QueryID,
			"query_type", info.QueryType,
			"dataset", info.Dataset,
			"execution_time_ms", info.ExecutionTime.Milliseconds(),
			"severity", severity,
			"performance_impact", info.PerformanceImpact)

	case "WARNING":
		klog.InfoS("检测到慢查询",
			"query_id", info.QueryID,
			"query_type", info.QueryType,
			"dataset", info.Dataset,
			"execution_time_ms", info.ExecutionTime.Milliseconds(),
			"severity", severity,
			"performance_impact", info.PerformanceImpact)
	}
}

// logDetailedSlowQueryInfo logs detailed slow query information for analysis
func (sql *SlowQueryLogger) logDetailedSlowQueryInfo(info *SlowQueryInfo) {
	// Convert to JSON for structured logging
	if jsonData, err := json.Marshal(info); err == nil {
		klog.V(2).InfoS("慢查询详细信息",
			"query_id", info.QueryID,
			"slow_query_details", string(jsonData))
	}

	// Log optimization hints
	if len(info.OptimizationHints) > 0 {
		klog.InfoS("慢查询优化建议",
			"query_id", info.QueryID,
			"query_type", info.QueryType,
			"optimization_hints", info.OptimizationHints)
	}
}

// generateQueryID generates a unique query identifier
func generateQueryID() string {
	return fmt.Sprintf("query-%d", time.Now().UnixNano())
}

// GetSlowQueryStats returns statistics about slow queries
func (sql *SlowQueryLogger) GetSlowQueryStats(ctx context.Context, dataset string) (*SlowQueryStats, error) {
	// This would query the query_stats table for actual statistics
	// For now, return a placeholder implementation
	return &SlowQueryStats{
		Dataset:                dataset,
		TotalSlowQueries:       0,
		CriticalSlowQueries:    0,
		WarningSlowQueries:     0,
		AverageExecutionTime:   0,
		MostCommonQueryType:    "",
		MostCommonDataset:      dataset,
		Timestamp:              time.Now(),
	}, nil
}

// SlowQueryStats contains aggregated slow query statistics
type SlowQueryStats struct {
	Dataset             string    `json:"dataset"`
	TotalSlowQueries    int64     `json:"total_slow_queries"`
	CriticalSlowQueries int64     `json:"critical_slow_queries"`
	WarningSlowQueries  int64     `json:"warning_slow_queries"`
	AverageExecutionTime float64  `json:"average_execution_time_ms"`
	MostCommonQueryType string    `json:"most_common_query_type"`
	MostCommonDataset   string    `json:"most_common_dataset"`
	Timestamp           time.Time `json:"timestamp"`
}

// SetDetailedLogging enables or disables detailed slow query logging
func (sql *SlowQueryLogger) SetDetailedLogging(enabled bool) {
	sql.enableDetailedLogging = enabled
	klog.InfoS("慢查询详细日志设置已更新",
		"enabled", enabled)
}