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

// BuildLogQuery constructs a log query leveraging Story 1-2 optimizations
func (qb *QueryBuilder) BuildLogQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	klog.V(4).InfoS("构建日志查询",
		"dataset", req.Dataset,
		"start_time", req.StartTime,
		"end_time", req.EndTime,
		"filter", req.Filter,
		"namespace", req.Namespace)

	qb.dataset = req.Dataset

	// 1. Base query leveraging partition pruning
	qb.baseQuery.WriteString(`
		SELECT
			timestamp, dataset, content, severity,
			container_id, container_name, pid,
			host_ip, host_name,
			k8s_namespace_name, k8s_pod_name, k8s_pod_uid, k8s_node_name,
			tags
		FROM logs
	`)

	// 2. Dataset filtering (REQUIRED for partition pruning)
	qb.AddCondition("dataset = ?", req.Dataset)

	// 3. Time range filtering (leveraging ORDER BY optimization)
	if req.StartTime != nil {
		qb.AddCondition("timestamp >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		qb.AddCondition("timestamp <= ?", *req.EndTime)
	}

	// 4. K8s metadata filtering (LowCardinality optimization)
	if req.Namespace != "" {
		qb.AddCondition("k8s_namespace_name = ?", req.Namespace)
	}
	if req.PodName != "" {
		qb.AddCondition("k8s_pod_name = ?", req.PodName)
	}
	if req.NodeName != "" {
		qb.AddCondition("k8s_node_name = ?", req.NodeName)
	}

	// 5. Host filtering
	if req.HostIP != "" {
		qb.AddCondition("host_ip = ?", req.HostIP)
	}
	if req.HostName != "" {
		qb.AddCondition("host_name = ?", req.HostName)
	}

	// 6. Container filtering
	if req.ContainerName != "" {
		qb.AddCondition("container_name = ?", req.ContainerName)
	}

	// 7. Severity filtering
	if req.Severity != "" {
		qb.AddCondition("severity = ?", req.Severity)
	}

	// 8. Full-text search (tokenbf_v1 index utilization)
	if req.Filter != "" {
		qb.AddCondition("hasToken(content, ?)", req.Filter)
	}

	// 9. Tag filtering (bloom filter index)
	for key, value := range req.Tags {
		qb.AddCondition("tags[?] = ?", key, value)
	}

	// 10. ORDER BY to match primary key for optimal performance
	qb.SetOrderBy(fmt.Sprintf("dataset, host_ip, timestamp %s", strings.ToUpper(req.Direction)))

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

// BuildCountQuery constructs a count query for pagination
func (qb *QueryBuilder) BuildCountQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	klog.V(4).InfoS("构建计数查询", "dataset", req.Dataset)

	qb.dataset = req.Dataset

	// Base count query with same filters as main query
	qb.baseQuery.WriteString("SELECT count(*) FROM logs")

	// Apply same filtering conditions as main query (excluding ORDER BY and LIMIT)
	qb.AddCondition("dataset = ?", req.Dataset)

	if req.StartTime != nil {
		qb.AddCondition("timestamp >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		qb.AddCondition("timestamp <= ?", *req.EndTime)
	}

	if req.Namespace != "" {
		qb.AddCondition("k8s_namespace_name = ?", req.Namespace)
	}
	if req.PodName != "" {
		qb.AddCondition("k8s_pod_name = ?", req.PodName)
	}
	if req.NodeName != "" {
		qb.AddCondition("k8s_node_name = ?", req.NodeName)
	}

	if req.HostIP != "" {
		qb.AddCondition("host_ip = ?", req.HostIP)
	}
	if req.HostName != "" {
		qb.AddCondition("host_name = ?", req.HostName)
	}

	if req.ContainerName != "" {
		qb.AddCondition("container_name = ?", req.ContainerName)
	}

	if req.Severity != "" {
		qb.AddCondition("severity = ?", req.Severity)
	}

	if req.Filter != "" {
		qb.AddCondition("hasToken(content, ?)", req.Filter)
	}

	for key, value := range req.Tags {
		qb.AddCondition("tags[?] = ?", key, value)
	}

	query, args := qb.Build()

	klog.V(4).InfoS("计数查询已构建", "dataset", req.Dataset, "condition_count", len(qb.conditions))

	return query, args, nil
}

// BuildInsertQuery constructs an insert query with batch optimization
func (qb *QueryBuilder) BuildInsertQuery() (string, error) {
	query := `
		INSERT INTO logs (
			timestamp, dataset, content, severity,
			container_id, container_name, pid,
			host_ip, host_name,
			k8s_namespace_name, k8s_pod_name, k8s_pod_uid, k8s_node_name,
			tags
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	klog.V(4).InfoS("插入查询已构建")
	return strings.TrimSpace(query), nil
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
	// Validate dataset is not empty (required for partition pruning)
	if req.Dataset == "" {
		return NewValidationError("query_validation", "dataset is required for data isolation")
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
