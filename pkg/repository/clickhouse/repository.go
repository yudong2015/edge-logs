package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/config"
	"github.com/outpostos/edge-logs/pkg/model/clickhouse"
	"github.com/outpostos/edge-logs/pkg/model/request"
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

	// Scan results
	var results []clickhouse.LogEntry
	for rows.Next() {
		var entry clickhouse.LogEntry
		if err := rows.Scan(
			&entry.Timestamp,
			&entry.Dataset,
			&entry.Content,
			&entry.Severity,
			&entry.ContainerID,
			&entry.ContainerName,
			&entry.PID,
			&entry.HostIP,
			&entry.HostName,
			&entry.K8sNamespace,
			&entry.K8sPodName,
			&entry.K8sPodUID,
			&entry.K8sNodeName,
			&entry.Tags,
		); err != nil {
			klog.ErrorS(err, "结果扫描失败", "dataset", req.Dataset)
			return nil, 0, MapClickHouseError(err, "scan_results").Err
		}
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

// InsertLog inserts a single log entry into ClickHouse with dataset isolation
func (r *ClickHouseRepository) InsertLog(ctx context.Context, log *clickhouse.LogEntry) error {
	startTime := time.Now()

	klog.V(4).InfoS("开始单条日志插入",
		"dataset", log.Dataset,
		"timestamp", log.Timestamp,
		"severity", log.Severity)

	// Validate log entry
	if err := r.validateLogEntry(log); err != nil {
		return NewValidationError("insert_log", err.Error()).Err
	}

	// Build insert query
	qb := NewQueryBuilder()
	query, err := qb.BuildInsertQuery()
	if err != nil {
		return NewQueryError("build_insert_query", "", err).Err
	}

	// Execute insert
	db := r.cm.GetDB()
	_, err = db.ExecContext(ctx, query,
		log.Timestamp,
		log.Dataset,
		log.Content,
		log.Severity,
		log.ContainerID,
		log.ContainerName,
		log.PID,
		log.HostIP,
		log.HostName,
		log.K8sNamespace,
		log.K8sPodName,
		log.K8sPodUID,
		log.K8sNodeName,
		log.Tags,
	)

	if err != nil {
		klog.ErrorS(err, "日志插入失败",
			"dataset", log.Dataset,
			"timestamp", log.Timestamp)
		return MapClickHouseError(err, "execute_insert").Err
	}

	duration := time.Since(startTime)
	klog.V(4).InfoS("单条日志插入完成",
		"dataset", log.Dataset,
		"duration_ms", duration.Milliseconds())

	// Record metrics
	if r.metrics != nil {
		queryParams, _ := json.Marshal(map[string]interface{}{
			"dataset": log.Dataset,
			"single":  true,
		})
		metricsCollector := NewQueryMetricsCollector(log.Dataset, "insert", string(queryParams))
		metricsCollector.Finish(r.metrics, 1)
	}

	return nil
}

// InsertLogsBatch inserts multiple log entries in a batch for iLogtail optimization
func (r *ClickHouseRepository) InsertLogsBatch(ctx context.Context, logs []clickhouse.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	startTime := time.Now()
	dataset := logs[0].Dataset // Assume all logs belong to the same dataset

	klog.InfoS("开始批量日志插入",
		"dataset", dataset,
		"batch_size", len(logs))

	// Validate all log entries
	for i, log := range logs {
		if err := r.validateLogEntry(&log); err != nil {
			return NewValidationError("insert_logs_batch", fmt.Sprintf("log at index %d: %v", i, err)).Err
		}
	}

	// Use native connection for batch operations
	conn := r.cm.GetConn()

	// Prepare batch insert
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO logs (
			timestamp, dataset, content, severity,
			container_id, container_name, pid,
			host_ip, host_name,
			k8s_namespace_name, k8s_pod_name, k8s_pod_uid, k8s_node_name,
			tags
		)
	`)
	if err != nil {
		klog.ErrorS(err, "批量插入准备失败", "dataset", dataset)
		return MapClickHouseError(err, "prepare_batch").Err
	}

	// Add all logs to batch
	for _, log := range logs {
		if err := batch.Append(
			log.Timestamp,
			log.Dataset,
			log.Content,
			log.Severity,
			log.ContainerID,
			log.ContainerName,
			log.PID,
			log.HostIP,
			log.HostName,
			log.K8sNamespace,
			log.K8sPodName,
			log.K8sPodUID,
			log.K8sNodeName,
			log.Tags,
		); err != nil {
			klog.ErrorS(err, "批量追加失败", "dataset", dataset)
			return MapClickHouseError(err, "append_batch").Err
		}
	}

	// Execute batch
	if err := batch.Send(); err != nil {
		klog.ErrorS(err, "批量发送失败",
			"dataset", dataset,
			"batch_size", len(logs))
		return MapClickHouseError(err, "send_batch").Err
	}

	duration := time.Since(startTime)
	klog.InfoS("批量日志插入完成",
		"dataset", dataset,
		"batch_size", len(logs),
		"duration_ms", duration.Milliseconds(),
		"throughput_logs_per_sec", float64(len(logs))*1000/float64(duration.Milliseconds()))

	// Record metrics
	if r.metrics != nil {
		queryParams, _ := json.Marshal(map[string]interface{}{
			"dataset":    dataset,
			"batch_size": len(logs),
		})
		metricsCollector := NewQueryMetricsCollector(dataset, "batch_insert", string(queryParams))
		metricsCollector.Finish(r.metrics, uint64(len(logs)))
	}

	return nil
}

// HealthCheck performs a comprehensive health check of the repository
func (r *ClickHouseRepository) HealthCheck(ctx context.Context) error {
	klog.V(4).InfoS("执行仓储层健康检查")

	// Check connection manager
	if err := r.cm.HealthCheck(ctx); err != nil {
		return fmt.Errorf("connection manager health check failed: %w", err)
	}

	// Test a simple query to verify schema access
	query := "SELECT count(*) FROM datasets WHERE name = 'default'"
	db := r.cm.GetDB()

	var count int
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		klog.ErrorS(err, "仓储层模式验证失败")
		return MapClickHouseError(err, "schema_validation").Err
	}

	klog.V(4).InfoS("仓储层健康检查成功", "default_dataset_exists", count > 0)
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

// validateLogEntry validates a log entry before insertion
func (r *ClickHouseRepository) validateLogEntry(log *clickhouse.LogEntry) error {
	// Dataset is required for data isolation
	if log.Dataset == "" {
		return fmt.Errorf("dataset is required for data isolation")
	}

	// Timestamp is required
	if log.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Content is required
	if log.Content == "" {
		return fmt.Errorf("content is required")
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
