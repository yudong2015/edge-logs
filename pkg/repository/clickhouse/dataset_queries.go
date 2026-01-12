package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// DatasetMetadata represents dataset statistics
type DatasetMetadata struct {
	Name            string    `json:"name"`
	TotalLogs       int64     `json:"total_logs"`
	DateRange       DateRange `json:"date_range"`
	LastUpdated     time.Time `json:"last_updated"`
	PartitionCount  int       `json:"partition_count"`
	DataSizeBytes   int64     `json:"data_size_bytes"`
}

// DateRange represents the earliest and latest timestamps in a dataset
type DateRange struct {
	Earliest time.Time `json:"earliest"`
	Latest   time.Time `json:"latest"`
}

// DatasetExists checks if a dataset exists and contains data in ClickHouse (OTEL format)
func (r *ClickHouseRepository) DatasetExists(ctx context.Context, dataset string) (bool, error) {
	klog.V(4).InfoS("检查数据集是否存在", "dataset", dataset)

	// Validate dataset parameter
	if dataset == "" {
		return false, fmt.Errorf("dataset parameter cannot be empty")
	}

	// Check if dataset has any data in the otel_logs table (ServiceName replaces dataset)
	query := `
		SELECT COUNT(*)
		FROM otel_logs
		WHERE ServiceName = ?
		LIMIT 1
	`

	db := r.cm.GetDB()
	var count int64
	err := db.QueryRowContext(ctx, query, dataset).Scan(&count)
	if err != nil {
		klog.ErrorS(err, "数据集存在性检查失败", "dataset", dataset)
		return false, MapClickHouseError(err, "dataset_existence_check").Err
	}

	exists := count > 0
	klog.V(4).InfoS("数据集存在性检查完成",
		"dataset", dataset,
		"exists", exists,
		"log_count", count)

	return exists, nil
}

// GetDatasetStats retrieves comprehensive statistics for a dataset (OTEL format)
func (r *ClickHouseRepository) GetDatasetStats(ctx context.Context, dataset string) (*DatasetMetadata, error) {
	klog.V(4).InfoS("获取数据集统计信息", "dataset", dataset)

	// Validate dataset parameter
	if dataset == "" {
		return nil, fmt.Errorf("dataset parameter cannot be empty")
	}

	db := r.cm.GetDB()
	metadata := &DatasetMetadata{
		Name: dataset,
	}

	// Get basic statistics: total logs, date range (OTEL format: Timestamp replaces timestamp)
	statsQuery := `
		SELECT
			COUNT(*) as total_logs,
			MIN(Timestamp) as earliest_time,
			MAX(Timestamp) as latest_time
		FROM otel_logs
		WHERE ServiceName = ?
	`

	var earliest, latest time.Time
	err := db.QueryRowContext(ctx, statsQuery, dataset).Scan(
		&metadata.TotalLogs,
		&earliest,
		&latest,
	)
	if err != nil {
		klog.ErrorS(err, "数据集基础统计查询失败", "dataset", dataset)
		return nil, MapClickHouseError(err, "dataset_basic_stats").Err
	}

	metadata.DateRange = DateRange{
		Earliest: earliest,
		Latest:   latest,
	}
	metadata.LastUpdated = latest

	// Get partition count from system.parts (OTEL table: otel_logs)
	partitionQuery := `
		SELECT COUNT(DISTINCT partition) as partition_count
		FROM system.parts
		WHERE table = 'otel_logs'
		AND database = ?
		AND active = 1
	`

	err = db.QueryRowContext(ctx, partitionQuery, r.config.Database).Scan(&metadata.PartitionCount)
	if err != nil {
		klog.ErrorS(err, "数据集分区统计查询失败", "dataset", dataset)
		// Don't fail completely, just set partition count to 0
		metadata.PartitionCount = 0
	}

	// Get data size from system.parts (OTEL table: otel_logs)
	sizeQuery := `
		SELECT COALESCE(SUM(bytes_on_disk), 0) as data_size_bytes
		FROM system.parts
		WHERE table = 'otel_logs'
		AND database = ?
		AND active = 1
	`

	err = db.QueryRowContext(ctx, sizeQuery, r.config.Database).Scan(&metadata.DataSizeBytes)
	if err != nil {
		klog.ErrorS(err, "数据集大小统计查询失败", "dataset", dataset)
		// Don't fail completely, just set size to 0
		metadata.DataSizeBytes = 0
	}

	klog.V(4).InfoS("数据集统计信息获取完成",
		"dataset", dataset,
		"total_logs", metadata.TotalLogs,
		"partition_count", metadata.PartitionCount,
		"data_size_bytes", metadata.DataSizeBytes,
		"date_range_days", latest.Sub(earliest).Hours()/24)

	return metadata, nil
}

// ListAvailableDatasets returns a list of datasets with data (OTEL format)
func (r *ClickHouseRepository) ListAvailableDatasets(ctx context.Context) ([]string, error) {
	klog.V(4).InfoS("获取可用数据集列表")

	// OTEL format: ServiceName replaces dataset
	query := `
		SELECT DISTINCT ServiceName
		FROM otel_logs
		WHERE ServiceName != ''
		ORDER BY ServiceName
	`

	db := r.cm.GetDB()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		klog.ErrorS(err, "可用数据集列表查询失败")
		return nil, MapClickHouseError(err, "list_datasets").Err
	}
	defer rows.Close()

	var datasets []string
	for rows.Next() {
		var dataset string
		if err := rows.Scan(&dataset); err != nil {
			klog.ErrorS(err, "数据集列表扫描失败")
			return nil, MapClickHouseError(err, "scan_datasets").Err
		}
		datasets = append(datasets, dataset)
	}

	if err := rows.Err(); err != nil {
		klog.ErrorS(err, "数据集列表迭代失败")
		return nil, MapClickHouseError(err, "iterate_datasets").Err
	}

	klog.V(4).InfoS("可用数据集列表获取完成", "dataset_count", len(datasets))
	return datasets, nil
}

// QueryWithDataset ensures all repository queries are dataset-scoped (OTEL format)
func (r *ClickHouseRepository) QueryWithDataset(ctx context.Context, query string, dataset string, args ...interface{}) (*sql.Rows, error) {
	// Validate that query contains ServiceName filter for security (OTEL format)
	queryLower := strings.ToLower(query)
	if !strings.Contains(queryLower, "servicename = ?") && !strings.Contains(queryLower, "servicename=?") {
		return nil, fmt.Errorf("query must include ServiceName filter for security")
	}

	// Validate dataset parameter
	if dataset == "" {
		return nil, fmt.Errorf("dataset parameter cannot be empty")
	}

	// Execute query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	klog.V(5).InfoS("执行数据集作用域查询",
		"dataset", dataset,
		"query", r.sanitizeQueryForLog(query))

	db := r.cm.GetDB()

	// Prepend dataset to args (ServiceName in OTEL format)
	finalArgs := append([]interface{}{dataset}, args...)
	return db.QueryContext(queryCtx, query, finalArgs...)
}

// GetDatasetHealth performs health checks specific to a dataset (OTEL format)
func (r *ClickHouseRepository) GetDatasetHealth(ctx context.Context, dataset string) (*DatasetHealth, error) {
	klog.V(4).InfoS("执行数据集健康检查", "dataset", dataset)

	health := &DatasetHealth{
		Dataset:   dataset,
		Timestamp: time.Now(),
		Status:    "healthy",
	}

	// Check if dataset exists
	exists, err := r.DatasetExists(ctx, dataset)
	if err != nil {
		health.Status = "error"
		health.ErrorMessage = fmt.Sprintf("Failed to check dataset existence: %v", err)
		return health, nil
	}

	if !exists {
		health.Status = "not_found"
		health.ErrorMessage = "Dataset contains no data"
		return health, nil
	}

	// Check recent data availability (last 24 hours) - OTEL format
	recentDataQuery := `
		SELECT COUNT(*)
		FROM otel_logs
		WHERE ServiceName = ?
		AND Timestamp >= ?
	`

	since := time.Now().Add(-24 * time.Hour)
	var recentCount int64
	db := r.cm.GetDB()
	err = db.QueryRowContext(ctx, recentDataQuery, dataset, since).Scan(&recentCount)
	if err != nil {
		health.Status = "warning"
		health.ErrorMessage = fmt.Sprintf("Failed to check recent data: %v", err)
		return health, nil
	}

	health.RecentLogCount = recentCount
	if recentCount == 0 {
		health.Status = "warning"
		health.ErrorMessage = "No recent data (last 24 hours)"
	}

	// Check data freshness (most recent log timestamp) - OTEL format
	freshnessQuery := `
		SELECT MAX(Timestamp)
		FROM otel_logs
		WHERE ServiceName = ?
	`

	var lastTimestamp time.Time
	err = db.QueryRowContext(ctx, freshnessQuery, dataset).Scan(&lastTimestamp)
	if err != nil {
		health.Status = "warning"
		health.ErrorMessage = fmt.Sprintf("Failed to check data freshness: %v", err)
	} else {
		health.LastLogTimestamp = &lastTimestamp

		// Warn if data is more than 6 hours old
		if time.Since(lastTimestamp) > 6*time.Hour {
			health.Status = "warning"
			health.ErrorMessage = fmt.Sprintf("Data may be stale, last log: %v", lastTimestamp)
		}
	}

	klog.V(4).InfoS("数据集健康检查完成",
		"dataset", dataset,
		"status", health.Status,
		"recent_count", recentCount,
		"last_timestamp", lastTimestamp)

	return health, nil
}

// DatasetHealth represents health status of a dataset
type DatasetHealth struct {
	Dataset          string     `json:"dataset"`
	Status           string     `json:"status"` // healthy, warning, error, not_found
	Timestamp        time.Time  `json:"timestamp"`
	RecentLogCount   int64      `json:"recent_log_count"`
	LastLogTimestamp *time.Time `json:"last_log_timestamp,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
}

// sanitizeQueryForLog removes sensitive data from query for logging
func (r *ClickHouseRepository) sanitizeQueryForLog(query string) string {
	// Remove potential sensitive data while keeping structure
	sanitized := strings.ReplaceAll(query, "\n", " ")
	sanitized = strings.ReplaceAll(sanitized, "\t", " ")

	// Limit length for logging
	if len(sanitized) > 200 {
		sanitized = sanitized[:200] + "..."
	}

	return sanitized
}