package clickhouse

import (
	"fmt"
	"strings"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// AggregationQueryBuilder builds ClickHouse aggregation queries
type AggregationQueryBuilder struct {
	baseQuery string
}

// NewAggregationQueryBuilder creates a new aggregation query builder
func NewAggregationQueryBuilder() *AggregationQueryBuilder {
	return &AggregationQueryBuilder{
		baseQuery: "SELECT %s FROM otel_logs WHERE %s GROUP BY %s %s %s",
	}
}

// BuildAggregationQuery builds a complete aggregation query
func (b *AggregationQueryBuilder) BuildAggregationQuery(req *request.AggregationRequest) (string, []interface{}, error) {
	var whereConditions []string
	var args []interface{}

	// Build SELECT clause with dimensions and functions
	selectClause, err := b.buildSelectClause(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build SELECT clause: %w", err)
	}

	// Build WHERE clause with filters
	whereClause, whereArgs, err := b.buildWhereClause(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
	}
	whereConditions = append(whereConditions, whereClause)
	args = append(args, whereArgs...)

	// Build GROUP BY clause
	groupByClause := b.buildGroupByClause(req)

	// Build ORDER BY clause
	orderByClause := b.buildOrderByClause(req)

	// Build LIMIT clause
	limitClause := b.buildLimitClause(req)

	// Assemble final query
	query := fmt.Sprintf("SELECT %s FROM otel_logs WHERE %s",
		selectClause, strings.Join(whereConditions, " AND "))

	if groupByClause != "" {
		query += " " + groupByClause
	}
	if orderByClause != "" {
		query += " " + orderByClause
	}
	if limitClause != "" {
		query += " " + limitClause
	}

	return query, args, nil
}

// buildSelectClause builds the SELECT clause with dimensions and functions
func (b *AggregationQueryBuilder) buildSelectClause(req *request.AggregationRequest) (string, error) {
	var selectParts []string

	// Add dimension fields
	for _, dim := range req.Dimensions {
		dimField, err := b.buildDimensionField(dim)
		if err != nil {
			return "", fmt.Errorf("failed to build dimension field: %w", err)
		}
		if dim.Alias != "" {
			selectParts = append(selectParts, fmt.Sprintf("%s AS %s", dimField, dim.Alias))
		} else {
			selectParts = append(selectParts, dimField)
		}
	}

	// Add aggregation functions
	for _, fn := range req.Functions {
		fnField, err := b.buildAggregationFunction(fn)
		if err != nil {
			return "", fmt.Errorf("failed to build aggregation function: %w", err)
		}
		if fn.Alias != "" {
			selectParts = append(selectParts, fmt.Sprintf("%s AS %s", fnField, fn.Alias))
		} else {
			selectParts = append(selectParts, fnField)
		}
	}

	return strings.Join(selectParts, ", "), nil
}

// buildDimensionField creates dimension field expression (OTEL format)
func (b *AggregationQueryBuilder) buildDimensionField(dim request.AggregationDimension) (string, error) {
	switch dim.Type {
	case request.DimensionSeverity:
		return "SeverityText", nil
	case request.DimensionNamespace:
		return "ResourceAttributes['k8s.namespace.name']", nil
	case request.DimensionPodName:
		return "ResourceAttributes['k8s.pod.name']", nil
	case request.DimensionNodeName:
		return "LogAttributes['k8s.node.name']", nil
	case request.DimensionHostName:
		return "ResourceAttributes['host.name']", nil
	case request.DimensionContainerName:
		return "ResourceAttributes['k8s.container.name']", nil
	case request.DimensionDataset:
		return "ServiceName", nil
	case request.DimensionTimestamp:
		return b.buildTimeBucketExpression(dim.TimeBucket)
	default:
		return "", fmt.Errorf("unsupported dimension type: %s", dim.Type)
	}
}

// buildTimeBucketExpression creates time bucketing expression for ClickHouse (OTEL format)
func (b *AggregationQueryBuilder) buildTimeBucketExpression(bucket request.TimeBucketInterval) (string, error) {
	switch bucket {
	case request.IntervalMinute:
		return "toStartOfMinute(Timestamp)", nil
	case request.Interval5Minutes:
		return "toDateTime64(intDiv(toUInt64(toDateTime64(Timestamp, 3)), 300) * 300, 3) AS time_bucket", nil
	case request.Interval15Minutes:
		return "toDateTime64(intDiv(toUInt64(toDateTime64(Timestamp, 3)), 900) * 900, 3) AS time_bucket", nil
	case request.IntervalHour:
		return "toStartOfHour(Timestamp)", nil
	case request.Interval6Hours:
		return "toDateTime64(intDiv(toUInt64(toDateTime64(Timestamp, 3)), 21600) * 21600, 3) AS time_bucket", nil
	case request.Interval12Hours:
		return "toDateTime64(intDiv(toUInt64(toDateTime64(Timestamp, 3)), 43200) * 43200, 3) AS time_bucket", nil
	case request.IntervalDay:
		return "toStartOfDay(Timestamp)", nil
	case request.IntervalWeek:
		return "toStartOfWeek(Timestamp)", nil
	default:
		return "", fmt.Errorf("unsupported time bucket interval: %s", bucket)
	}
}

// buildAggregationFunction creates aggregation function expression
func (b *AggregationQueryBuilder) buildAggregationFunction(fn request.AggregationFunction) (string, error) {
	switch fn.Type {
	case request.FunctionCount:
		return "count(*)", nil
	case request.FunctionSum:
		if fn.Field == "" {
			return "", fmt.Errorf("sum function requires field specification")
		}
		return fmt.Sprintf("sum(%s)", fn.Field), nil
	case request.FunctionAvg:
		if fn.Field == "" {
			return "", fmt.Errorf("avg function requires field specification")
		}
		return fmt.Sprintf("avg(%s)", fn.Field), nil
	case request.FunctionMin:
		if fn.Field == "" {
			return "", fmt.Errorf("min function requires field specification")
		}
		return fmt.Sprintf("min(%s)", fn.Field), nil
	case request.FunctionMax:
		if fn.Field == "" {
			return "", fmt.Errorf("max function requires field specification")
		}
		return fmt.Sprintf("max(%s)", fn.Field), nil
	case request.FunctionDistinctCount:
		if fn.Field == "" {
			return "", fmt.Errorf("distinct_count function requires field specification")
		}
		return fmt.Sprintf("uniqExact(%s)", fn.Field), nil
	default:
		return "", fmt.Errorf("unsupported function type: %s", fn.Type)
	}
}

// buildWhereClause builds WHERE clause with filters (OTEL format)
func (b *AggregationQueryBuilder) buildWhereClause(req *request.AggregationRequest) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	// Dataset filter: extract namespace from __path__ (replaces ServiceName)
	// Path format: /var/log/containers/<pod>_<namespace>_<container>-<id>.log
	conditions = append(conditions, "splitByString('_', ResourceAttributes['__path__'])[2] = ?")
	args = append(args, req.Dataset)

	// Time range filters
	if req.StartTime != nil {
		conditions = append(conditions, "Timestamp >= ?")
		args = append(args, *req.StartTime)
	}
	if req.EndTime != nil {
		conditions = append(conditions, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	// Namespace filters (from ResourceAttributes)
	if len(req.Namespaces) > 0 {
		placeholders := make([]string, len(req.Namespaces))
		for i := range req.Namespaces {
			placeholders[i] = "?"
			args = append(args, req.Namespaces[i])
		}
		conditions = append(conditions, fmt.Sprintf("ResourceAttributes['k8s.namespace.name'] IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Pod name filters (from ResourceAttributes)
	if len(req.PodNames) > 0 {
		placeholders := make([]string, len(req.PodNames))
		for i := range req.PodNames {
			placeholders[i] = "?"
			args = append(args, req.PodNames[i])
		}
		conditions = append(conditions, fmt.Sprintf("ResourceAttributes['k8s.pod.name'] IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Severity filter
	if req.Severity != "" {
		conditions = append(conditions, "SeverityText = ?")
		args = append(args, req.Severity)
	}

	// Content search filter (Body field in OTEL format)
	if req.ContentSearch != "" {
		conditions = append(conditions, "positionCaseInsensitive(Body, ?) > 0")
		args = append(args, req.ContentSearch)
	}

	return strings.Join(conditions, " AND "), args, nil
}

// buildGroupByClause builds GROUP BY clause
func (b *AggregationQueryBuilder) buildGroupByClause(req *request.AggregationRequest) string {
	if len(req.Dimensions) == 0 {
		return ""
	}

	var groupByFields []string
	for i, dim := range req.Dimensions {
		if dim.Alias != "" {
			groupByFields = append(groupByFields, dim.Alias)
		} else {
			// Use position
			groupByFields = append(groupByFields, fmt.Sprintf("%d", i+1))
		}
	}

	return "GROUP BY " + strings.Join(groupByFields, ", ")
}

// buildOrderByClause builds ORDER BY clause
func (b *AggregationQueryBuilder) buildOrderByClause(req *request.AggregationRequest) string {
	var orderByParts []string

	// Add explicit order by fields
	for _, orderField := range req.OrderBy {
		orderByParts = append(orderByParts, orderField)
	}

	// Add dimension-specific ordering
	for _, dim := range req.Dimensions {
		if dim.SortOrder != "" {
			field := dim.Alias
			if field == "" {
				field = b.getDimensionDefaultAlias(dim.Type)
			}
			orderByParts = append(orderByParts, fmt.Sprintf("%s %s", field, dim.SortOrder))
		}
	}

	// Default ordering for time dimensions
	if b.hasTimeDimension(req.Dimensions) && len(orderByParts) == 0 {
		orderByParts = append(orderByParts, "time_bucket DESC")
	}

	if len(orderByParts) > 0 {
		return "ORDER BY " + strings.Join(orderByParts, ", ")
	}

	return ""
}

// buildLimitClause builds LIMIT clause
func (b *AggregationQueryBuilder) buildLimitClause(req *request.AggregationRequest) string {
	if req.Limit > 0 {
		limitClause := fmt.Sprintf("LIMIT %d", req.Limit)
		if req.Offset > 0 {
			limitClause += fmt.Sprintf(" OFFSET %d", req.Offset)
		}
		return limitClause
	}
	return ""
}

// getDimensionDefaultAlias gets default alias for dimension type (OTEL format)
func (b *AggregationQueryBuilder) getDimensionDefaultAlias(dimType request.AggregationDimensionType) string {
	switch dimType {
	case request.DimensionSeverity:
		return "SeverityText"
	case request.DimensionNamespace:
		return "namespace"
	case request.DimensionPodName:
		return "pod_name"
	case request.DimensionNodeName:
		return "node_name"
	case request.DimensionHostName:
		return "host_name"
	case request.DimensionContainerName:
		return "container_name"
	case request.DimensionDataset:
		return "ServiceName"
	case request.DimensionTimestamp:
		return "time_bucket"
	default:
		return "dimension"
	}
}

// hasTimeDimension checks if dimensions include timestamp
func (b *AggregationQueryBuilder) hasTimeDimension(dimensions []request.AggregationDimension) bool {
	for _, dim := range dimensions {
		if dim.Type == request.DimensionTimestamp {
			return true
		}
	}
	return false
}
