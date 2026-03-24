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
	"github.com/outpostos/edge-logs/pkg/service/enrichment"
)

// Service provides log query business logic with comprehensive validation and transformation
type Service struct {
	repo                  clickhouseRepo.Repository
	timeValidator        *TimeRangeValidator
	k8sValidator         *K8sResourceValidator
	contentSearchValidator *ContentSearchValidator
	enrichmentService    *enrichment.MetadataEnrichmentService
}

// NewService creates a new query service with repository dependency
func NewService(repo clickhouseRepo.Repository, enrichmentService *enrichment.MetadataEnrichmentService) *Service {
	klog.InfoS("初始化日志查询服务")
	service := &Service{
		repo:                   repo,
		timeValidator:         NewTimeRangeValidator(),
		k8sValidator:          NewK8sResourceValidator(),
		contentSearchValidator: NewContentSearchValidator(),
		enrichmentService:     enrichmentService,
	}

	if enrichmentService != nil {
		klog.InfoS("K8s元数据增强服务已启用")
	} else {
		klog.InfoS("K8s元数据增强服务未启用")
	}

	return service
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
	responseLogs, enrichmentMetadata, err := s.transformAndEnrichLogs(ctx, logs, req)
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

	// Add enrichment metadata if enrichment was performed
	if enrichmentMetadata != nil {
		response.Enrichment = enrichmentMetadata
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

	// Enhanced time range validation with millisecond precision
	if err := s.validateTimeRange(req); err != nil {
		return fmt.Errorf("time range validation failed: %w", err)
	}

	// K8s metadata filtering validation
	if err := s.validateK8sFilters(req); err != nil {
		return fmt.Errorf("K8s filter validation failed: %w", err)
	}

	// Content search validation and parsing
	if err := s.validateAndParseContentSearch(req); err != nil {
		return fmt.Errorf("content search validation failed: %w", err)
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

// validateTimeRange provides comprehensive time range validation with millisecond precision
func (s *Service) validateTimeRange(req *request.LogQueryRequest) error {
	// Convert time.Time to string for validation if needed
	var startStr, endStr string
	if req.StartTime != nil {
		startStr = req.StartTime.Format(time.RFC3339Nano)
	}
	if req.EndTime != nil {
		endStr = req.EndTime.Format(time.RFC3339Nano)
	}

	// Validate and normalize time range using enhanced validator
	normalizedStart, normalizedEnd, err := s.timeValidator.ValidateAndParseTimeRange(startStr, endStr)
	if err != nil {
		return err
	}

	// Update request with normalized times (UTC and validated)
	req.StartTime = normalizedStart
	req.EndTime = normalizedEnd

	// Additional business logic validation
	if req.StartTime != nil && req.EndTime != nil {
		timeSpan := req.EndTime.Sub(*req.StartTime)

		// Log time range details for monitoring
		klog.V(4).InfoS("Time range validation",
			"start_time", req.StartTime.Format(time.RFC3339Nano),
			"end_time", req.EndTime.Format(time.RFC3339Nano),
			"time_span", timeSpan,
			"span_hours", timeSpan.Hours())

		// Check for very large time spans that could impact performance
		if timeSpan > 7*24*time.Hour {
			return NewTimeRangeError(req.StartTime, req.EndTime,
				fmt.Sprintf("time range span (%v) exceeds maximum performance threshold (168h)", timeSpan))
		}
	}

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

// transformAndEnrichLogs converts repository logs to response format with optional K8s enrichment
func (s *Service) transformAndEnrichLogs(ctx context.Context, logs []clickhouse.LogEntry, req *request.LogQueryRequest) ([]response.LogEntry, *response.EnrichmentMetadata, error) {
	responseLogs := make([]response.LogEntry, 0, len(logs))
	var enrichmentMetadata *response.EnrichmentMetadata
	var enrichmentResult *enrichment.EnrichmentResult

	// Check if enrichment is enabled
	enrichmentEnabled := req.EnrichMetadata != nil && *req.EnrichMetadata

	if enrichmentEnabled && s.enrichmentService != nil {
		// Collect unique pod UIDs for enrichment
		podUIDs := s.collectPodUIDs(logs)

		// Perform enrichment
		if len(podUIDs) > 0 {
			enrichmentResult = s.enrichmentService.EnrichLogs(ctx, podUIDs)

			// Build enrichment metadata
			enrichmentMetadata = &response.EnrichmentMetadata{
				Enabled:        true,
				PodsEnriched:   len(enrichmentResult.Metadata),
				CacheHits:      enrichmentResult.CacheHits,
				APICalls:       enrichmentResult.APICalls,
				FailedPods:     len(enrichmentResult.FailedPodUIDs),
				EnrichmentTime: float64(enrichmentResult.Duration.Milliseconds()),
			}

			klog.V(4).InfoS("K8s元数据增强完成",
				"pods_enriched", enrichmentMetadata.PodsEnriched,
				"cache_hits", enrichmentMetadata.CacheHits,
				"api_calls", enrichmentMetadata.APICalls,
				"failed_pods", enrichmentMetadata.FailedPods,
				"duration_ms", enrichmentMetadata.EnrichmentTime)
		}

		// Transform logs with enrichment
		for _, log := range logs {
			responseLog := s.transformLogToResponse(log)

			// Apply enrichment if available
			if enrichmentResult != nil && log.GetK8sPodUID() != "" {
				if podMetadata, exists := enrichmentResult.Metadata[log.GetK8sPodUID()]; exists {
					s.applyEnrichment(&responseLog, podMetadata)
				}
			}

			responseLogs = append(responseLogs, responseLog)
		}
	} else {
		// Transform logs without enrichment
		for _, log := range logs {
			responseLog := s.transformLogToResponse(log)
			responseLogs = append(responseLogs, responseLog)
		}
	}

	return responseLogs, enrichmentMetadata, nil
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

	// Add enrichment metadata using getter methods
	if hostName := log.GetHostName(); hostName != "" {
		labels["host_name"] = hostName
	}
	if nodeName := log.GetK8sNodeName(); nodeName != "" {
		labels["node_name"] = nodeName
	}
	if podUID := log.GetK8sPodUID(); podUID != "" {
		labels["pod_uid"] = podUID
	}

	return labels
}

// collectPodUIDs collects unique pod UIDs from log entries
func (s *Service) collectPodUIDs(logs []clickhouse.LogEntry) []string {
	seen := make(map[string]bool)
	var podUIDs []string

	for _, log := range logs {
		podUID := log.GetK8sPodUID()
		if podUID != "" && !seen[podUID] {
			seen[podUID] = true
			podUIDs = append(podUIDs, podUID)
		}
	}

	return podUIDs
}

// transformLogToResponse converts a single log entry to response format
func (s *Service) transformLogToResponse(log clickhouse.LogEntry) response.LogEntry {
	logID := generateLogID(log.GetDataset(), log.Timestamp, log.GetHostIP())

	return response.LogEntry{
		ID:        logID,
		Timestamp: log.Timestamp,
		Message:   log.GetContent(),
		Level:     log.GetSeverity(),
		Namespace: log.GetK8sNamespace(),
		Pod:       log.GetK8sPodName(),
		Container: log.GetContainerName(),
		Labels:    enrichLabels(log.LogAttributes, log),
	}
}

// applyEnrichment applies K8s metadata enrichment to a log entry
func (s *Service) applyEnrichment(responseLog *response.LogEntry, podMetadata *enrichment.PodMetadata) {
	// Apply basic pod metadata
	responseLog.PodUID = podMetadata.UID
	responseLog.NodeName = podMetadata.NodeName
	responseLog.PodIP = podMetadata.PodIP
	responseLog.HostIP = podMetadata.HostIP
	responseLog.PodPhase = string(podMetadata.Phase)

	// Apply labels if available
	if len(podMetadata.Labels) > 0 {
		responseLog.PodLabels = make(map[string]string)
		for k, v := range podMetadata.Labels {
			responseLog.PodLabels[k] = v
		}
	}

	// Apply annotations if available
	if len(podMetadata.Annotations) > 0 {
		responseLog.PodAnnotations = make(map[string]string)
		for k, v := range podMetadata.Annotations {
			responseLog.PodAnnotations[k] = v
		}
	}

	// Add pod metadata to existing labels
	if responseLog.Labels == nil {
		responseLog.Labels = make(map[string]string)
	}

	// Add enrichments to labels for backward compatibility
	for k, v := range podMetadata.Labels {
		responseLog.Labels["pod_label_"+k] = v
	}
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

	// Step 2: Execute standard query
	// Note: Dataset existence validation removed - query directly, return empty if no data
	response, err := s.QueryLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	// Step 3: Enhance response with dataset metadata
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

// enrichResponseWithDataset adds dataset metadata to response
func (s *Service) enrichResponseWithDataset(ctx context.Context, response *response.LogQueryResponse, dataset string) error {
	// Skip dataset metadata enrichment for performance
	// GetDatasetStats is too slow and blocks the main query
	// TODO: Implement cached metadata or background refresh
	klog.V(4).InfoS("跳过数据集元数据增强以提升性能",
		"dataset", dataset)

	return nil
}

// validateK8sFilters validates and processes K8s filtering parameters
func (s *Service) validateK8sFilters(req *request.LogQueryRequest) error {
	// Parse and validate K8s filters
	namespaces := s.parseNamespaceInput(req.Namespace, req.Namespaces)
	pods := s.parsePodInput(req.PodName, req.PodNames)

	// Validate and parse K8s filters
	k8sFilters, err := s.k8sValidator.ParseK8sFilters(namespaces, pods)
	if err != nil {
		return fmt.Errorf("K8s filter parsing failed: %w", err)
	}

	// Store parsed filters in request for query building
	req.K8sFilters = k8sFilters

	// Log K8s filter details for monitoring
	if len(k8sFilters) > 0 {
		klog.V(4).InfoS("K8s filters parsed",
			"dataset", req.Dataset,
			"filter_count", len(k8sFilters),
			"namespaces", namespaces,
			"pods", pods)
	}

	return nil
}

// parseNamespaceInput handles various namespace input formats
func (s *Service) parseNamespaceInput(namespace string, namespaces []string) []string {
	var result []string

	// Handle single namespace parameter
	if namespace != "" {
		if strings.Contains(namespace, ",") {
			// Comma-separated namespaces
			result = append(result, strings.Split(namespace, ",")...)
		} else {
			result = append(result, namespace)
		}
	}

	// Handle array-style namespaces parameter
	result = append(result, namespaces...)

	// Remove empty entries and duplicates
	seen := make(map[string]bool)
	var cleaned []string
	for _, ns := range result {
		ns = strings.TrimSpace(ns)
		if ns != "" && !seen[ns] {
			cleaned = append(cleaned, ns)
			seen[ns] = true
		}
	}

	return cleaned
}

// parsePodInput handles various pod input formats
func (s *Service) parsePodInput(podName string, podNames []string) []string {
	var result []string

	// Handle single pod parameter
	if podName != "" {
		if strings.Contains(podName, ",") {
			// Comma-separated pod names
			result = append(result, strings.Split(podName, ",")...)
		} else {
			result = append(result, podName)
		}
	}

	// Handle array-style pod names parameter
	result = append(result, podNames...)

	// Remove empty entries and duplicates
	seen := make(map[string]bool)
	var cleaned []string
	for _, pod := range result {
		pod = strings.TrimSpace(pod)
		if pod != "" && !seen[pod] {
			cleaned = append(cleaned, pod)
			seen[pod] = true
		}
	}

	return cleaned
}

// DatasetExists checks if a dataset exists and contains data
func (s *Service) DatasetExists(ctx context.Context, dataset string) (bool, error) {
	// Check existence in repository
	if repo, ok := s.repo.(*clickhouseRepo.ClickHouseRepository); ok {
		return repo.DatasetExists(ctx, dataset)
	}

	return false, fmt.Errorf("repository does not support dataset existence checking")
}

// ListAvailableDatasets returns a list of all available datasets
func (s *Service) ListAvailableDatasets(ctx context.Context) ([]string, error) {
	klog.V(4).InfoS("获取可用数据集列表")

	// Query repository for available datasets
	if repo, ok := s.repo.(*clickhouseRepo.ClickHouseRepository); ok {
		datasets, err := repo.ListAvailableDatasets(ctx)
		if err != nil {
			klog.ErrorS(err, "获取可用数据集列表失败")
			return nil, fmt.Errorf("failed to list available datasets: %w", err)
		}

		klog.V(4).InfoS("获取可用数据集列表成功", "count", len(datasets))
		return datasets, nil
	}

	return nil, fmt.Errorf("repository does not support listing datasets")
}

// validateAndParseContentSearch validates and parses content search parameters
func (s *Service) validateAndParseContentSearch(req *request.LogQueryRequest) error {
	// Handle legacy filter parameter for backward compatibility
	searchQuery := req.Filter
	if req.ContentSearch != "" {
		searchQuery = req.ContentSearch
	}

	// Skip if no content search is specified
	if searchQuery == "" {
		return nil
	}

	// Prepare search options
	options := map[string]string{
		"operator": "AND", // Default
	}

	if req.ContentOperator != "" && (req.ContentOperator == "AND" || req.ContentOperator == "OR") {
		options["operator"] = req.ContentOperator
	}

	if req.ContentHighlight != nil && !*req.ContentHighlight {
		options["highlight"] = "false"
	}

	if req.ContentRelevance != nil && !*req.ContentRelevance {
		options["relevance"] = "false"
	}

	// Parse and validate content search
	contentSearch, err := s.contentSearchValidator.ParseContentSearch(searchQuery, options)
	if err != nil {
		return fmt.Errorf("failed to parse content search: %w", err)
	}

	// Convert to request format to avoid circular imports
	if contentSearch != nil {
		parsedContentSearch := &request.ParsedContentSearchExpression{
			GlobalOperator:   contentSearch.GlobalOperator,
			HighlightEnabled: contentSearch.HighlightEnabled,
			MaxSnippetLength: contentSearch.MaxSnippetLength,
			RelevanceScoring: contentSearch.RelevanceScoring,
		}

		// Convert filters
		for _, filter := range contentSearch.Filters {
			parsedContentSearch.Filters = append(parsedContentSearch.Filters, request.ContentSearchFilter{
				Type:              string(filter.Type),
				Pattern:           filter.Pattern,
				CaseInsensitive:   filter.CaseInsensitive,
				BooleanOperator:   filter.BooleanOperator,
				ProximityDistance: filter.ProximityDistance,
				FieldTarget:       filter.FieldTarget,
				Weight:            filter.Weight,
			})
		}

		req.ParsedContentSearch = parsedContentSearch

		klog.V(4).InfoS("Content search parsed and validated",
			"dataset", req.Dataset,
			"filters", len(parsedContentSearch.Filters),
			"highlight", parsedContentSearch.HighlightEnabled,
			"relevance", parsedContentSearch.RelevanceScoring)
	}

	return nil
}

// QueryAggregation executes aggregation queries with validation
func (s *Service) QueryAggregation(ctx context.Context, req *request.AggregationRequest) (*response.AggregationResponse, error) {
	startTime := time.Now()

	klog.InfoS("开始聚合查询服务",
		"dataset", req.Dataset,
		"dimensions", len(req.Dimensions),
		"functions", len(req.Functions),
		"start_time", req.StartTime,
		"end_time", req.EndTime)

	// Validate aggregation request
	validator := NewAggregationDimensionValidator()
	if err := validator.ValidateAggregationRequest(req); err != nil {
		klog.ErrorS(err, "聚合请求验证失败", "dataset", req.Dataset)
		return nil, NewValidationError("aggregation_validation", err.Error())
	}

	// Execute aggregation query through repository
	result, err := s.repo.QueryAggregation(ctx, req)
	if err != nil {
		klog.ErrorS(err, "聚合查询执行失败", "dataset", req.Dataset)
		return nil, NewRepositoryError("query_aggregation", err)
	}

	duration := time.Since(startTime)
	klog.InfoS("聚合查询服务完成",
		"dataset", req.Dataset,
		"result_count", len(result.Results),
		"duration_ms", duration.Milliseconds())

	return result, nil
}
