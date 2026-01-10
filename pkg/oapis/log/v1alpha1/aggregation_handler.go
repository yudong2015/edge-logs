package v1alpha1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/request"
	aggResponse "github.com/outpostos/edge-logs/pkg/model/response"
	query "github.com/outpostos/edge-logs/pkg/service/query"
)

// AggregationMetrics tracks aggregation query metrics
type AggregationMetrics struct {
	// Metrics tracking for aggregation queries
}

// NewAggregationMetrics creates new aggregation metrics
func NewAggregationMetrics() *AggregationMetrics {
	klog.InfoS("聚合指标初始化完成")
	return &AggregationMetrics{}
}

// queryAggregation handles aggregation API requests
func (h *LogHandler) queryAggregation(req *restful.Request, resp *restful.Response) {
	startTime := time.Now()

	// Extract dataset from path parameter
	dataset := req.PathParameter("dataset")

	klog.InfoS("开始处理聚合查询请求",
		"dataset", dataset,
		"method", req.Request.Method,
		"url", req.Request.URL.String())

	// Build aggregation request from HTTP parameters
	aggReq, err := h.parseAggregationRequest(req, dataset)
	if err != nil {
		klog.ErrorS(err, "聚合请求解析失败",
			"dataset", dataset)
		h.writeErrorResponse(resp, http.StatusBadRequest, fmt.Sprintf("参数解析失败: %v", err))
		return
	}

	// Validate aggregation request
	validator := query.NewAggregationDimensionValidator()
	if err := validator.ValidateAggregationRequest(aggReq); err != nil {
		klog.ErrorS(err, "聚合请求验证失败", "dataset", dataset)
		h.writeErrorResponse(resp, http.StatusBadRequest, fmt.Sprintf("请求验证失败: %v", err))
		return
	}

	// Validate dataset exists
	if err := h.validateDataset(dataset); err != nil {
		klog.ErrorS(err, "数据集验证失败", "dataset", dataset)
		h.handleDatasetError(resp, err, dataset)
		return
	}

	// Execute aggregation query through repository
	var result *aggResponse.AggregationResponse
	result, err = h.queryService.QueryAggregation(req.Request.Context(), aggReq)
	if err != nil {
		duration := time.Since(startTime)
		klog.ErrorS(err, "聚合查询执行失败",
			"dataset", dataset,
			"duration_ms", duration.Milliseconds())
		h.writeErrorResponse(resp, http.StatusInternalServerError, fmt.Sprintf("聚合查询失败: %v", err))
		return
	}

	// Verify result type
	_ = (*aggResponse.AggregationResponse)(nil)

	duration := time.Since(startTime)
	klog.InfoS("聚合查询请求处理成功",
		"dataset", dataset,
		"result_count", len(result.Results),
		"duration_ms", duration.Milliseconds())

	// Write successful response
	resp.WriteHeaderAndEntity(http.StatusOK, result)
}

// parseAggregationRequest parses HTTP request parameters into AggregationRequest
func (h *LogHandler) parseAggregationRequest(req *restful.Request, dataset string) (*request.AggregationRequest, error) {
	aggReq := &request.AggregationRequest{
		Dataset: dataset,
	}

	// Parse time parameters
	startTimeStr := req.QueryParameter("start_time")
	endTimeStr := req.QueryParameter("end_time")

	if startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time format: %w", err)
		}
		aggReq.StartTime = &startTime
	}

	if endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time format: %w", err)
		}
		aggReq.EndTime = &endTime
	}

	// Parse filter parameters
	aggReq.Namespaces = parseQueryParamArray(req, "namespaces")
	aggReq.PodNames = parseQueryParamArray(req, "pod_names")
	aggReq.Severity = req.QueryParameter("severity")
	aggReq.ContentSearch = req.QueryParameter("content_search")

	// Parse dimensions
	dimensionsStr := req.QueryParameter("dimensions")
	if dimensionsStr == "" {
		return nil, fmt.Errorf("dimensions parameter is required")
	}
	dimensions, err := parseDimensions(dimensionsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid dimensions: %w", err)
	}
	aggReq.Dimensions = dimensions

	// Parse functions
	functionsStr := req.QueryParameter("functions")
	if functionsStr == "" {
		// Default to count if not specified
		aggReq.Functions = []request.AggregationFunction{
			{Type: request.FunctionCount},
		}
	} else {
		functions, err := parseFunctions(functionsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid functions: %w", err)
		}
		aggReq.Functions = functions
	}

	// Parse time bucket for timestamp dimension
	timeBucket := req.QueryParameter("time_bucket")
	if timeBucket != "" {
		// Apply time bucket to timestamp dimension
		for i := range aggReq.Dimensions {
			if aggReq.Dimensions[i].Type == request.DimensionTimestamp {
				aggReq.Dimensions[i].TimeBucket = request.TimeBucketInterval(timeBucket)
			}
		}
	}

	// Parse order by
	aggReq.OrderBy = parseQueryParamArray(req, "order_by")

	// Parse pagination
	if limitStr := req.QueryParameter("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		aggReq.Limit = limit
	}

	if offsetStr := req.QueryParameter("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, fmt.Errorf("invalid offset: %w", err)
		}
		aggReq.Offset = offset
	}

	return aggReq, nil
}

// parseDimensions parses dimensions parameter string
func parseDimensions(dimensionsStr string) ([]request.AggregationDimension, error) {
	var dimensions []request.AggregationDimension

	// Parse comma-separated dimensions
	dimTypes := splitAndTrim(dimensionsStr, ",")

	for _, dimType := range dimTypes {
		var dimension request.AggregationDimension

		switch dimType {
		case "severity":
			dimension = request.AggregationDimension{Type: request.DimensionSeverity}
		case "namespace":
			dimension = request.AggregationDimension{Type: request.DimensionNamespace}
		case "pod_name":
			dimension = request.AggregationDimension{Type: request.DimensionPodName}
		case "node_name":
			dimension = request.AggregationDimension{Type: request.DimensionNodeName}
		case "host_name":
			dimension = request.AggregationDimension{Type: request.DimensionHostName}
		case "container_name":
			dimension = request.AggregationDimension{Type: request.DimensionContainerName}
		case "timestamp":
			dimension = request.AggregationDimension{
				Type:       request.DimensionTimestamp,
				TimeBucket: request.IntervalHour, // Default to hourly
			}
		case "dataset":
			dimension = request.AggregationDimension{Type: request.DimensionDataset}
		default:
			return nil, fmt.Errorf("unsupported dimension type: %s", dimType)
		}

		dimensions = append(dimensions, dimension)
	}

	if len(dimensions) == 0 {
		return nil, fmt.Errorf("at least one dimension must be specified")
	}

	return dimensions, nil
}

// parseFunctions parses functions parameter string
func parseFunctions(functionsStr string) ([]request.AggregationFunction, error) {
	var functions []request.AggregationFunction

	// Parse comma-separated functions
	funcTypes := splitAndTrim(functionsStr, ",")

	for _, funcType := range funcTypes {
		var function request.AggregationFunction

		switch funcType {
		case "count":
			function = request.AggregationFunction{Type: request.FunctionCount}
		case "sum":
			function = request.AggregationFunction{Type: request.FunctionSum, Field: "*"}
		case "avg":
			function = request.AggregationFunction{Type: request.FunctionAvg, Field: "*"}
		case "min":
			function = request.AggregationFunction{Type: request.FunctionMin, Field: "*"}
		case "max":
			function = request.AggregationFunction{Type: request.FunctionMax, Field: "*"}
		case "distinct_count":
			function = request.AggregationFunction{Type: request.FunctionDistinctCount, Field: "severity"}
		default:
			return nil, fmt.Errorf("unsupported function type: %s", funcType)
		}

		functions = append(functions, function)
	}

	if len(functions) == 0 {
		return nil, fmt.Errorf("at least one function must be specified")
	}

	return functions, nil
}

// parseQueryParamArray parses query parameter as array
func parseQueryParamArray(req *restful.Request, param string) []string {
	val := req.QueryParameter(param)
	if val == "" {
		return nil
	}
	return splitAndTrim(val, ",")
}

// splitAndTrim splits string by separator and trims whitespace
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}

	parts := make([]string, 0)
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
