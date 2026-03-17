package clickhouse

import (
	"fmt"
	"strings"
	"time"

	"github.com/outpostos/edge-logs/pkg/model/request"
	"k8s.io/klog/v2"
)

// QueryBuilder provides type-safe ClickHouse query construction
type QueryBuilder struct {
	baseQuery  strings.Builder
	conditions []string
	args       []interface{}
	orderBy    string
	limit      *int
	offset     *int
	dataset    string // For logging context
}

// NewQueryBuilder creates a new query builder instance
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		conditions: make([]string, 0),
		args:       make([]interface{}, 0),
	}
}

// BuildLogQuery constructs a log query for OTEL standard table (ADR-001)
func (qb *QueryBuilder) BuildLogQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	klog.V(4).InfoS("构建日志查询",
		"dataset", req.Dataset,
		"start_time", req.StartTime,
		"end_time", req.EndTime,
		"filter", req.Filter,
		"namespace", req.Namespace)

	qb.dataset = req.Dataset

	// 1. Base query for OTEL logs table
	qb.baseQuery.WriteString(`
		SELECT
			Timestamp, TraceId, SpanId, TraceFlags,
			SeverityText, SeverityNumber, ServiceName, Body,
			ResourceSchemaUrl, ResourceAttributes,
			ScopeSchemaUrl, ScopeName, ScopeVersion, ScopeAttributes,
			LogAttributes
		FROM otel_logs
	`)

	// 2. Dataset filtering (multi-strategy for robustness)
	if req.Dataset != "" {
		// Strategy 1: Try ServiceName first (most reliable if set)
		// Strategy 2: Try LogAttributes['k8s.namespace.name'] (K8s metadata)
		// Strategy 3: Try __path__ extraction (fallback)
		//
		// Priority: ServiceName > LogAttributes > ResourceAttributes['__path__']
		//
		// Using OR condition with short-circuit evaluation for optimal performance
		qb.AddCondition("(ServiceName = ? OR LogAttributes['k8s.namespace.name'] = ? OR splitByString('_', ResourceAttributes['__path__'])[2] = ?)", req.Dataset, req.Dataset, req.Dataset)

		klog.V(3).InfoS("Dataset filtering applied",
			"dataset", req.Dataset,
			"strategy", "ServiceName OR LogAttributes OR __path__")
	}

	// 3. Time range filtering (leveraging ORDER BY optimization)
	if req.StartTime != nil {
		qb.AddCondition("Timestamp >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		qb.AddCondition("Timestamp <= ?", *req.EndTime)
	}

	// 4. K8s metadata filtering (from LogAttributes map)
	if req.Namespace != "" {
		qb.AddCondition("LogAttributes['k8s.namespace.name'] = ?", req.Namespace)
	}
	if req.PodName != "" {
		qb.AddCondition("LogAttributes['k8s.pod.name'] = ?", req.PodName)
	}
	if req.NodeName != "" {
		qb.AddCondition("LogAttributes['k8s.node.name'] = ?", req.NodeName)
	}

	// 5. Host filtering (from ResourceAttributes map)
	if req.HostIP != "" {
		qb.AddCondition("ResourceAttributes['host.ip'] = ?", req.HostIP)
	}
	if req.HostName != "" {
		qb.AddCondition("ResourceAttributes['host.name'] = ?", req.HostName)
	}

	// 6. Container filtering (from LogAttributes map)
	if req.ContainerName != "" {
		qb.AddCondition("LogAttributes['k8s.container.name'] = ?", req.ContainerName)
	}

	// 7. Severity filtering
	if req.Severity != "" {
		qb.AddCondition("SeverityText = ?", req.Severity)
	}

	// 8. Full-text search (tokenbf_v1 index utilization)
	if req.Filter != "" {
		qb.AddCondition("hasToken(Body, ?)", req.Filter)
	}

	// 9. Tag filtering (from LogAttributes map with bloom filter index)
	for key, value := range req.Tags {
		qb.AddCondition("LogAttributes[?] = ?", key, value)
	}

	// 10. ORDER BY to match primary key for optimal performance
	// Note: TimestampTime field doesn't exist in otel_logs table, using Timestamp only
	qb.SetOrderBy(fmt.Sprintf("ServiceName, Timestamp %s", strings.ToUpper(req.Direction)))

	// 11. Pagination
	if req.PageSize > 0 {
		qb.SetLimit(req.PageSize)
		if req.Page > 0 {
			qb.SetOffset(req.Page * req.PageSize)
		}
	}

	query, args := qb.Build()

	klog.V(4).InfoS("日志查询已构建",
		"dataset", req.Dataset,
		"condition_count", len(qb.conditions),
		"arg_count", len(args),
		"has_pagination", req.PageSize > 0)

	return query, args, nil
}

// BuildCountQuery constructs a count query for pagination (OTEL table)
func (qb *QueryBuilder) BuildCountQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	klog.V(4).InfoS("构建计数查询", "dataset", req.Dataset)

	qb.dataset = req.Dataset

	// Base count query for OTEL logs table
	qb.baseQuery.WriteString("SELECT count(*) FROM otel_logs")

	// Apply same filtering conditions as main query (excluding ORDER BY and LIMIT)
	if req.Dataset != "" {
		// Multi-strategy dataset filtering (same as main query)
		qb.AddCondition("(ServiceName = ? OR LogAttributes['k8s.namespace.name'] = ? OR splitByString('_', ResourceAttributes['__path__'])[2] = ?)", req.Dataset, req.Dataset, req.Dataset)
	}

	if req.StartTime != nil {
		qb.AddCondition("Timestamp >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		qb.AddCondition("Timestamp <= ?", *req.EndTime)
	}

	if req.Namespace != "" {
		qb.AddCondition("LogAttributes['k8s.namespace.name'] = ?", req.Namespace)
	}
	if req.PodName != "" {
		qb.AddCondition("LogAttributes['k8s.pod.name'] = ?", req.PodName)
	}
	if req.NodeName != "" {
		qb.AddCondition("LogAttributes['k8s.node.name'] = ?", req.NodeName)
	}

	if req.HostIP != "" {
		qb.AddCondition("ResourceAttributes['host.ip'] = ?", req.HostIP)
	}
	if req.HostName != "" {
		qb.AddCondition("ResourceAttributes['host.name'] = ?", req.HostName)
	}

	if req.ContainerName != "" {
		qb.AddCondition("LogAttributes['k8s.container.name'] = ?", req.ContainerName)
	}

	if req.Severity != "" {
		qb.AddCondition("SeverityText = ?", req.Severity)
	}

	if req.Filter != "" {
		qb.AddCondition("hasToken(Body, ?)", req.Filter)
	}

	for key, value := range req.Tags {
		qb.AddCondition("LogAttributes[?] = ?", key, value)
	}

	query, args := qb.Build()

	klog.V(4).InfoS("计数查询已构建", "dataset", req.Dataset, "condition_count", len(qb.conditions))

	return query, args, nil
}

// BuildInsertQuery is deprecated - OTEL Collector handles log insertion
// This method is kept for backward compatibility but should not be used
func (qb *QueryBuilder) BuildInsertQuery() (string, error) {
	klog.InfoS("警告: BuildInsertQuery 已废弃，日志写入由 OTEL Collector 处理")
	return "", fmt.Errorf("BuildInsertQuery is deprecated, use OTEL Collector for log ingestion")
}

// AddCondition adds a WHERE condition with parameters
func (qb *QueryBuilder) AddCondition(condition string, args ...interface{}) {
	qb.conditions = append(qb.conditions, condition)
	qb.args = append(qb.args, args...)
}

// SetOrderBy sets the ORDER BY clause
func (qb *QueryBuilder) SetOrderBy(orderBy string) {
	qb.orderBy = orderBy
}

// SetLimit sets the LIMIT clause
func (qb *QueryBuilder) SetLimit(limit int) {
	qb.limit = &limit
}

// SetOffset sets the OFFSET clause
func (qb *QueryBuilder) SetOffset(offset int) {
	qb.offset = &offset
}

// Build constructs the final SQL query with parameters
func (qb *QueryBuilder) Build() (string, []interface{}) {
	var query strings.Builder

	// Add base query
	query.WriteString(qb.baseQuery.String())

	// Add WHERE conditions
	if len(qb.conditions) > 0 {
		query.WriteString("\nWHERE ")
		query.WriteString(strings.Join(qb.conditions, " AND "))
	}

	// Add ORDER BY
	if qb.orderBy != "" {
		query.WriteString("\nORDER BY ")
		query.WriteString(qb.orderBy)
	}

	// Add LIMIT
	if qb.limit != nil {
		query.WriteString(fmt.Sprintf("\nLIMIT %d", *qb.limit))
	}

	// Add OFFSET
	if qb.offset != nil {
		query.WriteString(fmt.Sprintf("\nOFFSET %d", *qb.offset))
	}

	finalQuery := query.String()

	klog.V(5).InfoS("最终查询已构建",
		"dataset", qb.dataset,
		"query_length", len(finalQuery),
		"param_count", len(qb.args))

	return finalQuery, qb.args
}

// Reset clears the query builder for reuse
func (qb *QueryBuilder) Reset() {
	qb.baseQuery.Reset()
	qb.conditions = qb.conditions[:0]
	qb.args = qb.args[:0]
	qb.orderBy = ""
	qb.limit = nil
	qb.offset = nil
	qb.dataset = ""
}

// ValidateQuery performs basic query validation
func (qb *QueryBuilder) ValidateQuery(req *request.LogQueryRequest) error {
	// Dataset (ServiceName) is optional in OTEL schema, but recommended for partition pruning
	if req.Dataset == "" {
		klog.InfoS("查询未指定 dataset/ServiceName，可能影响查询性能")
	}

	// Validate time range is reasonable (prevent excessive scans)
	if req.StartTime != nil && req.EndTime != nil {
		duration := req.EndTime.Sub(*req.StartTime)
		if duration > 7*24*time.Hour {
			klog.InfoS("查询时间范围较大，可能影响性能",
				"dataset", req.Dataset,
				"duration_hours", duration.Hours())
		}
	}

	// Validate pagination limits
	if req.PageSize > 10000 {
		return NewValidationError("query_validation", "page_size cannot exceed 10000 for performance reasons")
	}

	// Warn about potentially expensive queries
	if req.Filter != "" && len(req.Filter) < 3 {
		klog.InfoS("短文本搜索可能影响性能",
			"dataset", req.Dataset,
			"filter_length", len(req.Filter))
	}

	return nil
}
