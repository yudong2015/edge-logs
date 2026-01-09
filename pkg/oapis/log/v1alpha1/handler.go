package v1alpha1

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/request"
	response "github.com/outpostos/edge-logs/pkg/response"
	query "github.com/outpostos/edge-logs/pkg/service/query"
)

// LogHandler handles log API requests with K8s API Aggregation pattern
type LogHandler struct {
	queryService *query.Service
}

// NewLogHandler creates a new log handler with service dependency
func NewLogHandler(queryService *query.Service) *LogHandler {
	klog.InfoS("初始化日志 API 处理器")
	return &LogHandler{
		queryService: queryService,
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
		Param(ws.QueryParameter("namespace", "Kubernetes 命名空间").DataType("string")).
		Param(ws.QueryParameter("pod_name", "Pod 名称").DataType("string")).
		Param(ws.QueryParameter("node_name", "节点名称").DataType("string")).
		Param(ws.QueryParameter("container_name", "容器名称").DataType("string")).
		Param(ws.QueryParameter("filter", "日志内容过滤").DataType("string")).
		Param(ws.QueryParameter("severity", "日志级别").DataType("string")).
		Param(ws.QueryParameter("page", "页码 (从0开始)").DataType("integer")).
		Param(ws.QueryParameter("page_size", "每页大小").DataType("integer")).
		Param(ws.QueryParameter("order_by", "排序字段 (timestamp, severity)").DataType("string")).
		Param(ws.QueryParameter("direction", "排序方向 (asc, desc)").DataType("string")).
		Returns(http.StatusOK, "查询成功", response.LogQueryResponse{}).
		Returns(http.StatusBadRequest, "请求参数错误", response.ErrorResponse{}).
		Returns(http.StatusNotFound, "数据集不存在", response.ErrorResponse{}).
		Returns(http.StatusInternalServerError, "服务器内部错误", response.ErrorResponse{}))

	// Health check endpoint
	ws.Route(ws.GET("/health").To(h.healthCheck).
		Doc("健康检查").
		Returns(http.StatusOK, "服务正常", response.HealthResponse{}))

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

	// Execute query through service layer
	result, err := h.queryService.QueryLogs(req.Request.Context(), queryReq)
	if err != nil {
		duration := time.Since(startTime)
		klog.ErrorS(err, "日志查询执行失败",
			"dataset", dataset,
			"duration_ms", duration.Milliseconds())
		h.writeErrorResponse(resp, h.mapErrorToStatusCode(err), fmt.Sprintf("查询失败: %v", err))
		return
	}

	duration := time.Since(startTime)
	klog.InfoS("日志查询请求处理成功",
		"dataset", dataset,
		"returned_logs", len(result.Logs),
		"total_count", result.TotalCount,
		"duration_ms", duration.Milliseconds())

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

	healthResponse := &response.HealthResponse{
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

	// Parse time parameters (ISO 8601 format)
	if startTimeStr := req.QueryParameter("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return nil, fmt.Errorf("start_time 格式错误: %w", err)
		}
		queryReq.StartTime = &startTime
	}

	if endTimeStr := req.QueryParameter("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, fmt.Errorf("end_time 格式错误: %w", err)
		}
		queryReq.EndTime = &endTime
	}

	// Parse filter parameters
	queryReq.Filter = req.QueryParameter("filter")
	queryReq.Severity = req.QueryParameter("severity")
	queryReq.Namespace = req.QueryParameter("namespace")
	queryReq.PodName = req.QueryParameter("pod_name")
	queryReq.NodeName = req.QueryParameter("node_name")
	queryReq.ContainerName = req.QueryParameter("container_name")

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
	errorResponse := &response.ErrorResponse{
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
	case contains(errMsg, "dataset is required"),
		contains(errMsg, "参数"),
		contains(errMsg, "格式错误"),
		contains(errMsg, "参数错误"):
		return http.StatusBadRequest

	case contains(errMsg, "not found"),
		contains(errMsg, "不存在"):
		return http.StatusNotFound

	case contains(errMsg, "connection"),
		contains(errMsg, "timeout"),
		contains(errMsg, "connection refused"):
		return http.StatusServiceUnavailable

	default:
		return http.StatusInternalServerError
	}
}

// contains checks if string contains substring (case sensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
