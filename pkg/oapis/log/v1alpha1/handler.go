package v1alpha1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/model/response"
	responseWrapper "github.com/outpostos/edge-logs/pkg/response"
	query "github.com/outpostos/edge-logs/pkg/service/query"
)

// LogHandler handles log API requests with K8s API Aggregation pattern
type LogHandler struct {
	queryService       *query.Service
	metrics           *DatasetMetrics
	timeMetrics       *TimeMetrics
	k8sMetrics        *K8sMetrics
	contentMetrics    *ContentSearchMetrics
}

// NewLogHandler creates a new log handler with service dependency
func NewLogHandler(queryService *query.Service) *LogHandler {
	klog.InfoS("初始化日志 API 处理器")
	return &LogHandler{
		queryService:   queryService,
		metrics:        NewDatasetMetrics(),
		timeMetrics:    NewTimeMetrics(),
		k8sMetrics:     NewK8sMetrics(),
		contentMetrics: NewContentSearchMetrics(),
	}
}

// InstallHandler installs log API routes with K8s API Aggregation pattern
func (h *LogHandler) InstallHandler(container *restful.Container) {
	klog.InfoS("安装日志 API 处理器",
		"api_group", "log.theriseunion.io",
		"version", "v1alpha1")

	// Create new WebService for log.theriseunion.io/v1alpha1
	ws := new(restful.WebService)
	ws.Path("/apis/log.theriseunion.io/v1alpha1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// Main log query endpoint with dataset-based routing
	ws.Route(ws.GET("/logdatasets/{dataset}/logs").To(h.queryLogs).
		Doc("查询边缘计算日志").
		Notes("根据数据集、时间范围、命名空间、Pod名称等条件查询日志").
		Param(ws.PathParameter("dataset", "数据集名称").DataType("string").Required(true)).
		Param(ws.QueryParameter("start_time", "开始时间 (ISO 8601格式)").DataType("string")).
		Param(ws.QueryParameter("end_time", "结束时间 (ISO 8601格式)").DataType("string")).
		Param(ws.QueryParameter("namespace", "单个Kubernetes命名空间").DataType("string")).
		Param(ws.QueryParameter("namespaces", "多个Kubernetes命名空间 (逗号分隔或数组)").DataType("string")).
		Param(ws.QueryParameter("pod_name", "单个Pod名称").DataType("string")).
		Param(ws.QueryParameter("pod_names", "多个Pod名称 (支持模式匹配: exact, prefix*, *suffix, *contains*, regex:pattern, icase:pattern)").DataType("string")).
		Param(ws.QueryParameter("pods", "Pod名称数组 (同pod_names)").DataType("string")).
		Param(ws.QueryParameter("node_name", "节点名称").DataType("string")).
		Param(ws.QueryParameter("container_name", "容器名称").DataType("string")).
		Param(ws.QueryParameter("filter", "日志内容过滤 (向后兼容)").DataType("string")).
		Param(ws.QueryParameter("content_search", "高级内容搜索 (支持: exact, icase:term, *wildcard*, regex:pattern, \"phrase search\", boolean:expr, proximity:5:terms)").DataType("string")).
		Param(ws.QueryParameter("content_highlight", "启用搜索结果高亮 (true/false)").DataType("boolean")).
		Param(ws.QueryParameter("content_relevance", "启用相关性评分 (true/false)").DataType("boolean")).
		Param(ws.QueryParameter("content_operator", "默认布尔运算符 (AND/OR)").DataType("string")).
		Param(ws.QueryParameter("severity", "日志级别").DataType("string")).
		Param(ws.QueryParameter("page", "页码 (从0开始)").DataType("integer")).
		Param(ws.QueryParameter("page_size", "每页大小").DataType("integer")).
		Param(ws.QueryParameter("order_by", "排序字段 (timestamp, severity)").DataType("string")).
		Param(ws.QueryParameter("direction", "排序方向 (asc, desc)").DataType("string")).
		Returns(http.StatusOK, "查询成功", response.LogQueryResponse{}).
		Returns(http.StatusBadRequest, "请求参数错误", responseWrapper.ErrorResponse{}).
		Returns(http.StatusNotFound, "数据集不存在", responseWrapper.ErrorResponse{}).
		Returns(http.StatusInternalServerError, "服务器内部错误", responseWrapper.ErrorResponse{}))

	// Health check endpoint
	ws.Route(ws.GET("/health").To(h.healthCheck).
		Doc("健康检查").
		Returns(http.StatusOK, "服务正常", responseWrapper.HealthResponse{}))

	container.Add(ws)

	klog.InfoS("日志 API 处理器安装完成",
		"endpoints", 2,
		"base_path", "/apis/log.theriseunion.io/v1alpha1")
}

// queryLogs handles log query requests with comprehensive parameter parsing and error handling
func (h *LogHandler) queryLogs(req *restful.Request, resp *restful.Response) {
	startTime := time.Now()

	// Extract dataset from path parameter
	dataset := req.PathParameter("dataset")

	klog.InfoS("开始处理日志查询请求",
		"dataset", dataset,
		"method", req.Request.Method,
		"url", req.Request.URL.String())

	// Build query request from HTTP parameters
	queryReq, err := h.parseQueryRequest(req, dataset)
	if err != nil {
		klog.ErrorS(err, "查询请求解析失败",
			"dataset", dataset)
		h.writeErrorResponse(resp, http.StatusBadRequest, fmt.Sprintf("参数解析失败: %v", err))
		return
	}

	// Validate dataset before executing query
	if err := h.validateDataset(dataset); err != nil {
		klog.ErrorS(err, "数据集验证失败", "dataset", dataset)
		h.handleDatasetError(resp, err, dataset)
		return
	}

	// Execute dataset-scoped query through service layer
	result, err := h.queryService.QueryLogsByDataset(req.Request.Context(), queryReq)
	if err != nil {
		duration := time.Since(startTime)
		klog.ErrorS(err, "日志查询执行失败",
			"dataset", dataset,
			"duration_ms", duration.Milliseconds())
		h.handleServiceError(resp, err, dataset)
		h.metrics.RecordDatasetError(dataset, "query_execution_failed")
		return
	}

	// Enhance response with dataset metadata
	h.enrichResponseWithDataset(result, dataset, queryReq)

	duration := time.Since(startTime)
	klog.InfoS("日志查询请求处理成功",
		"dataset", dataset,
		"returned_logs", len(result.Logs),
		"total_count", result.TotalCount,
		"duration_ms", duration.Milliseconds())

	// Record successful metrics
	h.metrics.RecordDatasetSuccess(dataset, len(result.Logs), duration)

	// Record time-specific metrics
	if queryReq.StartTime != nil && queryReq.EndTime != nil {
		timeSpan := queryReq.EndTime.Sub(*queryReq.StartTime)
		h.timeMetrics.RecordTimeQuery(dataset, duration, timeSpan, len(result.Logs))
	}

	// Record K8s-specific metrics
	if len(queryReq.K8sFilters) > 0 {
		h.k8sMetrics.RecordK8sQuery(dataset, duration, queryReq.K8sFilters, len(result.Logs))
	}

	// Log performance warnings
	if duration > 2*time.Second {
		klog.InfoS("查询响应时间超过目标",
			"dataset", dataset,
			"duration_ms", duration.Milliseconds(),
			"target_ms", 2000)
	}

	// Write successful response
	resp.WriteHeaderAndEntity(http.StatusOK, result)
}

// healthCheck handles health check requests
func (h *LogHandler) healthCheck(req *restful.Request, resp *restful.Response) {
	klog.V(4).InfoS("健康检查请求")

	healthResponse := &responseWrapper.HealthResponse{
		Status:  "healthy",
		Version: "v1alpha1",
		Service: "edge-logs-api",
	}

	resp.WriteHeaderAndEntity(http.StatusOK, healthResponse)
}

// parseQueryRequest parses HTTP request parameters into LogQueryRequest
func (h *LogHandler) parseQueryRequest(req *restful.Request, dataset string) (*request.LogQueryRequest, error) {
	queryReq := &request.LogQueryRequest{
		Dataset: dataset,
		Tags:    make(map[string]string),
	}

	// Parse time parameters with enhanced millisecond precision support
	startTime, endTime, err := h.parseTimeParameters(req)
	if err != nil {
		return nil, err
	}
	queryReq.StartTime = startTime
	queryReq.EndTime = endTime

	// Parse filter parameters
	queryReq.Filter = req.QueryParameter("filter")
	queryReq.Severity = req.QueryParameter("severity")
	queryReq.NodeName = req.QueryParameter("node_name")
	queryReq.ContainerName = req.QueryParameter("container_name")

	// Parse advanced content search parameters
	queryReq.ContentSearch = req.QueryParameter("content_search")
	queryReq.ContentOperator = req.QueryParameter("content_operator")

	// Parse boolean parameters for content search
	if highlightStr := req.QueryParameter("content_highlight"); highlightStr != "" {
		highlight, err := strconv.ParseBool(highlightStr)
		if err != nil {
			return nil, fmt.Errorf("content_highlight 参数错误: %w", err)
		}
		queryReq.ContentHighlight = &highlight
	}

	if relevanceStr := req.QueryParameter("content_relevance"); relevanceStr != "" {
		relevance, err := strconv.ParseBool(relevanceStr)
		if err != nil {
			return nil, fmt.Errorf("content_relevance 参数错误: %w", err)
		}
		queryReq.ContentRelevance = &relevance
	}

	// Parse K8s parameters with enhanced pattern support
	namespaces, pods, err := h.parseK8sParameters(req)
	if err != nil {
		return nil, fmt.Errorf("K8s参数解析失败: %w", err)
	}

	// Set legacy single parameters for backward compatibility
	queryReq.Namespace = req.QueryParameter("namespace")
	queryReq.PodName = req.QueryParameter("pod_name")

	// Set enhanced multi-value parameters
	queryReq.Namespaces = namespaces
	queryReq.PodNames = pods

	// Parse pagination parameters
	if pageStr := req.QueryParameter("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			return nil, fmt.Errorf("page 参数错误: %w", err)
		}
		if page < 0 {
			return nil, fmt.Errorf("page 参数必须 >= 0")
		}
		queryReq.Page = page
	}

	if pageSizeStr := req.QueryParameter("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			return nil, fmt.Errorf("page_size 参数错误: %w", err)
		}
		if pageSize <= 0 || pageSize > 10000 {
			return nil, fmt.Errorf("page_size 参数必须在 1-10000 之间")
		}
		queryReq.PageSize = pageSize
	}

	// Parse ordering parameters
	queryReq.OrderBy = req.QueryParameter("order_by")
	queryReq.Direction = req.QueryParameter("direction")

	klog.V(4).InfoS("查询请求解析完成",
		"dataset", dataset,
		"start_time", queryReq.StartTime,
		"end_time", queryReq.EndTime,
		"page", queryReq.Page,
		"page_size", queryReq.PageSize)

	return queryReq, nil
}

// writeErrorResponse writes error responses in consistent format
func (h *LogHandler) writeErrorResponse(resp *restful.Response, statusCode int, message string) {
	errorResponse := &responseWrapper.ErrorResponse{
		Code:    statusCode,
		Message: message,
	}

	resp.WriteHeaderAndEntity(statusCode, errorResponse)
}

// mapErrorToStatusCode maps service errors to appropriate HTTP status codes
func (h *LogHandler) mapErrorToStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	errMsg := err.Error()

	// Check for specific error patterns
	switch {
	case strings.Contains(errMsg, "dataset is required"),
		strings.Contains(errMsg, "参数"),
		strings.Contains(errMsg, "格式错误"),
		strings.Contains(errMsg, "参数错误"):
		return http.StatusBadRequest

	case strings.Contains(errMsg, "not found"),
		strings.Contains(errMsg, "不存在"):
		return http.StatusNotFound

	case strings.Contains(errMsg, "connection"),
		strings.Contains(errMsg, "timeout"),
		strings.Contains(errMsg, "connection refused"):
		return http.StatusServiceUnavailable

	default:
		return http.StatusInternalServerError
	}
}


// validateDataset validates dataset parameter with comprehensive rules
func (h *LogHandler) validateDataset(dataset string) error {
	// Basic format validation
	if dataset == "" {
		return NewDatasetValidationError(dataset, "dataset parameter is required")
	}

	// Use service layer for dataset existence checking
	exists, err := h.queryService.DatasetExists(context.Background(), dataset)
	if err != nil {
		return NewDatasetValidationError(dataset, fmt.Sprintf("failed to validate dataset: %v", err))
	}

	if !exists {
		return NewDatasetNotFoundError(dataset)
	}

	return nil
}

// handleDatasetError handles dataset-specific errors with appropriate HTTP responses
func (h *LogHandler) handleDatasetError(resp *restful.Response, err error, dataset string) {
	statusCode := MapDatasetErrorToHTTPStatus(err)
	message := GetDatasetErrorMessage(err, dataset)

	h.writeErrorResponse(resp, statusCode, message)

	// Record error metrics
	var errorType string
	switch err.(type) {
	case *DatasetNotFoundError:
		errorType = "not_found"
	case *DatasetUnauthorizedError:
		errorType = "unauthorized"
	case *DatasetValidationError:
		errorType = "validation_failed"
	case *DatasetSecurityError:
		errorType = "security_violation"
	default:
		errorType = "unknown"
	}

	h.metrics.RecordDatasetError(dataset, errorType)
}

// handleServiceError handles service layer errors
func (h *LogHandler) handleServiceError(resp *restful.Response, err error, dataset string) {
	errMsg := err.Error()

	// Check for K8s-specific errors
	if isK8sError(errMsg) {
		h.handleK8sServiceError(resp, err, dataset)
		return
	}

	// Handle general service errors
	statusCode := h.mapErrorToStatusCode(err)
	message := fmt.Sprintf("查询失败: %v", err)

	h.writeErrorResponse(resp, statusCode, message)
}

// isK8sError checks if an error is K8s-related
func isK8sError(errMsg string) bool {
	k8sErrorPatterns := []string{
		"K8s filter validation failed",
		"invalid namespace filter",
		"invalid pod filter",
		"K8s filter parsing failed",
		"too many K8s filters",
		"K8s filter complexity too high",
		"DNS-1123 compliant",
		"regex pattern may be too expensive",
	}

	for _, pattern := range k8sErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

// handleK8sServiceError handles K8s-specific service errors
func (h *LogHandler) handleK8sServiceError(resp *restful.Response, err error, dataset string) {
	statusCode, errorResponse := h.HandleK8sError(err, dataset)

	// Record K8s error metrics
	errorType := h.categorizeK8sError(err)
	errorReason := h.extractK8sErrorReason(err)
	h.k8sMetrics.RecordK8sError(dataset, errorType, errorReason)

	resp.WriteHeaderAndEntity(statusCode, errorResponse)
}

// categorizeK8sError categorizes K8s errors for metrics
func (h *LogHandler) categorizeK8sError(err error) string {
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "validation failed"):
		return "validation_error"
	case strings.Contains(errMsg, "complexity too high"):
		return "complexity_error"
	case strings.Contains(errMsg, "DNS-1123"):
		return "format_error"
	case strings.Contains(errMsg, "regex pattern"):
		return "pattern_error"
	case strings.Contains(errMsg, "too many"):
		return "limit_exceeded"
	default:
		return "unknown_error"
	}
}

// extractK8sErrorReason extracts specific error reason for metrics
func (h *LogHandler) extractK8sErrorReason(err error) string {
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "namespace"):
		return "namespace_issue"
	case strings.Contains(errMsg, "pod"):
		return "pod_issue"
	case strings.Contains(errMsg, "regex"):
		return "regex_issue"
	case strings.Contains(errMsg, "wildcard"):
		return "wildcard_issue"
	default:
		return "general_issue"
	}
}

// parseTimeParameters parses time parameters with comprehensive format support and millisecond precision
func (h *LogHandler) parseTimeParameters(req *restful.Request) (*time.Time, *time.Time, error) {
	parseStartTime := time.Now()

	// Use time validator from service layer
	timeValidator := query.NewTimeRangeValidator()

	// Extract time parameter strings
	startTimeStr := req.QueryParameter("start_time")
	endTimeStr := req.QueryParameter("end_time")

	// Determine format type and precision for metrics
	formatType, precision := h.analyzeTimeFormat(startTimeStr, endTimeStr)

	// Validate and parse time range using enhanced validator
	startTime, endTime, err := timeValidator.ValidateAndParseTimeRange(startTimeStr, endTimeStr)

	parseDuration := time.Since(parseStartTime)
	h.timeMetrics.RecordTimeParsing(parseDuration, formatType, precision)

	if err != nil {
		// Convert service layer errors to API errors
		return nil, nil, h.convertTimeValidationError(err, startTimeStr, endTimeStr)
	}

	return startTime, endTime, nil
}

// analyzeTimeFormat analyzes time parameter formats for metrics
func (h *LogHandler) analyzeTimeFormat(startTimeStr, endTimeStr string) (string, string) {
	formatType := "none"
	precision := "second"

	// Analyze the first non-empty time string
	timeStr := startTimeStr
	if timeStr == "" {
		timeStr = endTimeStr
	}

	if timeStr == "" {
		return formatType, precision
	}

	// Determine format type
	switch {
	case strings.Contains(timeStr, "T") && strings.Contains(timeStr, "Z"):
		formatType = "rfc3339_utc"
	case strings.Contains(timeStr, "T") && (strings.Contains(timeStr, "+") || strings.Contains(timeStr, "-")):
		formatType = "rfc3339_tz"
	case strings.Contains(timeStr, " "):
		formatType = "sql_format"
	default:
		formatType = "iso8601"
	}

	// Determine precision
	if strings.Contains(timeStr, ".") {
		dotIndex := -1
		for i, char := range timeStr {
			if char == '.' {
				dotIndex = i
				break
			}
		}
		if dotIndex != -1 {
			// Count digits after decimal point before Z or timezone
			fractionalPart := timeStr[dotIndex+1:]
			digitCount := 0
			for _, char := range fractionalPart {
				if char >= '0' && char <= '9' {
					digitCount++
				} else {
					break
				}
			}

			switch digitCount {
			case 1, 2, 3:
				precision = "millisecond"
			case 4, 5, 6:
				precision = "microsecond"
			case 7, 8, 9:
				precision = "nanosecond"
			default:
				precision = "fractional"
			}
		}
	}

	return formatType, precision
}

// convertTimeValidationError converts service layer time errors to API-appropriate errors
func (h *LogHandler) convertTimeValidationError(err error, startTimeStr, endTimeStr string) error {
	errMsg := err.Error()

	// Handle different types of time validation errors
	switch {
	case strings.Contains(errMsg, "start_time") && strings.Contains(errMsg, "invalid"):
		return NewTimeParameterError("start_time", startTimeStr, "invalid time format", http.StatusBadRequest)

	case strings.Contains(errMsg, "end_time") && strings.Contains(errMsg, "invalid"):
		return NewTimeParameterError("end_time", endTimeStr, "invalid time format", http.StatusBadRequest)

	case strings.Contains(errMsg, "time format must be ISO 8601"):
		param := "start_time"
		value := startTimeStr
		if strings.Contains(errMsg, "end_time") {
			param = "end_time"
			value = endTimeStr
		}
		supportedFormats := []string{
			"RFC3339: 2006-01-02T15:04:05Z",
			"With milliseconds: 2006-01-02T15:04:05.123Z",
			"With microseconds: 2006-01-02T15:04:05.123456Z",
			"With nanoseconds: 2006-01-02T15:04:05.123456789Z",
		}
		return NewTimeFormatError(param, value, supportedFormats, "2024-01-01T10:30:45.123Z")

	case strings.Contains(errMsg, "time range error"):
		return NewTimeRangeAPIError(nil, nil, errMsg, "Ensure start_time <= end_time and time span <= 24 hours")

	case strings.Contains(errMsg, "cannot be in the future"):
		return NewTimeParameterError("time_validation", "", "future times not allowed", http.StatusBadRequest)

	default:
		return NewTimeParameterError("time_parsing", "", fmt.Sprintf("时间参数解析失败: %v", err), http.StatusBadRequest)
	}
}

// parseK8sParameters parses K8s filtering parameters with enhanced pattern support
func (h *LogHandler) parseK8sParameters(req *restful.Request) ([]string, []string, error) {
	var namespaces, pods []string

	// Parse single namespace parameter (backward compatibility)
	if ns := req.QueryParameter("namespace"); ns != "" {
		if strings.Contains(ns, ",") {
			namespaces = strings.Split(ns, ",")
		} else {
			namespaces = []string{ns}
		}
	}

	// Parse multiple namespaces parameter
	if nsArray := req.QueryParameter("namespaces"); nsArray != "" {
		additionalNs := strings.Split(nsArray, ",")
		namespaces = append(namespaces, additionalNs...)
	}

	// Parse single pod parameter (backward compatibility)
	if pod := req.QueryParameter("pod_name"); pod != "" {
		if strings.Contains(pod, ",") {
			pods = strings.Split(pod, ",")
		} else {
			pods = []string{pod}
		}
	}

	// Parse multiple pods parameter
	if podArray := req.QueryParameter("pods"); podArray != "" {
		additionalPods := strings.Split(podArray, ",")
		pods = append(pods, additionalPods...)
	}

	// Parse pod names parameter (alternative naming)
	if podNames := req.QueryParameter("pod_names"); podNames != "" {
		morePods := strings.Split(podNames, ",")
		pods = append(pods, morePods...)
	}

	// Clean up inputs
	namespaces = h.cleanStringArray(namespaces)
	pods = h.cleanStringArray(pods)

	return namespaces, pods, nil
}

// cleanStringArray removes empty entries and trims whitespace
func (h *LogHandler) cleanStringArray(input []string) []string {
	var result []string
	for _, item := range input {
		if cleaned := strings.TrimSpace(item); cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

// enrichResponseWithDataset enriches response with dataset metadata and query summary
func (h *LogHandler) enrichResponseWithDataset(result *response.LogQueryResponse, dataset string, queryReq *request.LogQueryRequest) {
	// Set dataset in response
	result.Dataset = dataset

	// Build query summary with enhanced time precision
	result.Query = &response.QuerySummary{
		StartTime: queryReq.StartTime,
		EndTime:   queryReq.EndTime,
		Filter:    queryReq.Filter,
		Namespace: queryReq.Namespace,
		Filters:   make(map[string]string),
	}

	// Add additional filters to summary
	if queryReq.PodName != "" {
		result.Query.Filters["pod_name"] = queryReq.PodName
	}
	if len(queryReq.PodNames) > 0 {
		result.Query.Filters["pod_names"] = fmt.Sprintf("[%s]", strings.Join(queryReq.PodNames, ", "))
	}
	if len(queryReq.Namespaces) > 0 {
		result.Query.Filters["namespaces"] = fmt.Sprintf("[%s]", strings.Join(queryReq.Namespaces, ", "))
	}
	if queryReq.NodeName != "" {
		result.Query.Filters["node_name"] = queryReq.NodeName
	}
	if queryReq.ContainerName != "" {
		result.Query.Filters["container_name"] = queryReq.ContainerName
	}
	if queryReq.Severity != "" {
		result.Query.Filters["severity"] = queryReq.Severity
	}

	// Add K8s filter metadata
	if len(queryReq.K8sFilters) > 0 {
		result.Query.Filters["k8s_filters_count"] = fmt.Sprintf("%d", len(queryReq.K8sFilters))

		namespaceFilterCount := 0
		podFilterCount := 0
		for _, filter := range queryReq.K8sFilters {
			if filter.Field == "namespace" {
				namespaceFilterCount++
			} else if filter.Field == "pod" {
				podFilterCount++
			}
		}

		if namespaceFilterCount > 0 {
			result.Query.Filters["namespace_filters"] = fmt.Sprintf("%d", namespaceFilterCount)
		}
		if podFilterCount > 0 {
			result.Query.Filters["pod_filters"] = fmt.Sprintf("%d", podFilterCount)
		}
	}

	// Add time range metadata for debugging
	if queryReq.StartTime != nil && queryReq.EndTime != nil {
		timeSpan := queryReq.EndTime.Sub(*queryReq.StartTime)
		result.Query.Filters["time_span_seconds"] = fmt.Sprintf("%.3f", timeSpan.Seconds())
		result.Query.Filters["time_precision"] = "nanosecond"
	}

	klog.V(4).InfoS("响应增强完成",
		"dataset", dataset,
		"has_metadata", result.Metadata != nil,
		"filter_count", len(result.Query.Filters),
		"time_precision_enabled", queryReq.StartTime != nil || queryReq.EndTime != nil)
}
