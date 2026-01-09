package clickhouse

import (
	"context"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

// QueryStats represents query execution statistics
type QueryStats struct {
	QueryID         string    `ch:"query_id"`
	Dataset         string    `ch:"dataset"`
	UserID          string    `ch:"user_id"`
	QueryType       string    `ch:"query_type"`   // 'search', 'aggregation', 'export'
	QueryParams     string    `ch:"query_params"` // JSON
	ExecutionTimeMs uint32    `ch:"execution_time_ms"`
	RowsExamined    uint64    `ch:"rows_examined"`
	RowsReturned    uint64    `ch:"rows_returned"`
	CreatedAt       time.Time `ch:"created_at"`
}

// MetricsRecorder handles query metrics recording
type MetricsRecorder struct {
	cm *ConnectionManager
}

// NewMetricsRecorder creates a new metrics recorder
func NewMetricsRecorder(cm *ConnectionManager) *MetricsRecorder {
	return &MetricsRecorder{cm: cm}
}

// RecordQueryMetrics asynchronously records query execution metrics
func (mr *MetricsRecorder) RecordQueryMetrics(dataset, queryType, queryParams string, duration time.Duration, rowsReturned uint64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := mr.insertQueryStats(ctx, dataset, queryType, queryParams, duration, rowsReturned); err != nil {
			klog.ErrorS(err, "记录查询指标失败",
				"dataset", dataset,
				"query_type", queryType,
				"duration_ms", duration.Milliseconds())
		}
	}()
}

// insertQueryStats inserts query statistics into the database
func (mr *MetricsRecorder) insertQueryStats(ctx context.Context, dataset, queryType, queryParams string, duration time.Duration, rowsReturned uint64) error {
	stats := &QueryStats{
		QueryID:         generateQueryID(),
		Dataset:         dataset,
		UserID:          "", // Will be set by auth middleware in future
		QueryType:       queryType,
		QueryParams:     queryParams,
		ExecutionTimeMs: uint32(duration.Milliseconds()),
		RowsExamined:    0, // Will be populated from ClickHouse query stats in future
		RowsReturned:    rowsReturned,
		CreatedAt:       time.Now(),
	}

	query := `
		INSERT INTO query_stats (
			query_id, dataset, user_id, query_type, query_params,
			execution_time_ms, rows_examined, rows_returned, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	db := mr.cm.GetDB()
	_, err := db.ExecContext(ctx, query,
		stats.QueryID,
		stats.Dataset,
		stats.UserID,
		stats.QueryType,
		stats.QueryParams,
		stats.ExecutionTimeMs,
		stats.RowsExamined,
		stats.RowsReturned,
		stats.CreatedAt,
	)

	if err != nil {
		return MapClickHouseError(err, "insert_query_stats").Err
	}

	klog.V(4).InfoS("查询指标已记录",
		"query_id", stats.QueryID,
		"dataset", dataset,
		"query_type", queryType,
		"duration_ms", duration.Milliseconds(),
		"rows_returned", rowsReturned)

	return nil
}

// GetQueryStatsByDataset retrieves query statistics for a dataset
func (mr *MetricsRecorder) GetQueryStatsByDataset(ctx context.Context, dataset string, since time.Time) ([]QueryStats, error) {
	query := `
		SELECT
			query_id, dataset, user_id, query_type, query_params,
			execution_time_ms, rows_examined, rows_returned, created_at
		FROM query_stats
		WHERE dataset = ? AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 1000
	`

	db := mr.cm.GetDB()
	rows, err := db.QueryContext(ctx, query, dataset, since)
	if err != nil {
		return nil, MapClickHouseError(err, "get_query_stats").Err
	}
	defer rows.Close()

	var stats []QueryStats
	for rows.Next() {
		var stat QueryStats
		if err := rows.Scan(
			&stat.QueryID,
			&stat.Dataset,
			&stat.UserID,
			&stat.QueryType,
			&stat.QueryParams,
			&stat.ExecutionTimeMs,
			&stat.RowsExamined,
			&stat.RowsReturned,
			&stat.CreatedAt,
		); err != nil {
			return nil, MapClickHouseError(err, "scan_query_stats").Err
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, MapClickHouseError(err, "iterate_query_stats").Err
	}

	klog.V(4).InfoS("查询指标已检索",
		"dataset", dataset,
		"stats_count", len(stats),
		"since", since)

	return stats, nil
}

// GetPerformanceMetrics calculates performance metrics for a dataset
func (mr *MetricsRecorder) GetPerformanceMetrics(ctx context.Context, dataset string, since time.Time) (*PerformanceMetrics, error) {
	query := `
		SELECT
			count() as total_queries,
			avg(execution_time_ms) as avg_execution_time_ms,
			quantile(0.95)(execution_time_ms) as p95_execution_time_ms,
			quantile(0.99)(execution_time_ms) as p99_execution_time_ms,
			max(execution_time_ms) as max_execution_time_ms,
			sum(rows_returned) as total_rows_returned
		FROM query_stats
		WHERE dataset = ? AND created_at >= ?
	`

	db := mr.cm.GetDB()
	row := db.QueryRowContext(ctx, query, dataset, since)

	var metrics PerformanceMetrics
	if err := row.Scan(
		&metrics.TotalQueries,
		&metrics.AvgExecutionTimeMs,
		&metrics.P95ExecutionTimeMs,
		&metrics.P99ExecutionTimeMs,
		&metrics.MaxExecutionTimeMs,
		&metrics.TotalRowsReturned,
	); err != nil {
		return nil, MapClickHouseError(err, "get_performance_metrics").Err
	}

	metrics.Dataset = dataset
	metrics.Since = since

	klog.V(4).InfoS("性能指标已计算",
		"dataset", dataset,
		"total_queries", metrics.TotalQueries,
		"avg_execution_time_ms", metrics.AvgExecutionTimeMs,
		"p95_execution_time_ms", metrics.P95ExecutionTimeMs)

	return &metrics, nil
}

// PerformanceMetrics represents aggregated performance metrics
type PerformanceMetrics struct {
	Dataset            string    `json:"dataset"`
	TotalQueries       uint64    `json:"total_queries"`
	AvgExecutionTimeMs float64   `json:"avg_execution_time_ms"`
	P95ExecutionTimeMs float64   `json:"p95_execution_time_ms"`
	P99ExecutionTimeMs float64   `json:"p99_execution_time_ms"`
	MaxExecutionTimeMs uint32    `json:"max_execution_time_ms"`
	TotalRowsReturned  uint64    `json:"total_rows_returned"`
	Since              time.Time `json:"since"`
}

// generateQueryID generates a unique query ID
func generateQueryID() string {
	return uuid.New().String()
}

// QueryMetricsCollector collects runtime query metrics
type QueryMetricsCollector struct {
	startTime   time.Time
	dataset     string
	queryType   string
	queryParams string
}

// NewQueryMetricsCollector creates a new query metrics collector
func NewQueryMetricsCollector(dataset, queryType, queryParams string) *QueryMetricsCollector {
	return &QueryMetricsCollector{
		startTime:   time.Now(),
		dataset:     dataset,
		queryType:   queryType,
		queryParams: queryParams,
	}
}

// Finish completes metrics collection and records the results
func (qmc *QueryMetricsCollector) Finish(mr *MetricsRecorder, rowsReturned uint64) {
	duration := time.Since(qmc.startTime)

	// Log query execution summary
	klog.InfoS("查询执行完成",
		"dataset", qmc.dataset,
		"query_type", qmc.queryType,
		"duration_ms", duration.Milliseconds(),
		"rows_returned", rowsReturned)

	// Record metrics asynchronously
	if mr != nil {
		mr.RecordQueryMetrics(qmc.dataset, qmc.queryType, qmc.queryParams, duration, rowsReturned)
	}

	// Log performance warnings if needed
	if duration > 5*time.Second {
		klog.InfoS("查询执行时间较长",
			"dataset", qmc.dataset,
			"duration_ms", duration.Milliseconds(),
			"warning_threshold_ms", 5000)
	}
}
