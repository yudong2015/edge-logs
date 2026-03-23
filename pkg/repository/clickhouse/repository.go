package clickhouse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/config"
	"github.com/outpostos/edge-logs/pkg/model/clickhouse"
	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/model/response"
	"github.com/outpostos/edge-logs/pkg/model/search"
)

// Type aliases for shared search types
type ContentSearchType = search.ContentSearchType
type ContentSearchFilter = search.ContentSearchFilter
type ContentSearchExpression = search.ContentSearchExpression

// Constants from search package
const (
	ContentSearchExact        = search.ContentSearchExact
	ContentSearchCaseInsensitive = search.ContentSearchCaseInsensitive
	ContentSearchRegex        = search.ContentSearchRegex
	ContentSearchWildcard     = search.ContentSearchWildcard
	ContentSearchPhrase       = search.ContentSearchPhrase
	ContentSearchProximity    = search.ContentSearchProximity
	ContentSearchBoolean      = search.ContentSearchBoolean
)

// Repository interface defines log data access methods with dataset isolation
type Repository interface {
	QueryLogs(ctx context.Context, req *request.LogQueryRequest) ([]clickhouse.LogEntry, int, error)
	QueryAggregation(ctx context.Context, req *request.AggregationRequest) (*response.AggregationResponse, error)
	InsertLog(ctx context.Context, log *clickhouse.LogEntry) error
	InsertLogsBatch(ctx context.Context, logs []clickhouse.LogEntry) error
	HealthCheck(ctx context.Context) error
	Close() error

	// Dataset-specific methods for enhanced data isolation
	DatasetExists(ctx context.Context, dataset string) (bool, error)
	GetDatasetStats(ctx context.Context, dataset string) (*DatasetMetadata, error)
	ListAvailableDatasets(ctx context.Context) ([]string, error)
	GetDatasetHealth(ctx context.Context, dataset string) (*DatasetHealth, error)
}

// ClickHouseRepository implements Repository interface for ClickHouse
type ClickHouseRepository struct {
	cm      *ConnectionManager
	metrics *MetricsRecorder
	config  *config.ClickHouseConfig
}

// NewClickHouseRepository creates a new ClickHouse repository with connection management
func NewClickHouseRepository(cfg *config.ClickHouseConfig) (*ClickHouseRepository, error) {
	klog.InfoS("初始化 ClickHouse 仓储层",
		"host", cfg.Host,
		"database", cfg.Database,
		"max_open_conns", cfg.MaxOpenConns)

	// Initialize connection manager
	cm, err := NewConnectionManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Initialize metrics recorder
	metrics := NewMetricsRecorder(cm)

	repository := &ClickHouseRepository{
		cm:      cm,
		metrics: metrics,
		config:  cfg,
	}

	klog.InfoS("ClickHouse 仓储层初始化完成")
	return repository, nil
}

// QueryLogs queries logs from ClickHouse with dataset isolation and performance optimization
func (r *ClickHouseRepository) QueryLogs(ctx context.Context, req *request.LogQueryRequest) ([]clickhouse.LogEntry, int, error) {
	startTime := time.Now()

	klog.InfoS("开始日志查询",
		"dataset", req.Dataset,
		"start_time", req.StartTime,
		"end_time", req.EndTime,
		"namespace", req.Namespace,
		"filter", req.Filter,
		"page", req.Page,
		"page_size", req.PageSize)

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, 0, NewValidationError("query_logs", err.Error()).Err
	}

	// Initialize query metrics collector
	queryParams, _ := json.Marshal(req)
	metricsCollector := NewQueryMetricsCollector(req.Dataset, "search", string(queryParams))
	defer func() {
		if r.metrics != nil {
			metricsCollector.Finish(r.metrics, uint64(len([]clickhouse.LogEntry{})))
		}
	}()

	// Use content search query builder if content search is present
	var query, countQuery string
	var args, countArgs []interface{}
	var err error

	if req.ParsedContentSearch != nil && len(req.ParsedContentSearch.Filters) > 0 {
		// Use content search query builder for advanced content search
		csqb := NewContentSearchQueryBuilder()

		// Convert parsed content search back to service format for query building
		contentSearch := &ContentSearchExpression{
			GlobalOperator:   req.ParsedContentSearch.GlobalOperator,
			HighlightEnabled: req.ParsedContentSearch.HighlightEnabled,
			MaxSnippetLength: req.ParsedContentSearch.MaxSnippetLength,
			RelevanceScoring: req.ParsedContentSearch.RelevanceScoring,
		}

		for _, filter := range req.ParsedContentSearch.Filters {
			contentSearch.Filters = append(contentSearch.Filters, ContentSearchFilter{
				Type:              ContentSearchType(filter.Type),
				Pattern:           filter.Pattern,
				CaseInsensitive:   filter.CaseInsensitive,
				BooleanOperator:   filter.BooleanOperator,
				ProximityDistance: filter.ProximityDistance,
				FieldTarget:       filter.FieldTarget,
				Weight:            filter.Weight,
			})
		}

		// Build content search optimized main query
		query, args, err = csqb.BuildContentSearchQuery(req, contentSearch)
		if err != nil {
			klog.ErrorS(err, "内容搜索查询构建失败", "dataset", req.Dataset)
			return nil, 0, NewQueryError("build_content_search_query", "", err).Err
		}

		// Build content search optimized count query for pagination
		countQuery, countArgs, err = csqb.BuildContentSearchCountQuery(req, contentSearch)
		if err != nil {
			klog.ErrorS(err, "内容搜索计数查询构建失败", "dataset", req.Dataset)
			return nil, 0, NewQueryError("build_content_search_count_query", "", err).Err
		}
	} else if len(req.K8sFilters) > 0 {
		// Use K8s-optimized query builder when K8s filters are present
		kqb := NewK8sQueryBuilder()

		// Validate K8s query for performance and complexity
		if err := kqb.ValidateK8sQuery(req); err != nil {
			klog.ErrorS(err, "K8s查询验证失败", "dataset", req.Dataset)
			return nil, 0, err
		}

		// Build K8s-optimized main query
		query, args, err = kqb.BuildK8sOptimizedQuery(req)
		if err != nil {
			klog.ErrorS(err, "K8s优化查询构建失败", "dataset", req.Dataset)
			return nil, 0, NewQueryError("build_k8s_query", "", err).Err
		}

		// Build K8s-optimized count query for pagination
		countQuery, countArgs, err = kqb.BuildK8sCountQuery(req)
		if err != nil {
			klog.ErrorS(err, "K8s计数查询构建失败", "dataset", req.Dataset)
			return nil, 0, NewQueryError("build_k8s_count_query", "", err).Err
		}
	} else {
		// Use time-optimized query builder for time-only queries
		tqb := NewTimeQueryBuilder()

		// Validate time query for performance and precision
		if err := tqb.ValidateTimeQuery(req); err != nil {
			klog.ErrorS(err, "时间查询验证失败", "dataset", req.Dataset)
			return nil, 0, err
		}

		// Build time-optimized main query
		query, args, err = tqb.BuildOptimizedTimeRangeQuery(req)
		if err != nil {
			klog.ErrorS(err, "时间优化查询构建失败", "dataset", req.Dataset)
			return nil, 0, NewQueryError("build_time_query", "", err).Err
		}

		// Build time-optimized count query for pagination
		countQuery, countArgs, err = tqb.BuildTimeRangeCountQuery(req)
		if err != nil {
			klog.ErrorS(err, "时间计数查询构建失败", "dataset", req.Dataset)
			return nil, 0, NewQueryError("build_time_count_query", "", err).Err
		}
	}

	// Execute count query first
	var total int
	db := r.cm.GetDB()
	countRow := db.QueryRowContext(ctx, countQuery, countArgs...)
	if err := countRow.Scan(&total); err != nil {
		klog.ErrorS(err, "计数查询执行失败",
			"dataset", req.Dataset,
			"query", countQuery)
		return nil, 0, MapClickHouseError(err, "execute_count_query").Err
	}

	klog.V(4).InfoS("计数查询完成",
		"dataset", req.Dataset,
		"total_count", total)

	// Execute main query
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		klog.ErrorS(err, "主查询执行失败",
			"dataset", req.Dataset,
			"query", query)
		return nil, 0, MapClickHouseError(err, "execute_main_query").Err
	}
	defer rows.Close()

	// Scan results (OTEL format)
	var results []clickhouse.LogEntry
	for rows.Next() {
		var entry clickhouse.LogEntry
		var k8sPodName, k8sNamespaceName, k8sContainerName, k8sContainerID string

		if err := rows.Scan(
			&entry.Timestamp,
			&entry.TraceID,
			&entry.SpanID,
			&entry.TraceFlags,
			&entry.SeverityText,
			&entry.SeverityNumber,
			&entry.ServiceName,
			&entry.Body,
			&entry.ResourceSchemaUrl,
			&entry.ResourceAttributes,
			&entry.ScopeSchemaUrl,
			&entry.ScopeName,
			&entry.ScopeVersion,
			&entry.ScopeAttributes,
			&entry.LogAttributes,
			&k8sPodName,
			&k8sNamespaceName,
			&k8sContainerName,
			&k8sContainerID,
		); err != nil {
			klog.ErrorS(err, "结果扫描失败", "dataset", req.Dataset)
			return nil, 0, MapClickHouseError(err, "scan_results").Err
		}

		// Set extracted K8s fields
		entry.K8sPodName = k8sPodName
		entry.K8sNamespaceName = k8sNamespaceName
		entry.K8sContainerName = k8sContainerName
		entry.K8sContainerID = k8sContainerID

		results = append(results, entry)
	}

	if err := rows.Err(); err != nil {
		klog.ErrorS(err, "结果迭代失败", "dataset", req.Dataset)
		return nil, 0, MapClickHouseError(err, "iterate_results").Err
	}

	duration := time.Since(startTime)

	klog.InfoS("日志查询完成",
		"dataset", req.Dataset,
		"returned_rows", len(results),
		"total_rows", total,
		"duration_ms", duration.Milliseconds())

	// Update metrics collector with actual row count
	if r.metrics != nil {
		metricsCollector.Finish(r.metrics, uint64(len(results)))
	}

	return results, total, nil
}

// InsertLog is deprecated - OTEL Collector handles log insertion
// This method is kept for backward compatibility but logs a warning
func (r *ClickHouseRepository) InsertLog(ctx context.Context, log *clickhouse.LogEntry) error {
	klog.InfoS("警告: InsertLog 已废弃，日志写入由 OTEL Collector 处理",
		"service_name", log.ServiceName,
		"timestamp", log.Timestamp)

	// Return error indicating deprecation
	return fmt.Errorf("InsertLog is deprecated, use OTEL Collector for log ingestion")
}

// InsertLogsBatch is deprecated - OTEL Collector handles log insertion
// This method is kept for backward compatibility but logs a warning
func (r *ClickHouseRepository) InsertLogsBatch(ctx context.Context, logs []clickhouse.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	klog.InfoS("警告: InsertLogsBatch 已废弃，日志写入由 OTEL Collector 处理",
		"batch_size", len(logs))

	// Return error indicating deprecation
	return fmt.Errorf("InsertLogsBatch is deprecated, use OTEL Collector for log ingestion")
}

// HealthCheck performs a comprehensive health check of the repository (OTEL format)
func (r *ClickHouseRepository) HealthCheck(ctx context.Context) error {
	klog.V(4).InfoS("执行仓储层健康检查")

	// Check connection manager
	if err := r.cm.HealthCheck(ctx); err != nil {
		return fmt.Errorf("connection manager health check failed: %w", err)
	}

	// Test a simple query to verify unified logs table access
	query := "SELECT count(*) FROM logs_k8s LIMIT 1"
	db := r.cm.GetDB()

	var count int
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		klog.ErrorS(err, "仓储层模式验证失败")
		return MapClickHouseError(err, "schema_validation").Err
	}

	klog.V(4).InfoS("仓储层健康检查成功", "logs_accessible", true)
	return nil
}

// Close gracefully closes all repository connections
func (r *ClickHouseRepository) Close() error {
	klog.InfoS("关闭仓储层连接")

	if r.cm != nil {
		return r.cm.Close()
	}

	return nil
}

// GetDB returns the underlying SQL database connection for direct queries
func (r *ClickHouseRepository) GetDB() *sql.DB {
	if r.cm != nil {
		return r.cm.GetDB()
	}
	return nil
}

// validateLogEntry validates a log entry before insertion (OTEL format)
// Note: This is kept for backward compatibility but insert methods are deprecated
func (r *ClickHouseRepository) validateLogEntry(log *clickhouse.LogEntry) error {
	// ServiceName is required for data isolation (replaces dataset)
	if log.ServiceName == "" {
		return fmt.Errorf("ServiceName is required for data isolation")
	}

	// Timestamp is required
	if log.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Body is required (replaces content)
	if log.Body == "" {
		return fmt.Errorf("Body is required")
	}

	// Validate timestamp is not too old (prevent partition issues)
	if time.Since(log.Timestamp) > 90*24*time.Hour {
		return fmt.Errorf("timestamp is too old (>90 days)")
	}

	// Validate timestamp is not too far in the future
	if log.Timestamp.After(time.Now().Add(1 * time.Hour)) {
		return fmt.Errorf("timestamp is too far in the future")
	}

	return nil
}

// QueryAggregation executes aggregation queries on ClickHouse
func (r *ClickHouseRepository) QueryAggregation(ctx context.Context, req *request.AggregationRequest) (*response.AggregationResponse, error) {
	startTime := time.Now()

	klog.InfoS("开始聚合查询",
		"dataset", req.Dataset,
		"dimensions", len(req.Dimensions),
		"functions", len(req.Functions),
		"start_time", req.StartTime,
		"end_time", req.EndTime)

	// Build aggregation query
	queryBuilder := NewAggregationQueryBuilder()
	query, args, err := queryBuilder.BuildAggregationQuery(req)
	if err != nil {
		klog.ErrorS(err, "聚合查询构建失败", "dataset", req.Dataset)
		return nil, fmt.Errorf("failed to build aggregation query: %w", err)
	}

	klog.V(4).InfoS("执行聚合查询",
		"dataset", req.Dataset,
		"query", query,
		"args_count", len(args))

	// Execute query
	db := r.cm.GetDB()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		klog.ErrorS(err, "聚合查询执行失败",
			"dataset", req.Dataset,
			"query", query)
		return nil, MapClickHouseError(err, "execute_aggregation_query").Err
	}
	defer rows.Close()

	// Parse results
	var results []response.AggregationResult

	// Get column names for proper mapping
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	dimensionCount := len(req.Dimensions)

	for rows.Next() {
		// Create slice to hold all column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			klog.ErrorS(err, "聚合结果扫描失败", "dataset", req.Dataset)
			return nil, MapClickHouseError(err, "scan_aggregation_results").Err
		}

		// Separate dimensions and metrics
		dimensions := make(map[string]interface{})
		metrics := make(map[string]interface{})

		for i, col := range columns {
			if i < dimensionCount {
				// This is a dimension
				dimAlias := req.Dimensions[i].Alias
				if dimAlias == "" {
					dimAlias = string(req.Dimensions[i].Type)
				}
				dimensions[dimAlias] = values[i]
			} else {
				// This is a metric (aggregation function result)
				metrics[col] = values[i]
			}
		}

		results = append(results, response.AggregationResult{
			Dimensions: dimensions,
			Metrics:    metrics,
		})
	}

	if err := rows.Err(); err != nil {
		klog.ErrorS(err, "聚合结果迭代失败", "dataset", req.Dataset)
		return nil, MapClickHouseError(err, "iterate_aggregation_results").Err
	}

	duration := time.Since(startTime)

	klog.InfoS("聚合查询完成",
		"dataset", req.Dataset,
		"result_rows", len(results),
		"duration_ms", duration.Milliseconds())

	// Build response
	aggResp := &response.AggregationResponse{
		Dataset: req.Dataset,
		Results: results,
		Metadata: &response.AggregationMetadata{
			QueryDurationMs: duration.Milliseconds(),
			DimensionCount:  len(req.Dimensions),
			FunctionCount:   len(req.Functions),
			ResultSetSize:   len(results),
			GeneratedAt:     time.Now(),
		},
		Query: &response.AggregationQueryInfo{
			Dimensions: buildDimensionNames(req.Dimensions),
			Functions:  buildFunctionNames(req.Functions),
			StartTime:  req.StartTime,
			EndTime:    req.EndTime,
		},
	}

	return aggResp, nil
}

// buildDimensionNames extracts dimension names from request
func buildDimensionNames(dimensions []request.AggregationDimension) []string {
	var names []string
	for _, dim := range dimensions {
		name := string(dim.Type)
		if dim.Alias != "" {
			name = dim.Alias
		}
		names = append(names, name)
	}
	return names
}

// buildFunctionNames extracts function names from request
func buildFunctionNames(functions []request.AggregationFunction) []string {
	var names []string
	for _, fn := range functions {
		name := string(fn.Type)
		if fn.Field != "" {
			name += "(" + fn.Field + ")"
		}
		if fn.Alias != "" {
			name = fn.Alias
		}
		names = append(names, name)
	}
	return names
}

