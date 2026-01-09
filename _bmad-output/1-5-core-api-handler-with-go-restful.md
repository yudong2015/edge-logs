# Story 1.5: core-api-handler-with-go-restful

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want to access logs through a REST API,
so that I can query edge computing logs using standard HTTP requests with proper response formatting.

## Acceptance Criteria

**Given** Log query service is implemented
**When** I implement the REST API handler
**Then** I can access logs via GET /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/logs
**And** API accepts query parameters for start_time, end_time, namespace, pod_name, filter, and limit
**And** Responses follow the defined JSON structure with proper HTTP status codes
**And** Request logging and metrics are implemented
**And** API documentation is generated with go-restful OpenAPI

## Tasks / Subtasks

- [ ] Implement K8s API Aggregation pattern with go-restful (AC: 1)
  - [ ] Configure go-restful WebService for log.theriseunion.io/v1alpha1 API group
  - [ ] Implement dataset-based URL routing: /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/logs
  - [ ] Register WebService with existing restful.Container in apiserver
  - [ ] Add proper API path and content type configuration (JSON)
  - [ ] Integrate with existing API server infrastructure
- [ ] Implement comprehensive HTTP query parameter handling (AC: 2)
  - [ ] Parse all query parameters: start_time, end_time, namespace, pod_name, filter, limit
  - [ ] Convert HTTP query strings to proper Go data types (time.Time, int, string)
  - [ ] Implement parameter validation and error responses
  - [ ] Handle required vs optional parameters with appropriate defaults
  - [ ] Support ISO 8601 timestamp parsing for time parameters
  - [ ] Add dataset path parameter extraction and validation
- [ ] Integrate with log query service layer (AC: 1)
  - [ ] Create service request from HTTP parameters
  - [ ] Call Story 1-4 query service with properly formatted requests
  - [ ] Handle service layer errors and map to HTTP status codes
  - [ ] Transform service responses to HTTP JSON responses
  - [ ] Implement proper error response formatting
- [ ] Implement comprehensive request/response handling (AC: 3)
  - [ ] Define HTTP response structures following architecture specifications
  - [ ] Implement proper JSON serialization with correct field names
  - [ ] Handle HTTP status codes (200, 400, 404, 500, etc.)
  - [ ] Add response headers for content type and cache control
  - [ ] Implement pagination response formatting
  - [ ] Handle empty result sets appropriately
- [ ] Add request logging and metrics collection (AC: 4)
  - [ ] Integrate with existing klog/v2 structured logging
  - [ ] Log all incoming requests with method, path, parameters
  - [ ] Log response status codes and processing time
  - [ ] Add Prometheus metrics for API call tracking
  - [ ] Track request counts by dataset and response status
  - [ ] Monitor query performance metrics
- [ ] Implement go-restful OpenAPI documentation (AC: 5)
  - [ ] Add comprehensive API endpoint documentation
  - [ ] Document all parameters with types and descriptions
  - [ ] Document response structures and status codes
  - [ ] Enable OpenAPI/Swagger generation
  - [ ] Add example requests and responses
- [ ] Add comprehensive API testing
  - [ ] Create unit tests for handler methods
  - [ ] Mock query service dependencies for isolated testing
  - [ ] Add integration tests for HTTP endpoints
  - [ ] Test parameter validation and error handling
  - [ ] Test various query combinations and edge cases
  - [ ] Add performance/load testing for API endpoints

## Dev Notes

### Architecture Compliance Requirements

**Critical:** This API handler layer completes the foundation stack by exposing the completed log query service (Story 1-4) through a K8s API Aggregation pattern using go-restful/v3. It implements the REST API tier that operators will consume directly.

**Key Technical Requirements:**
- **Framework Alignment:** Must use go-restful/v3 consistent with edge-apiserver
- **API Pattern:** K8s API Aggregation with proper group/version: log.theriseunion.io/v1alpha1
- **Service Integration:** Direct integration with pkg/service/query/ from Story 1-4
- **Error Handling:** Comprehensive HTTP status code mapping and error responses
- **Documentation:** OpenAPI/Swagger documentation generation via go-restful
- **Monitoring:** Request logging with klog/v2 and Prometheus metrics collection
- **Performance:** Maintain sub-2 second query response times (NFR1)

### K8s API Aggregation Pattern

**Based on architecture.md requirements, implementing proper K8s API aggregation:**

```go
// InstallHandler registers the log API with K8s API Aggregation pattern
func (h *LogHandler) InstallHandler(container *restful.Container) {
    // Create new WebService for log.theriseunion.io/v1alpha1
    ws := new(restful.WebService)
    ws.Path("/apis/log.theriseunion.io/v1alpha1").
        Consumes(restful.MIME_JSON).
        Produces(restful.MIME_JSON)

    // Main log query endpoint with dataset-based routing
    ws.Route(ws.GET("/logdatasets/{dataset}/logs").To(h.queryLogs).
        Doc("Query logs from a specific dataset").
        Param(ws.PathParameter("dataset", "Dataset identifier (e.g., edge-prod-traffic)")).
        Param(ws.QueryParameter("start_time", "Start timestamp (required, ISO 8601 format)").DataType("dateTime")).
        Param(ws.QueryParameter("end_time", "End timestamp (required, ISO 8601 format)").DataType("dateTime")).
        Param(ws.QueryParameter("namespace", "Filter by Kubernetes namespace").DataType("string")).
        Param(ws.QueryParameter("pod_name", "Filter by pod name").DataType("string")).
        Param(ws.QueryParameter("filter", "Log content filter/keyword search").DataType("string")).
        Param(ws.QueryParameter("limit", "Maximum records to return").DataType("integer")).
        Returns(200, "OK", response.LogQueryResponse{}).
        Returns(400, "Bad Request", ErrorResponse{}).
        Returns(404, "Dataset Not Found", ErrorResponse{}).
        Returns(500, "Internal Server Error", ErrorResponse{}))

    container.Add(ws)
}
```

### HTTP to Service Layer Integration

**Critical integration between HTTP layer and Story 1-4 service layer:**

```go
// queryLogs handles the main log query API endpoint
func (h *LogHandler) queryLogs(req *restful.Request, resp *restful.Response) {
    startTime := time.Now()
    dataset := req.PathParameter("dataset")

    // 1. Extract and validate query parameters from HTTP request
    queryReq, err := h.parseQueryRequest(req, dataset)
    if err != nil {
        klog.ErrorS(err, "Invalid query parameters", "dataset", dataset)
        h.writeErrorResponse(resp, http.StatusBadRequest, err.Error())
        h.metrics.RecordError(dataset, "invalid_parameters")
        return
    }

    // 2. Call Story 1-4 service layer
    klog.InfoS("Processing log query request",
        "dataset", dataset,
        "start_time", queryReq.StartTime,
        "end_time", queryReq.EndTime,
        "namespace", queryReq.Namespace)

    serviceResp, err := h.queryService.QueryLogs(req.Request.Context(), queryReq)
    if err != nil {
        klog.ErrorS(err, "Service query failed", "dataset", dataset)
        h.handleServiceError(resp, err, dataset)
        return
    }

    // 3. Transform service response to HTTP response
    klog.InfoS("Query completed successfully",
        "dataset", dataset,
        "result_count", len(serviceResp.Items),
        "duration_ms", time.Since(startTime).Milliseconds())

    h.writeSuccessResponse(resp, serviceResp)
    h.metrics.RecordSuccess(dataset, len(serviceResp.Items), time.Since(startTime))
}
```

### Query Parameter Parsing and Validation

**Comprehensive HTTP parameter parsing aligned with service layer expectations:**

```go
// parseQueryRequest converts HTTP parameters to service request
func (h *LogHandler) parseQueryRequest(req *restful.Request, dataset string) (*request.LogQueryRequest, error) {
    queryReq := &request.LogQueryRequest{
        Dataset: dataset,
    }

    // Parse required time parameters
    startTimeStr := req.QueryParameter("start_time")
    endTimeStr := req.QueryParameter("end_time")

    if startTimeStr == "" || endTimeStr == "" {
        return nil, errors.New("start_time and end_time are required parameters")
    }

    startTime, err := time.Parse(time.RFC3339Nano, startTimeStr)
    if err != nil {
        return nil, fmt.Errorf("invalid start_time format: %w (expected ISO 8601)", err)
    }
    queryReq.StartTime = &startTime

    endTime, err := time.Parse(time.RFC3339Nano, endTimeStr)
    if err != nil {
        return nil, fmt.Errorf("invalid end_time format: %w (expected ISO 8601)", err)
    }
    queryReq.EndTime = &endTime

    // Parse optional filtering parameters
    queryReq.Namespace = req.QueryParameter("namespace")
    queryReq.PodName = req.QueryParameter("pod_name")
    queryReq.Filter = req.QueryParameter("filter")

    // Parse pagination
    if limitStr := req.QueryParameter("limit"); limitStr != "" {
        limit, err := strconv.Atoi(limitStr)
        if err != nil {
            return nil, fmt.Errorf("invalid limit parameter: %w", err)
        }
        queryReq.PageSize = limit
    } else {
        queryReq.PageSize = 100 // default
    }

    // Validate the complete request
    if err := queryReq.Validate(); err != nil {
        return nil, fmt.Errorf("request validation failed: %w", err)
    }

    return queryReq, nil
}
```

### HTTP Response Formatting

**Proper HTTP response handling with correct status codes and formatting:**

```go
// writeSuccessResponse writes successful log query response
func (h *LogHandler) writeSuccessResponse(resp *restful.Response, serviceResp *response.LogQueryResponse) {
    // Transform service response to API response format
    apiResp := LogQueryAPIResponse{
        Logs:       transformLogEntries(serviceResp.Items),
        TotalCount: serviceResp.Total,
        Page:       serviceResp.Page,
        PageSize:   serviceResp.Limit,
        HasMore:    serviceResp.HasMore,
    }

    resp.WriteHeader(http.StatusOK)
    resp.WriteEntity(apiResp)
}

// writeErrorResponse writes error responses with proper format
func (h *LogHandler) writeErrorResponse(resp *restful.Response, statusCode int, message string) {
    errorResp := ErrorResponse{
        Error: ErrorDetails{
            Code:    statusCode,
            Message: message,
            Type:    getErrorType(statusCode),
        },
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }

    resp.WriteHeader(statusCode)
    resp.WriteEntity(errorResp)
}

// handleServiceError maps service errors to HTTP status codes
func (h *LogHandler) handleServiceError(resp *restful.Response, err error, dataset string) {
    switch {
    case isValidationError(err):
        h.writeErrorResponse(resp, http.StatusBadRequest, err.Error())
        h.metrics.RecordError(dataset, "validation_error")
    case isNotFoundError(err):
        h.writeErrorResponse(resp, http.StatusNotFound, "Dataset not found or no logs matching criteria")
        h.metrics.RecordError(dataset, "not_found")
    case isTimeoutError(err):
        h.writeErrorResponse(resp, http.StatusGatewayTimeout, "Query timeout")
        h.metrics.RecordError(dataset, "timeout")
    default:
        h.writeErrorResponse(resp, http.StatusInternalServerError, "Internal server error")
        h.metrics.RecordError(dataset, "internal_error")
    }
}
```

### Request Logging and Metrics

**Comprehensive monitoring aligned with architecture requirements:**

```go
// APIMetrics tracks API performance and usage
type APIMetrics struct {
    requestCounter   prometheus.CounterVec
    requestDuration  prometheus.HistogramVec
    errorCounter     prometheus.CounterVec
    resultSizeGauge  prometheus.HistogramVec
}

func (m *APIMetrics) RecordSuccess(dataset string, resultCount int, duration time.Duration) {
    m.requestCounter.With(prometheus.Labels{
        "dataset": dataset,
        "status":  "success",
    }).Inc()

    m.requestDuration.With(prometheus.Labels{
        "dataset": dataset,
        "status":  "success",
    }).Observe(duration.Seconds())

    m.resultSizeGauge.With(prometheus.Labels{
        "dataset": dataset,
    }).Observe(float64(resultCount))

    klog.InfoS("API request completed",
        "dataset", dataset,
        "result_count", resultCount,
        "duration_ms", duration.Milliseconds(),
        "status", "success")
}

func (m *APIMetrics) RecordError(dataset string, errorType string) {
    m.errorCounter.With(prometheus.Labels{
        "dataset":    dataset,
        "error_type": errorType,
    }).Inc()

    klog.ErrorS(nil, "API request error",
        "dataset", dataset,
        "error_type", errorType)
}
```

### OpenAPI Documentation Integration

**Enable automatic API documentation generation:**

```go
// Configure OpenAPI documentation
func configureOpenAPI(container *restful.Container) {
    config := restfulspec.Config{
        WebServices:                   container.RegisteredWebServices(),
        APIPath:                       "/apidocs.json",
        PostBuildSwaggerObjectHandler: enrichSwaggerObject}
    }

    container.Add(restfulspec.NewOpenAPIService(config))
}

func enrichSwaggerObject(swo *spec.Swagger) {
    swo.Info = &spec.Info{
        InfoProps: spec.InfoProps{
            Title:       "Edge Logs API",
            Description: "Kubernetes-native log aggregation API for edge computing",
            Version:     "v1alpha1",
            Contact: &spec.ContactInfo{
                Name: "Edge Logs Team",
            },
        },
    }
    swo.Tags = []spec.Tag{
        {TagProps: spec.TagProps{
            Name:        "logs",
            Description: "Operations for querying edge computing logs"}},
    }
}
```

### Integration with Existing API Server

**Seamless integration with pkg/apiserver/apiserver.go infrastructure:**

```go
// In cmd/apiserver/main.go, add handler registration
func main() {
    // ... existing setup code ...

    // Create API server
    server, err := apiserver.New(cfg)
    if err != nil {
        klog.Fatalf("Failed to create API server: %v", err)
    }

    // Create query service (from Story 1-4)
    queryService := query.NewService(cfg, repository)

    // Create and register log API handler
    logHandler := v1alpha1.NewLogHandler(queryService)
    logHandler.InstallHandler(server.Container())

    // Start server
    // ...
}
```

### Testing Strategy

**Comprehensive testing aligned with service layer patterns:**

1. **Unit Tests:**
   - Parameter parsing and validation
   - Error mapping and HTTP status codes
   - Response formatting
   - Metrics collection

2. **Integration Tests:**
   - HTTP endpoint testing with mock service
   - End-to-end request/response flows
   - Error condition handling
   - Performance benchmarking

3. **Contract Tests:**
   - API response format validation
   - OpenAPI documentation accuracy
   - Service layer integration contracts

### Security and Validation Requirements

**Critical security considerations for API layer:**

| Security Area | Implementation |
|---------------|----------------|
| **Dataset Authorization** | Validate dataset access permissions before service calls |
| **Input Validation** | Comprehensive parameter validation to prevent injection attacks |
| **Rate Limiting** | Integration with middleware for API throttling |
| **Audit Logging** | Log all API access with user context and timestamps |
| **Error Messages** | Sanitized error responses to avoid information leakage |

### Project Structure Notes

**File organization following established patterns:**

```
pkg/oapis/log/v1alpha1/
├── handler.go              # Main API handler implementation (enhance existing)
├── parameters.go           # HTTP parameter parsing (new)
├── responses.go            # HTTP response formatting (new)
├── errors.go              # Error handling and mapping (new)
├── metrics.go             # API metrics and monitoring (new)
├── openapi.go             # OpenAPI documentation config (new)
├── handler_test.go        # Unit tests (enhance existing)
├── integration_test.go    # Integration tests (new)
└── benchmark_test.go      # Performance tests (new)
```

**Integration Points:**
- Must use existing pkg/model/request/log.go structures
- Must use existing pkg/model/response/log.go structures
- Must integrate with pkg/service/query/service.go from Story 1-4
- Must register with existing pkg/apiserver/apiserver.go container
- Must use pkg/config/config.go for API configuration

### Dependencies and Version Requirements

**API layer dependencies (aligned with architecture requirements):**

```go
// Core dependencies already established
require (
    github.com/emicklei/go-restful/v3 v3.11.0  // REST API framework
    github.com/go-openapi/spec v0.20.9          // OpenAPI spec
    k8s.io/klog/v2 v2.100.1                    // Logging
    github.com/prometheus/client_golang v1.17.0 // Metrics
    github.com/stretchr/testify v1.8.4          // Testing
)
```

### Performance Requirements

**API layer performance targets (maintaining NFR1 sub-2 second total response time):**

- **Parameter Parsing:** < 5ms for standard request parsing
- **Service Call Overhead:** < 10ms for service integration
- **Response Serialization:** < 20ms for JSON formatting
- **Total API Latency:** < 50ms to maintain sub-2 second end-to-end response time
- **Concurrent Requests:** Support 100+ concurrent API calls

### API Endpoint Examples

**Usage examples for the implemented API:**

```bash
# Basic log query
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T01:00:00Z&limit=100"

# Query with namespace filter
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-cn/logs?namespace=default&start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T01:00:00Z"

# Query with content filter
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?filter=error&start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T01:00:00Z"

# Query with multiple filters
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?namespace=kube-system&pod_name=coredns&filter=warning&limit=50&start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T01:00:00Z"
```

### References

- [Source: _bmad-output/epics.md#Story 1.5] - Complete user story and acceptance criteria
- [Source: _bmad-output/architecture.md#4.3 API 设计] - K8s API Aggregation pattern requirements
- [Source: _bmad-output/1-4-basic-log-query-service.md] - Service layer integration points
- [Source: pkg/oapis/log/v1alpha1/handler.go] - Existing handler structure to enhance
- [Source: pkg/apiserver/apiserver.go] - API server infrastructure to integrate with
- [Source: pkg/service/query/service.go] - Story 1-4 service layer to consume
- [Source: pkg/model/request/log.go] - Request model structures
- [Source: pkg/model/response/log.go] - Response model structures

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-20250514

### Debug Log References

Implementing K8s API Aggregation pattern using go-restful/v3 to expose the completed log query service (Story 1-4) through a standards-compliant REST API. This completes the foundation stack enabling operators to query edge computing logs via standard HTTP requests.

### Completion Notes List

Core API handler layer implements the REST API tier that completes the foundation stack by exposing the log query service through K8s API Aggregation pattern. Provides comprehensive parameter handling, error mapping, metrics collection, and OpenAPI documentation generation for production-ready API consumption.

### File List

Primary files to be enhanced/created:
- pkg/oapis/log/v1alpha1/handler.go (enhance existing)
- pkg/oapis/log/v1alpha1/parameters.go (new)
- pkg/oapis/log/v1alpha1/responses.go (new)
- pkg/oapis/log/v1alpha1/errors.go (new)
- pkg/oapis/log/v1alpha1/metrics.go (new)
- pkg/oapis/log/v1alpha1/openapi.go (new)
- pkg/oapis/log/v1alpha1/handler_test.go (enhance existing)
- pkg/oapis/log/v1alpha1/integration_test.go (new)
- cmd/apiserver/main.go (modify to register handler)