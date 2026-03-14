package clickhouse

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// TimeQueryBuilder provides specialized time-range query optimization for ClickHouse
type TimeQueryBuilder struct {
	*QueryBuilder
}

// NewTimeQueryBuilder creates a new time-optimized query builder
func NewTimeQueryBuilder() *TimeQueryBuilder {
	return &TimeQueryBuilder{
		QueryBuilder: NewQueryBuilder(),
	}
}

// BuildOptimizedTimeRangeQuery builds time-range queries optimized for ClickHouse DateTime64(9) (OTEL format)
func (tqb *TimeQueryBuilder) BuildOptimizedTimeRangeQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	klog.V(4).InfoS("构建时间范围优化查询",
		"dataset", req.Dataset,
		"start_time", formatTimeForLog(req.StartTime),
		"end_time", formatTimeForLog(req.EndTime),
		"estimated_time_span", estimateTimeSpan(req.StartTime, req.EndTime))

	tqb.dataset = req.Dataset

	// Build base query with OTEL table field selection
	tqb.baseQuery.WriteString(`
		SELECT
			Timestamp, TraceId, SpanId, TraceFlags,
			SeverityText, SeverityNumber, ServiceName, Body,
			ResourceSchemaUrl, ResourceAttributes,
			ScopeSchemaUrl, ScopeName, ScopeVersion, ScopeAttributes,
			LogAttributes
		FROM otel_logs
	`)

	// 1. Dataset filtering: extract namespace from __path__
	// Path format: /var/log/containers/<pod>_<namespace>_<container>-<id>.log
	if req.Dataset != "" {
		tqb.AddCondition("splitByString('_', ResourceAttributes['__path__'])[2] = ?", req.Dataset)
	}

	// 2. Time range conditions with millisecond precision using toDateTime64
	if req.StartTime != nil {
		// Use >= for inclusive start boundary with nanosecond precision
		tqb.AddTimeCondition(">=", *req.StartTime)
	}

	if req.EndTime != nil {
		// Use <= for inclusive end boundary with nanosecond precision
		tqb.AddTimeCondition("<=", *req.EndTime)
	}

	// 3. Apply remaining filters after time filtering for optimal performance
	tqb.applyAdditionalFilters(req)

	// 4. Set optimal ordering for time-range queries
	tqb.SetTimeOptimizedOrdering(req.Direction)

	// 5. Apply pagination
	if req.PageSize > 0 {
		tqb.SetLimit(req.PageSize)
		if req.Page > 0 {
			tqb.SetOffset(req.Page * req.PageSize)
		}
	}

	query, args := tqb.Build()

	// Log query analysis for performance monitoring
	tqb.logQueryAnalysis(req, query)

	return query, args, nil
}

// BuildTimeRangeCountQuery builds optimized count queries for time ranges (OTEL format)
func (tqb *TimeQueryBuilder) BuildTimeRangeCountQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	klog.V(4).InfoS("构建时间范围计数查询", "dataset", req.Dataset)

	// CRITICAL: Reset builder state before building count query
	// This prevents appending to the previous query's baseQuery
	tqb.Reset()

	tqb.dataset = req.Dataset
	tqb.baseQuery.WriteString("SELECT count(*) FROM otel_logs")

	// Same filtering logic as main query but without ordering/pagination
	// Extract namespace from __path__ instead of using empty ServiceName
	if req.Dataset != "" {
		tqb.AddCondition("splitByString('_', ResourceAttributes['__path__'])[2] = ?", req.Dataset)
	}

	if req.StartTime != nil {
		tqb.AddTimeCondition(">=", *req.StartTime)
	}

	if req.EndTime != nil {
		tqb.AddTimeCondition("<=", *req.EndTime)
	}

	tqb.applyAdditionalFilters(req)

	query, args := tqb.Build()

	klog.V(4).InfoS("时间范围计数查询已构建",
		"dataset", req.Dataset,
		"condition_count", len(tqb.conditions))

	return query, args, nil
}

// AddTimeCondition adds time-specific conditions with optimal ClickHouse DateTime64 handling (OTEL format)
func (tqb *TimeQueryBuilder) AddTimeCondition(operator string, t time.Time) {
	// Use toDateTime64 with nanosecond precision for optimal time comparisons
	// ClickHouse automatically optimizes DateTime64 operations
	condition := fmt.Sprintf("Timestamp %s toDateTime64(?, 9)", operator)

	// Convert to Unix timestamp with nanosecond precision
	unixNano := t.UnixNano()
	seconds := float64(unixNano) / 1e9

	tqb.AddCondition(condition, seconds)

	klog.V(5).InfoS("已添加时间条件",
		"operator", operator,
		"timestamp", t.Format(time.RFC3339Nano),
		"unix_nano", unixNano,
		"seconds", seconds)
}

// SetTimeOptimizedOrdering sets ordering optimized for time-range queries (OTEL format)
func (tqb *TimeQueryBuilder) SetTimeOptimizedOrdering(direction string) {
	// For time-range queries, Timestamp ordering is most important for performance
	// ORDER BY uses Timestamp (TimestampTime field doesn't exist in otel_logs table)
	if strings.ToLower(direction) == "asc" {
		tqb.SetOrderBy("ServiceName ASC, Timestamp ASC")
	} else {
		tqb.SetOrderBy("ServiceName ASC, Timestamp DESC")
	}
}

// applyAdditionalFilters applies non-time filters after time filtering (OTEL format)
func (tqb *TimeQueryBuilder) applyAdditionalFilters(req *request.LogQueryRequest) {
	// K8s metadata filtering (from LogAttributes map)
	if req.Namespace != "" {
		tqb.AddCondition("LogAttributes['k8s.namespace.name'] = ?", req.Namespace)
	}
	if req.PodName != "" {
		tqb.AddCondition("LogAttributes['k8s.pod.name'] LIKE ?", "%"+req.PodName+"%")
	}
	if req.NodeName != "" {
		tqb.AddCondition("LogAttributes['k8s.node.name'] = ?", req.NodeName)
	}

	// Host filtering (from ResourceAttributes map)
	if req.HostIP != "" {
		tqb.AddCondition("ResourceAttributes['host.ip'] = ?", req.HostIP)
	}
	if req.HostName != "" {
		tqb.AddCondition("ResourceAttributes['host.name'] = ?", req.HostName)
	}

	// Container filtering (from LogAttributes map)
	if req.ContainerName != "" {
		tqb.AddCondition("LogAttributes['k8s.container.name'] = ?", req.ContainerName)
	}

	// Severity filtering
	if req.Severity != "" {
		tqb.AddCondition("SeverityText = ?", req.Severity)
	}

	// Full-text search (applied after time filtering for performance)
	if req.Filter != "" {
		// Use positionCaseInsensitive for better performance with time-filtered datasets
		tqb.AddCondition("positionCaseInsensitive(Body, ?) > 0", req.Filter)
	}

	// Tag filtering (from LogAttributes map with bloom filter index)
	for key, value := range req.Tags {
		tqb.AddCondition("LogAttributes[?] = ?", key, value)
	}
}

// logQueryAnalysis logs query performance analysis information
func (tqb *TimeQueryBuilder) logQueryAnalysis(req *request.LogQueryRequest, query string) {
	timeSpan := estimateTimeSpan(req.StartTime, req.EndTime)
	partitionEstimate := estimatePartitionCount(req.StartTime, req.EndTime)

	klog.InfoS("时间范围查询分析",
		"dataset", req.Dataset,
		"time_span_hours", timeSpan,
		"estimated_partitions", partitionEstimate,
		"query_length", len(query),
		"condition_count", len(tqb.conditions))

	// Warn about potentially expensive queries
	if timeSpan > 24 {
		klog.InfoS("大时间范围查询检测",
			"dataset", req.Dataset,
			"time_span_hours", timeSpan,
			"performance_impact", "high")
	}

	if partitionEstimate > 10 {
		klog.InfoS("高分区扫描查询检测",
			"dataset", req.Dataset,
			"estimated_partitions", partitionEstimate,
			"performance_impact", "medium")
	}
}

// ValidateTimeQuery performs time-specific query validation
func (tqb *TimeQueryBuilder) ValidateTimeQuery(req *request.LogQueryRequest) error {
	// Validate time range is present for optimal performance
	if req.StartTime == nil && req.EndTime == nil {
		klog.InfoS("无时间范围查询检测",
			"dataset", req.Dataset,
			"performance_warning", "unbounded time scan")
	}

	// Validate time range is reasonable
	if req.StartTime != nil && req.EndTime != nil {
		timeSpan := req.EndTime.Sub(*req.StartTime)

		if timeSpan > 7*24*time.Hour {
			return fmt.Errorf("time range too large for optimal performance: %v (max: 168h)", timeSpan)
		}

		if timeSpan < 0 {
			return fmt.Errorf("invalid time range: start_time after end_time")
		}

		// Check for microsecond-precision queries that might need special handling
		if timeSpan < time.Millisecond {
			klog.V(2).InfoS("亚毫秒级时间查询",
				"dataset", req.Dataset,
				"time_span_ns", timeSpan.Nanoseconds())
		}
	}

	return nil
}

// Helper functions for time analysis

// formatTimeForLog formats time for logging (handles nil pointers)
func formatTimeForLog(t *time.Time) string {
	if t == nil {
		return "<nil>"
	}
	return t.Format(time.RFC3339Nano)
}

// estimateTimeSpan calculates time span in hours (handles nil pointers)
func estimateTimeSpan(start, end *time.Time) float64 {
	if start == nil || end == nil {
		return 0
	}
	return end.Sub(*start).Hours()
}

// estimatePartitionCount estimates the number of partitions that will be scanned
func estimatePartitionCount(start, end *time.Time) int {
	if start == nil || end == nil {
		return 0
	}

	// Assume daily partitions for estimation
	timeSpan := end.Sub(*start)
	days := int(timeSpan.Hours() / 24)

	// Add 1 for partial days and minimum of 1 partition
	if days == 0 {
		return 1
	}
	return days + 1
}

// GetTimeQueryMetrics returns metrics about time query performance characteristics
func (tqb *TimeQueryBuilder) GetTimeQueryMetrics(req *request.LogQueryRequest) map[string]interface{} {
	metrics := make(map[string]interface{})

	if req.StartTime != nil && req.EndTime != nil {
		timeSpan := req.EndTime.Sub(*req.StartTime)
		metrics["time_span_seconds"] = timeSpan.Seconds()
		metrics["time_span_hours"] = timeSpan.Hours()
		metrics["estimated_partitions"] = estimatePartitionCount(req.StartTime, req.EndTime)

		// Categorize query complexity
		if timeSpan.Hours() < 1 {
			metrics["complexity"] = "low"
		} else if timeSpan.Hours() < 24 {
			metrics["complexity"] = "medium"
		} else {
			metrics["complexity"] = "high"
		}
	} else {
		metrics["complexity"] = "unbounded"
	}

	metrics["has_content_filter"] = req.Filter != ""
	metrics["filter_count"] = len(tqb.conditions)

	return metrics
}