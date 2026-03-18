package clickhouse

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// K8sQueryBuilder provides specialized K8s metadata query optimization for ClickHouse
type K8sQueryBuilder struct {
	*TimeQueryBuilder
	k8sFilterBuilder *K8sFilterBuilder
}

// NewK8sQueryBuilder creates a new K8s-optimized query builder
func NewK8sQueryBuilder() *K8sQueryBuilder {
	return &K8sQueryBuilder{
		TimeQueryBuilder: NewTimeQueryBuilder(),
		k8sFilterBuilder: NewK8sFilterBuilder(),
	}
}

// BuildK8sOptimizedQuery builds queries optimized for K8s metadata filtering (OTEL format)
func (kqb *K8sQueryBuilder) BuildK8sOptimizedQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	klog.V(4).InfoS("构建K8s元数据优化查询",
		"dataset", req.Dataset,
		"k8s_filters", len(req.K8sFilters),
		"estimated_complexity", kqb.estimateK8sComplexity(req.K8sFilters))

	kqb.dataset = req.Dataset

	// Build base query with OTEL table field selection and K8s metadata extraction
	kqb.baseQuery.WriteString(`
		SELECT
			Timestamp, TraceId, SpanId, TraceFlags,
			SeverityText, SeverityNumber, ServiceName, Body,
			ResourceSchemaUrl, ResourceAttributes,
			ScopeSchemaUrl, ScopeName, ScopeVersion, ScopeAttributes,
			LogAttributes,
			-- Extract K8s metadata from __path__
			-- Path format: /var/log/containers/<pod>_<namespace>_<container>-<hash>.log
			-- Extract pod name (last element after splitting by '/')
			arrayElement(splitByString('/', splitByString('_', ResourceAttributes['__path__'])[1]), length(splitByString('/', splitByString('_', ResourceAttributes['__path__'])[1]))) as k8s_pod_name,
			splitByString('_', ResourceAttributes['__path__'])[2] as k8s_namespace_name,
			-- Extract container name (everything before the last '-' followed by 64-char hash and .log)
			substring(splitByString('_', ResourceAttributes['__path__'])[3], 1, length(splitByString('_', ResourceAttributes['__path__'])[3]) - 69) as k8s_container_name,
			-- Extract 64-char hash (before .log)
			substring(splitByString('_', ResourceAttributes['__path__'])[3], length(splitByString('_', ResourceAttributes['__path__'])[3]) - 68, 64) as k8s_container_id
		FROM otel_logs
	`)

	// Build comprehensive WHERE conditions with proper precedence
	var whereConditions []string
	var args []interface{}

	// 1. Dataset filter: extract namespace from __path__
	// Path format: /var/log/containers/<pod>_<namespace>_<container>-<id>.log
	if req.Dataset != "" {
		whereConditions = append(whereConditions, "splitByString('_', ResourceAttributes['__path__'])[2] = ?")
		args = append(args, req.Dataset)
	}

	// 2. Time range filters (optimized for DateTime64(9))
	if req.StartTime != nil {
		whereConditions = append(whereConditions, "Timestamp >= ?")
		args = append(args, *req.StartTime)
	}
	if req.EndTime != nil {
		whereConditions = append(whereConditions, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	// 3. K8s metadata filters (from LogAttributes map)
	if len(req.K8sFilters) > 0 {
		kqb.k8sFilterBuilder.SetFilters(req.K8sFilters)
		k8sConditions, k8sArgs, err := kqb.k8sFilterBuilder.BuildK8sFilterConditions()
		if err != nil {
			return "", nil, fmt.Errorf("failed to build K8s filter conditions: %w", err)
		}
		whereConditions = append(whereConditions, k8sConditions...)
		args = append(args, k8sArgs...)
	}

	// 4. Additional content and metadata filters
	if req.Filter != "" {
		whereConditions = append(whereConditions, "positionCaseInsensitive(Body, ?) > 0")
		args = append(args, req.Filter)
	}

	if req.Severity != "" {
		whereConditions = append(whereConditions, "SeverityText = ?")
		args = append(args, req.Severity)
	}

	if req.NodeName != "" {
		whereConditions = append(whereConditions, "LogAttributes['k8s.node.name'] = ?")
		args = append(args, req.NodeName)
	}

	if req.ContainerName != "" {
		whereConditions = append(whereConditions, "LogAttributes['k8s.container.name'] = ?")
		args = append(args, req.ContainerName)
	}

	// Build final query with optimized ordering for K8s metadata
	whereClause := strings.Join(whereConditions, " AND ")
	query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY Timestamp DESC, LogAttributes['k8s.namespace.name'] ASC, LogAttributes['k8s.pod.name'] ASC
		LIMIT %d OFFSET %d`,
		kqb.baseQuery.String(),
		whereClause,
		req.PageSize,
		req.Page*req.PageSize)

	klog.V(4).InfoS("K8s元数据查询构建完成",
		"dataset", req.Dataset,
		"where_conditions", len(whereConditions),
		"args_count", len(args),
		"estimated_selectivity", kqb.k8sFilterBuilder.EstimateFilterSelectivity())

	return query, args, nil
}

// BuildK8sCountQuery builds count queries optimized for K8s metadata filtering (OTEL format)
func (kqb *K8sQueryBuilder) BuildK8sCountQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	klog.V(4).InfoS("构建K8s元数据计数查询", "dataset", req.Dataset)

	var whereConditions []string
	var args []interface{}

	// Dataset filter: extract namespace from __path__
	// Path format: /var/log/containers/<pod>_<namespace>_<container>-<id>.log
	if req.Dataset != "" {
		whereConditions = append(whereConditions, "splitByString('_', ResourceAttributes['__path__'])[2] = ?")
		args = append(args, req.Dataset)
	}

	// Time range filters
	if req.StartTime != nil {
		whereConditions = append(whereConditions, "Timestamp >= ?")
		args = append(args, *req.StartTime)
	}
	if req.EndTime != nil {
		whereConditions = append(whereConditions, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	// K8s metadata filters
	if len(req.K8sFilters) > 0 {
		kqb.k8sFilterBuilder.SetFilters(req.K8sFilters)
		k8sConditions, k8sArgs, err := kqb.k8sFilterBuilder.BuildK8sFilterConditions()
		if err != nil {
			return "", nil, fmt.Errorf("failed to build K8s count conditions: %w", err)
		}
		whereConditions = append(whereConditions, k8sConditions...)
		args = append(args, k8sArgs...)
	}

	// Additional filters
	if req.Filter != "" {
		whereConditions = append(whereConditions, "positionCaseInsensitive(Body, ?) > 0")
		args = append(args, req.Filter)
	}

	if req.Severity != "" {
		whereConditions = append(whereConditions, "SeverityText = ?")
		args = append(args, req.Severity)
	}

	if req.NodeName != "" {
		whereConditions = append(whereConditions, "LogAttributes['k8s.node.name'] = ?")
		args = append(args, req.NodeName)
	}

	if req.ContainerName != "" {
		whereConditions = append(whereConditions, "LogAttributes['k8s.container.name'] = ?")
		args = append(args, req.ContainerName)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM otel_logs
		WHERE %s`,
		strings.Join(whereConditions, " AND "))

	return query, args, nil
}

// ValidateK8sQuery performs validation specific to K8s metadata queries
func (kqb *K8sQueryBuilder) ValidateK8sQuery(req *request.LogQueryRequest) error {
	// Validate K8s filter complexity
	if len(req.K8sFilters) > 0 {
		complexity := kqb.k8sFilterBuilder.EstimateFilterComplexity()
		if complexity > 100.0 {
			return fmt.Errorf("K8s filter complexity too high (%.1f), consider simplifying filters", complexity)
		}

		selectivity := kqb.k8sFilterBuilder.EstimateFilterSelectivity()
		if selectivity < 0.001 {
			klog.InfoS("K8s查询选择性很高，可能需要较长执行时间",
				"dataset", req.Dataset,
				"estimated_selectivity", selectivity,
				"filter_count", len(req.K8sFilters))
		}
	}

	// Validate time range for K8s queries (K8s metadata queries can be expensive over long periods)
	if req.StartTime != nil && req.EndTime != nil {
		timeSpan := req.EndTime.Sub(*req.StartTime)
		if timeSpan > 24*time.Hour && len(req.K8sFilters) > 5 {
			return fmt.Errorf("complex K8s queries over long time ranges (>24h) are not allowed for performance reasons")
		}
	}

	return nil
}

// estimateK8sComplexity provides complexity estimation for K8s metadata queries
func (kqb *K8sQueryBuilder) estimateK8sComplexity(filters []request.K8sFilter) float64 {
	kqb.k8sFilterBuilder.SetFilters(filters)
	return kqb.k8sFilterBuilder.EstimateFilterComplexity()
}

// GetK8sQueryOptimizationHints provides hints for query optimization
func (kqb *K8sQueryBuilder) GetK8sQueryOptimizationHints(req *request.LogQueryRequest) map[string]interface{} {
	hints := make(map[string]interface{})

	if len(req.K8sFilters) > 0 {
		kqb.k8sFilterBuilder.SetFilters(req.K8sFilters)

		hints["filter_complexity"] = kqb.k8sFilterBuilder.EstimateFilterComplexity()
		hints["estimated_selectivity"] = kqb.k8sFilterBuilder.EstimateFilterSelectivity()
		hints["filter_count"] = len(req.K8sFilters)

		// Analyze filter distribution
		namespaceCount := 0
		podCount := 0
		for _, filter := range req.K8sFilters {
			if filter.Field == "namespace" {
				namespaceCount++
			} else if filter.Field == "pod" {
				podCount++
			}
		}

		hints["namespace_filters"] = namespaceCount
		hints["pod_filters"] = podCount

		// Performance recommendations
		if namespaceCount > 10 {
			hints["recommendation"] = "Consider reducing namespace filter count for better performance"
		} else if podCount > 20 {
			hints["recommendation"] = "Consider using prefix patterns instead of exact matches for pod filters"
		} else {
			hints["recommendation"] = "Filter configuration looks optimal"
		}
	}

	return hints
}

