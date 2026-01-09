package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/clickhouse"
	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/model/response"
	clickhouseRepo "github.com/outpostos/edge-logs/pkg/repository/clickhouse"
)

// Service provides log query business logic with comprehensive validation and transformation
type Service struct {
	repo              clickhouseRepo.Repository
	datasetValidator *DatasetValidator
}

// NewService creates a new query service with repository dependency
func NewService(repo clickhouseRepo.Repository) *Service {
	klog.InfoS("初始化日志查询服务")
	return &Service{
		repo:             repo,
		datasetValidator: NewDatasetValidator(),
	}
}

// QueryLogs queries logs with comprehensive business logic, validation, and transformation
func (s *Service) QueryLogs(ctx context.Context, req *request.LogQueryRequest) (*response.LogQueryResponse, error) {
	startTime := time.Now()

	klog.InfoS("开始日志查询服务",
		"dataset", req.Dataset,
		"start_time", req.StartTime,
		"end_time", req.EndTime,
		"filter", req.Filter,
		"namespace", req.Namespace,
		"page", req.Page,
		"page_size", req.PageSize)

	// Step 1: Comprehensive input validation
	if err := s.validateQueryRequest(req); err != nil {
		klog.ErrorS(err, "查询请求验证失败",
			"dataset", req.Dataset)
		return nil, NewValidationError("query_logs", err.Error())
	}

	// Step 2: Apply business logic filters and preprocessing
	if err := s.preprocessQueryRequest(req); err != nil {
		klog.ErrorS(err, "查询请求预处理失败",
			"dataset", req.Dataset)
		return nil, NewBusinessLogicError("preprocess_query", err.Error())
	}

	// Step 3: Execute repository query
	logs, total, err := s.repo.QueryLogs(ctx, req)
	if err != nil {
		klog.ErrorS(err, "仓储层查询失败",
			"dataset", req.Dataset)
		return nil, NewRepositoryError("query_logs", err)
	}

	// Step 4: Transform and enrich query results
	responseLogs, err := s.transformLogsToResponse(logs)
	if err != nil {
		klog.ErrorS(err, "日志转换失败",
			"dataset", req.Dataset,
			"log_count", len(logs))
		return nil, NewTransformationError("transform_logs", err.Error())
	}

	// Step 5: Build comprehensive response
	response := &response.LogQueryResponse{
		Logs:       responseLogs,
		TotalCount: total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		HasMore:    len(responseLogs) == req.PageSize && (req.Page*req.PageSize+len(responseLogs)) < total,
	}

	duration := time.Since(startTime)
	klog.InfoS("日志查询服务完成",
		"dataset", req.Dataset,
		"returned_logs", len(responseLogs),
		"total_count", total,
		"duration_ms", duration.Milliseconds(),
		"has_more", response.HasMore)

	// Log performance warnings
	if duration > 2*time.Second {
		klog.InfoS("查询响应时间超过目标",
			"dataset", req.Dataset,
			"duration_ms", duration.Milliseconds(),
			"target_ms", 2000)
	}

	return response, nil
}

// validateQueryRequest performs comprehensive input validation
func (s *Service) validateQueryRequest(req *request.LogQueryRequest) error {
	// Basic validation via model method
	if err := req.Validate(); err != nil {
		return err
	}

	// Additional service-level validation

	// Dataset security validation
	if err := s.validateDatasetAccess(req.Dataset); err != nil {
		return fmt.Errorf("dataset access denied: %w", err)
	}

	// Time range validation
	if req.StartTime != nil && req.EndTime != nil {
		duration := req.EndTime.Sub(*req.StartTime)
		if duration > 7*24*time.Hour {
			return fmt.Errorf("time range too large: %v, maximum allowed: 168h", duration)
		}
		if duration < 0 {
			return fmt.Errorf("start_time must be before end_time")
		}
	}

	// Filter parameter sanitization
	if req.Filter != "" {
		if len(req.Filter) < 2 {
			return fmt.Errorf("filter too short: minimum 2 characters")
		}
		if len(req.Filter) > 1000 {
			return fmt.Errorf("filter too long: maximum 1000 characters")
		}
		// Basic SQL injection protection
		if containsSQLInjection(req.Filter) {
			return fmt.Errorf("filter contains potentially harmful content")
		}
	}

	// Pagination validation
	if req.Page < 0 {
		return fmt.Errorf("page must be non-negative")
	}
	if req.PageSize <= 0 || req.PageSize > 10000 {
		return fmt.Errorf("page_size must be between 1 and 10000")
	}

	return nil
}

// validateDatasetAccess validates dataset access permissions using comprehensive validator
func (s *Service) validateDatasetAccess(dataset string) error {
	// Use the comprehensive dataset validator
	if err := s.datasetValidator.ValidateDataset(dataset); err != nil {
		return fmt.Errorf("dataset validation failed: %w", err)
	}

	// Additional service-level validations can be added here
	// TODO: Implement actual dataset authorization once auth system is in place
	// For now, dataset validator handles security validation

	return nil
}

// preprocessQueryRequest applies business logic preprocessing
func (s *Service) preprocessQueryRequest(req *request.LogQueryRequest) error {
	// Set default time range if not provided
	if req.StartTime == nil && req.EndTime == nil {
		now := time.Now()
		startTime := now.Add(-24 * time.Hour) // Default to last 24 hours
		req.StartTime = &startTime
		req.EndTime = &now

		klog.V(4).InfoS("应用默认时间范围",
			"dataset", req.Dataset,
			"start_time", startTime,
			"end_time", now)
	}

	// Set default page size if not provided
	if req.PageSize == 0 {
		req.PageSize = 100
	}

	// Normalize severity levels
	if req.Severity != "" {
		req.Severity = normalizeSeverityLevel(req.Severity)
	}

	return nil
}

// transformLogsToResponse converts repository logs to response format with enrichment
func (s *Service) transformLogsToResponse(logs []clickhouse.LogEntry) ([]response.LogEntry, error) {
	responseLogs := make([]response.LogEntry, 0, len(logs))

	for _, log := range logs {
		// Generate unique ID for log entry
		logID := generateLogID(log.Dataset, log.Timestamp, log.HostIP)

		// Map clickhouse fields to response fields
		responseLog := response.LogEntry{
			ID:        logID,
			Timestamp: log.Timestamp,
			Message:   log.Content,
			Level:     log.Severity,
			Namespace: log.K8sNamespace,
			Pod:       log.K8sPodName,
			Container: log.ContainerName,
			Labels:    enrichLabels(log.Tags, log),
		}

		responseLogs = append(responseLogs, responseLog)
	}

	return responseLogs, nil
}

// Helper functions

// containsSQLInjection performs basic SQL injection detection
func containsSQLInjection(input string) bool {
	// Basic SQL injection patterns
	dangerousPatterns := []string{
		"'; DROP", "'; DELETE", "'; UPDATE", "'; INSERT",
		"UNION SELECT", "OR 1=1", "AND 1=1",
		"')", "';--", "/*", "*/",
	}

	for _, pattern := range dangerousPatterns {
		if containsIgnoreCase(input, pattern) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if string contains substring (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// normalizeSeverityLevel normalizes severity levels to standard format
func normalizeSeverityLevel(level string) string {
	switch strings.ToLower(level) {
	case "error", "err", "e":
		return "ERROR"
	case "warn", "warning", "w":
		return "WARN"
	case "info", "information", "i":
		return "INFO"
	case "debug", "d":
		return "DEBUG"
	default:
		return strings.ToUpper(level)
	}
}

// generateLogID creates a unique ID for a log entry
func generateLogID(dataset string, timestamp time.Time, hostIP string) string {
	return fmt.Sprintf("%s-%d-%s", dataset, timestamp.UnixNano(), hostIP)
}

// enrichLabels enriches response labels with additional metadata
func enrichLabels(originalTags map[string]string, log clickhouse.LogEntry) map[string]string {
	labels := make(map[string]string)

	// Copy original tags
	for k, v := range originalTags {
		labels[k] = v
	}

	// Add enrichment metadata
	if log.HostName != "" {
		labels["host_name"] = log.HostName
	}
	if log.K8sNodeName != "" {
		labels["node_name"] = log.K8sNodeName
	}
	if log.K8sPodUID != "" {
		labels["pod_uid"] = log.K8sPodUID
	}

	return labels
}

// QueryLogsByDataset queries logs with strict dataset scoping and existence validation
func (s *Service) QueryLogsByDataset(ctx context.Context, req *request.LogQueryRequest) (*response.LogQueryResponse, error) {
	startTime := time.Now()

	klog.InfoS("开始数据集作用域日志查询",
		"dataset", req.Dataset,
		"start_time", req.StartTime,
		"end_time", req.EndTime,
		"filter", req.Filter,
		"namespace", req.Namespace)

	// Step 1: Validate dataset parameter is present
	if req.Dataset == "" {
		return nil, NewValidationError("query_logs_by_dataset", "dataset parameter is required")
	}

	// Step 2: Enhanced dataset validation including existence check
	if err := s.validateDatasetWithExistence(ctx, req.Dataset); err != nil {
		klog.ErrorS(err, "数据集验证失败", "dataset", req.Dataset)
		return nil, err
	}

	// Step 3: Execute standard query with existing validation
	response, err := s.QueryLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	// Step 4: Enhance response with dataset metadata
	if err := s.enrichResponseWithDataset(ctx, response, req.Dataset); err != nil {
		klog.ErrorS(err, "数据集元数据增强失败", "dataset", req.Dataset)
		// Don't fail the query, just log the error
	}

	duration := time.Since(startTime)
	klog.InfoS("数据集作用域日志查询完成",
		"dataset", req.Dataset,
		"returned_logs", len(response.Logs),
		"total_count", response.TotalCount,
		"duration_ms", duration.Milliseconds())

	return response, nil
}

// validateDatasetWithExistence validates dataset and checks existence
func (s *Service) validateDatasetWithExistence(ctx context.Context, dataset string) error {
	// Basic validation
	if err := s.validateDatasetAccess(dataset); err != nil {
		return err
	}

	// Check if dataset exists and contains data
	if repo, ok := s.repo.(*clickhouseRepo.ClickHouseRepository); ok {
		exists, err := repo.DatasetExists(ctx, dataset)
		if err != nil {
			return NewRepositoryError("dataset_existence_check", err)
		}
		if !exists {
			return NewValidationError("dataset_not_found", fmt.Sprintf("dataset '%s' not found or contains no data", dataset))
		}
	}

	return nil
}

// enrichResponseWithDataset adds dataset metadata to response
func (s *Service) enrichResponseWithDataset(ctx context.Context, response *response.LogQueryResponse, dataset string) error {
	// Try to get dataset metadata if repository supports it
	if repo, ok := s.repo.(*clickhouseRepo.ClickHouseRepository); ok {
		metadata, err := repo.GetDatasetStats(ctx, dataset)
		if err != nil {
			return fmt.Errorf("failed to get dataset metadata: %w", err)
		}

		// Add dataset information to response (this will be implemented when we enhance response structure)
		klog.V(4).InfoS("数据集元数据获取成功",
			"dataset", dataset,
			"total_logs", metadata.TotalLogs,
			"partition_count", metadata.PartitionCount)
	}

	return nil
}

// DatasetExists checks if a dataset exists and contains data
func (s *Service) DatasetExists(ctx context.Context, dataset string) (bool, error) {
	// Validate dataset format first
	if err := s.datasetValidator.ValidateDataset(dataset); err != nil {
		return false, fmt.Errorf("invalid dataset format: %w", err)
	}

	// Check existence in repository
	if repo, ok := s.repo.(*clickhouseRepo.ClickHouseRepository); ok {
		return repo.DatasetExists(ctx, dataset)
	}

	return false, fmt.Errorf("repository does not support dataset existence checking")
}
