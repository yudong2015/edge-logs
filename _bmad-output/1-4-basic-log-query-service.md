# Story 1.4: basic-log-query-service

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a developer,
I want to implement a log query service layer,
so that I can provide business logic for log retrieval with proper data transformation and validation.

## Acceptance Criteria

**Given** ClickHouse repository layer is implemented
**When** I create the log query service
**Then** I can query logs by dataset with time range filtering
**And** I can apply basic content filtering and severity filtering
**And** Query results are properly formatted and paginated
**And** Input validation is performed on all query parameters
**And** Service layer properly handles and logs errors

## Tasks / Subtasks

- [ ] Enhance log query service business logic (AC: 1-3)
  - [ ] Implement comprehensive dataset-based query routing
  - [ ] Add time range validation and filtering logic
  - [ ] Implement content and severity filtering
  - [ ] Add proper pagination and result formatting
  - [ ] Integrate with existing ClickHouse repository layer
- [ ] Implement input validation and parameter sanitization (AC: 4)
  - [ ] Create comprehensive request validation logic
  - [ ] Add dataset name validation and security checks
  - [ ] Implement time range validation (start_time, end_time)
  - [ ] Add filter parameter sanitization for SQL injection protection
  - [ ] Validate pagination parameters (limit, offset)
- [ ] Implement comprehensive error handling (AC: 5)
  - [ ] Create service-specific error types and wrapping
  - [ ] Map repository errors to service-level errors
  - [ ] Add detailed error logging with klog/v2
  - [ ] Implement graceful degradation for partial failures
  - [ ] Add timeout and context cancellation handling
- [ ] Implement query result transformation and formatting (AC: 3)
  - [ ] Transform ClickHouse LogEntry to API response format
  - [ ] Implement proper JSON serialization
  - [ ] Add metadata enrichment for query results
  - [ ] Handle timezone conversion and formatting
  - [ ] Implement result caching for performance optimization
- [ ] Add comprehensive service testing
  - [ ] Create unit tests for all business logic methods
  - [ ] Mock repository dependencies for isolated testing
  - [ ] Add integration tests with repository layer
  - [ ] Implement performance benchmarking for service operations
  - [ ] Add edge case and error condition testing

## Dev Notes

### Architecture Compliance Requirements

**Critical:** This service layer builds directly on top of the ClickHouse repository layer completed in Story 1-3. It implements the business logic tier that provides data transformation, validation, and error handling before API consumption.

**Key Technical Requirements:**
- **Framework Alignment:** Must use existing service architecture patterns from pkg/service/
- **Repository Integration:** Direct integration with pkg/repository/clickhouse/ interfaces
- **Error Handling:** Comprehensive error wrapping and klog/v2 structured logging
- **Data Models:** Use existing request/response models from pkg/model/
- **Performance:** Support sub-2 second query response times (NFR1)

### Service Layer Architecture Pattern

**Based on existing pkg/service/query/service.go structure, enhanced for comprehensive functionality:**

```go
// Service provides log query business logic with comprehensive capabilities
type Service struct {
    repo          clickhouse.Repository
    config        *config.Config
    validator     *QueryValidator
    transformer   *ResultTransformer
    cache         QueryCache // Optional for performance
    metrics       QueryMetrics
}

// QueryLogs provides enhanced business logic for log retrieval
func (s *Service) QueryLogs(ctx context.Context, req *request.LogQueryRequest) (*response.LogQueryResponse, error) {
    // 1. Request validation and sanitization
    if err := s.validator.ValidateRequest(req); err != nil {
        klog.ErrorS(err, "Invalid query request", "dataset", req.Dataset)
        return nil, NewValidationError("invalid request parameters", err)
    }

    // 2. Business logic and filtering
    enhancedReq := s.enhanceRequest(req)

    // 3. Repository query with error handling
    logs, total, err := s.repo.QueryLogs(ctx, enhancedReq)
    if err != nil {
        return nil, s.handleRepositoryError(err, "QueryLogs", req)
    }

    // 4. Result transformation and formatting
    response := s.transformer.TransformToResponse(logs, total, req)

    // 5. Metrics and logging
    s.metrics.RecordQuery(req.Dataset, len(logs), time.Since(startTime))

    return response, nil
}
```

### Integration with Story 1-3 Repository Layer

**Critical Integration Points with ClickHouse Repository:**

| Service Layer Responsibility | Repository Integration |
|----------------------------|----------------------|
| **Request Validation** | Maps to repository query parameters |
| **Business Logic** | Delegates to repository.QueryLogs() |
| **Error Handling** | Wraps repository errors with service context |
| **Data Transformation** | Converts repository models to API responses |
| **Caching/Performance** | Optional layer above repository calls |

### Query Validation Pattern

**Comprehensive validation aligned with ClickHouse schema and security:**

```go
// QueryValidator handles all request validation logic
type QueryValidator struct {
    allowedDatasets map[string]bool
    maxQueryRange   time.Duration
    maxLimit        int
}

func (v *QueryValidator) ValidateRequest(req *request.LogQueryRequest) error {
    // Dataset validation (security-critical)
    if !v.isValidDataset(req.Dataset) {
        return errors.New("invalid or unauthorized dataset")
    }

    // Time range validation
    if req.StartTime.After(req.EndTime) {
        return errors.New("start_time must be before end_time")
    }

    if req.EndTime.Sub(req.StartTime) > v.maxQueryRange {
        return errors.New("query time range exceeds maximum allowed")
    }

    // Pagination validation
    if req.Limit <= 0 || req.Limit > v.maxLimit {
        return fmt.Errorf("limit must be between 1 and %d", v.maxLimit)
    }

    // Content filter sanitization
    if req.Filter != "" {
        if err := v.validateContentFilter(req.Filter); err != nil {
            return fmt.Errorf("invalid content filter: %w", err)
        }
    }

    return nil
}
```

### Error Handling and Logging Pattern

**Service-level error wrapping with comprehensive logging:**

```go
// ServiceError represents service-layer errors with context
type ServiceError struct {
    Type    string
    Message string
    Cause   error
    Context map[string]interface{}
}

func (s *Service) handleRepositoryError(err error, operation string, req *request.LogQueryRequest) error {
    // Log structured error information
    klog.ErrorS(err, "Repository operation failed",
        "operation", operation,
        "dataset", req.Dataset,
        "start_time", req.StartTime,
        "end_time", req.EndTime,
        "error_type", reflect.TypeOf(err).String())

    // Map repository errors to service errors
    switch {
    case isConnectionError(err):
        return &ServiceError{
            Type:    "connection_error",
            Message: "Unable to connect to log storage",
            Cause:   err,
            Context: map[string]interface{}{
                "dataset": req.Dataset,
                "operation": operation,
            },
        }
    case isTimeoutError(err):
        return &ServiceError{
            Type:    "timeout_error",
            Message: "Query timed out",
            Cause:   err,
            Context: map[string]interface{}{
                "dataset": req.Dataset,
                "query_range": req.EndTime.Sub(req.StartTime).String(),
            },
        }
    default:
        return &ServiceError{
            Type:    "repository_error",
            Message: "Log query failed",
            Cause:   err,
            Context: map[string]interface{}{"operation": operation},
        }
    }
}
```

### Result Transformation Pattern

**Transform repository models to API response format:**

```go
// ResultTransformer handles conversion from repository models to API responses
type ResultTransformer struct {
    timezone *time.Location
}

func (t *ResultTransformer) TransformToResponse(
    logs []clickhouse.LogEntry,
    total int64,
    req *request.LogQueryRequest,
) *response.LogQueryResponse {

    // Convert repository LogEntry to response LogEntry
    responseLogs := make([]response.LogEntry, 0, len(logs))
    for _, log := range logs {
        responseLogs = append(responseLogs, response.LogEntry{
            Timestamp:     t.formatTimestamp(log.Timestamp),
            Dataset:       log.Dataset,
            Content:       log.Content,
            Severity:      log.Severity,
            ContainerName: log.ContainerName,
            HostName:      log.HostName,
            Namespace:     log.K8sNamespace,
            PodName:       log.K8sPodName,
            NodeName:      log.K8sNodeName,
            Tags:          log.Tags,
        })
    }

    // Calculate pagination metadata
    totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))
    currentPage := (req.Offset / req.Limit) + 1

    return &response.LogQueryResponse{
        Items:      responseLogs,
        Total:      total,
        Page:       currentPage,
        Limit:      req.Limit,
        TotalPages: totalPages,
        HasMore:    (currentPage * req.Limit) < int(total),
    }
}
```

### Performance Optimization Pattern

**Query caching and performance monitoring:**

```go
// QueryCache provides optional performance optimization
type QueryCache interface {
    Get(ctx context.Context, key string) (*response.LogQueryResponse, error)
    Set(ctx context.Context, key string, response *response.LogQueryResponse, ttl time.Duration) error
}

// QueryMetrics tracks service performance
type QueryMetrics struct {
    queryCounter    prometheus.Counter
    queryDuration   prometheus.Histogram
    errorCounter    prometheus.Counter
}

func (m *QueryMetrics) RecordQuery(dataset string, resultCount int, duration time.Duration) {
    m.queryCounter.With(prometheus.Labels{"dataset": dataset}).Inc()
    m.queryDuration.With(prometheus.Labels{"dataset": dataset}).Observe(duration.Seconds())

    klog.InfoS("Query metrics recorded",
        "dataset", dataset,
        "result_count", resultCount,
        "duration_ms", duration.Milliseconds())
}
```

### Integration Testing Strategy

**Comprehensive testing aligned with repository layer:**

1. **Unit Tests:**
   - Request validation logic
   - Error handling and mapping
   - Result transformation
   - Business logic methods

2. **Integration Tests:**
   - Service + Repository integration
   - End-to-end query flows
   - Error propagation testing
   - Performance benchmarks

3. **Mock Testing:**
   - Repository interface mocking
   - Error condition simulation
   - Timeout and cancellation testing

### Security and Validation Requirements

**Critical security considerations for service layer:**

| Security Area | Implementation |
|---------------|----------------|
| **Dataset Authorization** | Validate dataset access permissions |
| **Input Sanitization** | Prevent SQL injection in content filters |
| **Query Limits** | Enforce maximum query ranges and result sizes |
| **Rate Limiting** | Integration with middleware for query throttling |
| **Audit Logging** | Log all query operations with user context |

### Project Structure Notes

**File organization following established patterns:**

```
pkg/service/query/
├── service.go              # Main service implementation (enhance existing)
├── validator.go            # Request validation logic (new)
├── transformer.go          # Result transformation (new)
├── errors.go              # Service-specific error types (new)
├── metrics.go             # Query metrics and monitoring (new)
├── cache.go               # Optional query caching (new)
├── service_test.go        # Unit tests (enhance existing)
├── integration_test.go    # Integration tests (new)
└── benchmark_test.go      # Performance benchmarks (new)
```

**Integration Points:**
- Must use existing pkg/model/request/log.go structures
- Must use existing pkg/model/response/log.go structures
- Must integrate with pkg/repository/clickhouse/repository.go interfaces
- Must use pkg/config/config.go for service configuration
- Must integrate with pkg/middleware/ for logging and metrics

### Dependencies and Version Requirements

**Service layer dependencies (building on repository layer):**

```go
// Additional dependencies for service layer
require (
    github.com/prometheus/client_golang v1.17.0  // Metrics
    github.com/patrickmn/go-cache v2.1.0+incompatible  // Optional caching
    github.com/stretchr/testify v1.8.4  // Testing
    k8s.io/klog/v2 v2.100.1  // Logging
)
```

### Performance Requirements

**Service layer performance targets (building on NFR1):**

- **Query Processing:** < 100ms overhead above repository layer
- **Validation:** < 10ms for standard request validation
- **Transformation:** < 50ms for result transformation
- **Error Handling:** < 5ms for error mapping and logging
- **Total Service Latency:** < 200ms to maintain sub-2 second total response time

### References

- [Source: _bmad-output/epics.md#Story 1.4] - Complete user story and acceptance criteria
- [Source: _bmad-output/architecture.md#API 设计] - Service layer architecture requirements
- [Source: _bmad-output/1-3-clickhouse-repository-layer.md] - Repository layer integration points
- [Source: pkg/service/query/service.go] - Existing service structure to enhance
- [Source: pkg/model/request/log.go] - Request model structures
- [Source: pkg/model/response/log.go] - Response model structures
- [Source: pkg/repository/clickhouse/repository.go] - Repository interface to integrate

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-20250514

### Debug Log References

Building comprehensive service layer on top of completed ClickHouse repository layer (Story 1-3). Service provides business logic, validation, error handling, and data transformation between repository and API layers.

### Completion Notes List

Service layer implements the business logic tier that bridges the repository layer (Story 1-3) with the upcoming API handler layer (Story 1-5). Provides comprehensive validation, error handling, result transformation, and performance optimization.

### File List

Primary files to be enhanced/created:
- pkg/service/query/service.go (enhance existing)
- pkg/service/query/validator.go (new)
- pkg/service/query/transformer.go (new)
- pkg/service/query/errors.go (new)
- pkg/service/query/metrics.go (new)
- pkg/service/query/service_test.go (enhance existing)
- pkg/service/query/integration_test.go (new)