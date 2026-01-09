package v1alpha1

import (
	"time"

	"k8s.io/klog/v2"
)

// TimeMetrics tracks time-range query performance and patterns using structured logging
type TimeMetrics struct {
	enabled bool
}

// NewTimeMetrics creates a new time metrics collector
func NewTimeMetrics() *TimeMetrics {
	return &TimeMetrics{
		enabled: true,
	}
}

// RecordTimeQuery records metrics for a time-range query using structured logging
func (m *TimeMetrics) RecordTimeQuery(dataset string, duration time.Duration, timeSpan time.Duration, resultCount int) {
	if !m.enabled {
		return
	}

	// Categorize time span for analysis
	spanCategory := m.categorizeTimeSpan(timeSpan)
	complexity := m.categorizeComplexity(timeSpan, resultCount)

	klog.V(3).InfoS("时间范围查询指标",
		"dataset", dataset,
		"duration_ms", duration.Milliseconds(),
		"time_span_hours", timeSpan.Hours(),
		"time_span_category", spanCategory,
		"complexity", complexity,
		"result_count", resultCount)

	// Record slow queries for alerting
	if duration > 500*time.Millisecond {
		reason := m.determineSlownessReason(timeSpan, duration, resultCount)
		klog.InfoS("慢时间范围查询检测",
			"dataset", dataset,
			"duration_ms", duration.Milliseconds(),
			"reason", reason,
			"time_span_category", spanCategory,
			"time_span_hours", timeSpan.Hours(),
			"result_count", resultCount)
	}
}

// RecordTimeParsing records time parameter parsing metrics using structured logging
func (m *TimeMetrics) RecordTimeParsing(duration time.Duration, formatType, precision string) {
	if !m.enabled {
		return
	}

	klog.V(4).InfoS("时间参数解析指标",
		"duration_ms", duration.Milliseconds(),
		"format_type", formatType,
		"precision", precision)

	// Log slow parsing operations
	if duration > 10*time.Millisecond {
		klog.V(2).InfoS("慢时间解析检测",
			"duration_ms", duration.Milliseconds(),
			"format_type", formatType,
			"precision", precision)
	}
}

// RecordTimeError records time validation errors using structured logging
func (m *TimeMetrics) RecordTimeError(dataset, errorType, parameter string) {
	if !m.enabled {
		return
	}

	klog.InfoS("时间验证错误",
		"dataset", dataset,
		"error_type", errorType,
		"parameter", parameter)
}

// RecordPartitionScan records partition scanning metrics using structured logging
func (m *TimeMetrics) RecordPartitionScan(dataset string, estimatedPartitions int) {
	if !m.enabled {
		return
	}

	partitionCategory := m.categorizePartitionCount(estimatedPartitions)
	klog.V(3).InfoS("分区扫描指标",
		"dataset", dataset,
		"estimated_partitions", estimatedPartitions,
		"partition_category", partitionCategory)
}

// Helper methods for categorization

// categorizeTimeSpan categorizes time spans for metric labeling
func (m *TimeMetrics) categorizeTimeSpan(timeSpan time.Duration) string {
	hours := timeSpan.Hours()
	switch {
	case hours == 0:
		return "unknown"
	case hours <= 0.25:
		return "sub_hour"
	case hours <= 1:
		return "hourly"
	case hours <= 6:
		return "multi_hour"
	case hours <= 24:
		return "daily"
	default:
		return "extended"
	}
}

// categorizeComplexity determines query complexity based on time span and results
func (m *TimeMetrics) categorizeComplexity(timeSpan time.Duration, resultCount int) string {
	hours := timeSpan.Hours()

	switch {
	case hours <= 1 && resultCount <= 1000:
		return "low"
	case hours <= 6 && resultCount <= 5000:
		return "medium"
	case hours <= 24 && resultCount <= 10000:
		return "high"
	default:
		return "very_high"
	}
}

// categorizePartitionCount categorizes partition scan counts
func (m *TimeMetrics) categorizePartitionCount(partitions int) string {
	switch {
	case partitions <= 1:
		return "single"
	case partitions <= 5:
		return "few"
	case partitions <= 10:
		return "moderate"
	case partitions <= 25:
		return "many"
	default:
		return "excessive"
	}
}

// determineSlownessReason determines why a query was slow
func (m *TimeMetrics) determineSlownessReason(timeSpan, duration time.Duration, resultCount int) string {
	hours := timeSpan.Hours()

	switch {
	case hours > 12:
		return "large_time_span"
	case resultCount > 10000:
		return "high_result_count"
	case duration > 2*time.Second:
		return "timeout_risk"
	case hours < 1:
		return "processing_heavy"
	default:
		return "unknown"
	}
}

// GetTimeMetricsSummary returns current time metrics summary for monitoring
func (m *TimeMetrics) GetTimeMetricsSummary() map[string]interface{} {
	return map[string]interface{}{
		"metrics_available": []string{
			"time_query_duration",
			"time_range_span",
			"time_parsing_duration",
			"time_validation_errors",
			"partition_scans",
			"slow_time_queries",
		},
		"span_categories": []string{
			"sub_hour", "hourly", "multi_hour", "daily", "extended",
		},
		"complexity_levels": []string{
			"low", "medium", "high", "very_high",
		},
		"error_types": []string{
			"format_validation_failed",
			"range_validation_failed",
			"parameter_validation_failed",
		},
	}
}