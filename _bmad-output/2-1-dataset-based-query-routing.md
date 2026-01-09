# Story 2.1: dataset-based-query-routing

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want to query logs from specific datasets through URL routing,
so that I can access logs from different edge clusters or environments with proper data isolation.

## Acceptance Criteria

**Given** Core API is implemented (Story 1-5 completed)
**When** I make a request with a dataset in the URL path
**Then** The API routes the query to the correct dataset in ClickHouse
**And** Dataset validation ensures only valid datasets are accessible
**And** Each query is properly scoped to the specified dataset
**And** Cross-dataset queries are prevented for security
**And** Dataset information is included in response metadata

## Tasks / Subtasks

- [ ] Enhance dataset validation in API handler (AC: 2)
  - [ ] Create dataset whitelist validation service
  - [ ] Implement dataset existence check against ClickHouse SYSTEM.PARTS
  - [ ] Add dataset authorization checks (prevent unauthorized access)
  - [ ] Return 404 error for non-existent datasets with proper error message
  - [ ] Add dataset validation to query parameter parsing layer
- [ ] Implement proper dataset scoping in query service (AC: 3)
  - [ ] Enhance pkg/service/query/service.go to enforce dataset parameter
  - [ ] Ensure all ClickHouse queries include `WHERE dataset = ?` clause
  - [ ] Validate dataset parameter is never empty or null in service layer
  - [ ] Add dataset validation in service request models
  - [ ] Implement dataset-specific query optimization
- [ ] Prevent cross-dataset security vulnerabilities (AC: 4)
  - [ ] Audit all ClickHouse queries to ensure dataset scoping
  - [ ] Add unit tests for SQL injection prevention via dataset parameter
  - [ ] Implement dataset parameter sanitization and validation
  - [ ] Add integration tests for cross-dataset access prevention
  - [ ] Document dataset security model and access patterns
- [ ] Include dataset metadata in API responses (AC: 5)
  - [ ] Add dataset field to LogQueryResponse structure
  - [ ] Include dataset statistics in response metadata (total logs, date range)
  - [ ] Add dataset information to error responses for debugging
  - [ ] Update API documentation to reflect dataset metadata inclusion
  - [ ] Ensure consistent dataset naming in responses
- [ ] Enhance query routing performance (AC: 1)
  - [ ] Optimize ClickHouse queries with dataset as first WHERE clause parameter
  - [ ] Implement dataset-based connection pooling if beneficial
  - [ ] Add dataset-specific metrics and monitoring
  - [ ] Benchmark dataset routing performance vs baseline
  - [ ] Document performance characteristics of dataset routing

## Dev Notes

### Architecture Compliance Requirements

**Critical:** This story enhances the existing API handler (Story 1-5) to provide proper dataset-based query routing and data isolation. Dataset serves as the primary isolation boundary following the architecture's "dataset作为独立字段，支持多数据源逻辑隔离" requirement.

**Key Technical Requirements:**
- **Dataset as Primary Key:** Dataset routing is the core isolation mechanism, implemented in URL path: `/logdatasets/{dataset}/logs`
- **ClickHouse Integration:** All queries must include `WHERE dataset = ?` as the first filter clause for performance
- **Security Model:** Prevent cross-dataset access through validation and query scoping
- **Performance:** Maintain sub-2 second response times with dataset-specific optimizations
- **API Consistency:** Build upon existing go-restful API structure from Story 1-5

### Dataset Routing Implementation

**Based on architecture.md specifications, implementing proper dataset routing:**

```go
// Enhanced dataset validation in API handler layer
func (h *LogHandler) validateDataset(dataset string) error {
    // 1. Sanitize dataset parameter
    if dataset == "" {
        return fmt.Errorf("dataset parameter is required")
    }

    // 2. Validate dataset format (alphanumeric, hyphens, underscores only)
    if !h.datasetRegex.MatchString(dataset) {
        return fmt.Errorf("invalid dataset format: %s", dataset)
    }

    // 3. Check if dataset exists in ClickHouse
    exists, err := h.queryService.DatasetExists(context.Background(), dataset)
    if err != nil {
        return fmt.Errorf("failed to validate dataset existence: %w", err)
    }
    if !exists {
        return NewDatasetNotFoundError(dataset)
    }

    // 4. Check dataset authorization (if implemented)
    if h.authService != nil {
        if !h.authService.IsDatasetAuthorized(dataset) {
            return NewDatasetUnauthorizedError(dataset)
        }
    }

    return nil
}

// Enhanced queryLogs handler with dataset routing
func (h *LogHandler) queryLogs(req *restful.Request, resp *restful.Response) {
    startTime := time.Now()
    dataset := req.PathParameter("dataset")

    // 1. Validate dataset first (before any processing)
    if err := h.validateDataset(dataset); err != nil {
        klog.ErrorS(err, "Dataset validation failed", "dataset", dataset)
        h.handleDatasetError(resp, err, dataset)
        return
    }

    // 2. Parse and validate query parameters
    queryReq, err := h.parseQueryRequest(req, dataset)
    if err != nil {
        klog.ErrorS(err, "Invalid query parameters", "dataset", dataset)
        h.writeErrorResponse(resp, http.StatusBadRequest, err.Error())
        h.metrics.RecordError(dataset, "invalid_parameters")
        return
    }

    // 3. Execute dataset-scoped query
    klog.InfoS("Processing dataset-scoped log query",
        "dataset", dataset,
        "start_time", queryReq.StartTime,
        "end_time", queryReq.EndTime,
        "filters", queryReq.GetFilters())

    serviceResp, err := h.queryService.QueryLogsByDataset(req.Request.Context(), queryReq)
    if err != nil {
        klog.ErrorS(err, "Dataset query failed", "dataset", dataset)
        h.handleServiceError(resp, err, dataset)
        return
    }

    // 4. Include dataset metadata in response
    h.enrichResponseWithDataset(serviceResp, dataset)
    h.writeSuccessResponse(resp, serviceResp)
    h.metrics.RecordDatasetSuccess(dataset, len(serviceResp.Items), time.Since(startTime))
}
```

### Service Layer Dataset Scoping

**Enhanced service layer to enforce dataset isolation:**

```go
// Enhanced QueryLogsByDataset with strict dataset scoping
func (s *LogQueryService) QueryLogsByDataset(ctx context.Context, req *request.LogQueryRequest) (*response.LogQueryResponse, error) {
    // 1. Validate dataset parameter is present
    if req.Dataset == "" {
        return nil, NewValidationError("dataset parameter is required")
    }

    // 2. Build dataset-scoped query with dataset as first WHERE clause
    query := s.buildDatasetQuery(req)

    // 3. Log the query for audit purposes
    klog.InfoS("Executing dataset-scoped query",
        "dataset", req.Dataset,
        "query_hash", s.hashQuery(query),
        "filters", req.GetFilterSummary())

    // 4. Execute query with dataset enforcement
    rows, err := s.repository.QueryWithDataset(ctx, query, req.Dataset)
    if err != nil {
        return nil, fmt.Errorf("dataset query execution failed: %w", err)
    }
    defer rows.Close()

    // 5. Parse results and include dataset metadata
    items, err := s.parseLogItems(rows, req.Dataset)
    if err != nil {
        return nil, fmt.Errorf("failed to parse query results: %w", err)
    }

    // 6. Build response with dataset information
    return &response.LogQueryResponse{
        Items:     items,
        Dataset:   req.Dataset,
        Total:     len(items),
        HasMore:   len(items) >= req.PageSize,
        Query:     req.GetSanitizedQuery(),
        Metadata:  s.buildDatasetMetadata(req.Dataset),
    }, nil
}

// buildDatasetQuery ensures dataset is always the first WHERE clause for performance
func (s *LogQueryService) buildDatasetQuery(req *request.LogQueryRequest) string {
    var whereConditions []string
    var args []interface{}

    // 1. Dataset must be first WHERE condition for ClickHouse optimization
    whereConditions = append(whereConditions, "dataset = ?")
    args = append(args, req.Dataset)

    // 2. Add time range filters
    if req.StartTime != nil {
        whereConditions = append(whereConditions, "timestamp >= ?")
        args = append(args, *req.StartTime)
    }
    if req.EndTime != nil {
        whereConditions = append(whereConditions, "timestamp <= ?")
        args = append(args, *req.EndTime)
    }

    // 3. Add optional filters
    if req.Namespace != "" {
        whereConditions = append(whereConditions, "k8s_namespace_name = ?")
        args = append(args, req.Namespace)
    }
    if req.PodName != "" {
        whereConditions = append(whereConditions, "k8s_pod_name LIKE ?")
        args = append(args, "%"+req.PodName+"%")
    }
    if req.Filter != "" {
        whereConditions = append(whereConditions, "positionCaseInsensitive(content, ?) > 0")
        args = append(args, req.Filter)
    }

    query := fmt.Sprintf(`
        SELECT timestamp, content, severity, k8s_namespace_name, k8s_pod_name, k8s_node_name, host_ip
        FROM logs
        WHERE %s
        ORDER BY timestamp DESC
        LIMIT %d
    `, strings.Join(whereConditions, " AND "), req.PageSize)

    return query
}
```

### Repository Layer Dataset Enforcement

**Enhanced repository layer with dataset validation:**

```go
// QueryWithDataset ensures all repository queries are dataset-scoped
func (r *ClickHouseRepository) QueryWithDataset(ctx context.Context, query string, dataset string) (driver.Rows, error) {
    // 1. Validate that query contains dataset filter
    if !strings.Contains(strings.ToLower(query), "dataset = ?") {
        return nil, fmt.Errorf("query must include dataset filter for security")
    }

    // 2. Validate dataset parameter
    if dataset == "" {
        return nil, fmt.Errorf("dataset parameter cannot be empty")
    }

    // 3. Check dataset existence in SYSTEM.PARTS
    exists, err := r.checkDatasetExists(ctx, dataset)
    if err != nil {
        return nil, fmt.Errorf("failed to verify dataset existence: %w", err)
    }
    if !exists {
        return nil, NewDatasetNotFoundError(dataset)
    }

    // 4. Execute query with timeout
    queryCtx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
    defer cancel()

    klog.V(2).InfoS("Executing dataset query",
        "dataset", dataset,
        "query", r.sanitizeQueryForLog(query))

    return r.conn.Query(queryCtx, query, dataset)
}

// checkDatasetExists validates dataset exists in ClickHouse
func (r *ClickHouseRepository) checkDatasetExists(ctx context.Context, dataset string) (bool, error) {
    query := `
        SELECT COUNT(*)
        FROM system.parts
        WHERE database = ? AND table = ?
        AND partition LIKE ?
        AND active = 1
    `

    var count uint64
    err := r.conn.QueryRow(ctx, query, r.config.Database, "logs", dataset+"_%").Scan(&count)
    if err != nil {
        return false, fmt.Errorf("failed to check dataset existence: %w", err)
    }

    return count > 0, nil
}
```

### Dataset Security and Validation

**Comprehensive dataset security model:**

```go
// Dataset validation patterns and security
type DatasetValidator struct {
    allowedPatterns []string
    blockedDatasets []string
    regex          *regexp.Regexp
}

func NewDatasetValidator() *DatasetValidator {
    return &DatasetValidator{
        // Allow alphanumeric, hyphens, underscores, max 64 chars
        regex: regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`),
        allowedPatterns: []string{
            "prod-*",
            "staging-*",
            "dev-*",
            "edge-*",
        },
        blockedDatasets: []string{
            "system",
            "internal",
            "admin",
        },
    }
}

func (v *DatasetValidator) ValidateDataset(dataset string) error {
    // 1. Format validation
    if !v.regex.MatchString(dataset) {
        return fmt.Errorf("dataset format invalid: must be alphanumeric with hyphens/underscores, max 64 chars")
    }

    // 2. Blocked dataset check
    for _, blocked := range v.blockedDatasets {
        if dataset == blocked {
            return fmt.Errorf("dataset '%s' is reserved and cannot be accessed", dataset)
        }
    }

    // 3. Pattern allowlist check (if configured)
    if len(v.allowedPatterns) > 0 {
        allowed := false
        for _, pattern := range v.allowedPatterns {
            if matched, _ := filepath.Match(pattern, dataset); matched {
                allowed = true
                break
            }
        }
        if !allowed {
            return fmt.Errorf("dataset '%s' does not match allowed patterns", dataset)
        }
    }

    return nil
}
```

### Response Enhancement with Dataset Metadata

**Enhanced response structures with dataset information:**

```go
// Enhanced LogQueryResponse with dataset metadata
type LogQueryResponse struct {
    Items     []LogEntry     `json:"items"`
    Dataset   string         `json:"dataset"`           // Dataset name from request
    Total     int            `json:"total"`
    HasMore   bool           `json:"has_more"`
    Query     QuerySummary   `json:"query"`
    Metadata  DatasetMetadata `json:"metadata"`
}

type DatasetMetadata struct {
    Name            string    `json:"name"`
    TotalLogs       int64     `json:"total_logs"`
    DateRange       DateRange `json:"date_range"`
    LastUpdated     time.Time `json:"last_updated"`
    PartitionCount  int       `json:"partition_count"`
    DataSizeBytes   int64     `json:"data_size_bytes"`
}

// buildDatasetMetadata enriches response with dataset information
func (s *LogQueryService) buildDatasetMetadata(dataset string) DatasetMetadata {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Query dataset statistics from ClickHouse system tables
    stats, err := s.repository.GetDatasetStats(ctx, dataset)
    if err != nil {
        klog.ErrorS(err, "Failed to get dataset metadata", "dataset", dataset)
        return DatasetMetadata{Name: dataset}
    }

    return DatasetMetadata{
        Name:            dataset,
        TotalLogs:       stats.TotalRows,
        DateRange:       stats.DateRange,
        LastUpdated:     stats.LastModified,
        PartitionCount:  stats.PartitionCount,
        DataSizeBytes:   stats.DataSizeBytes,
    }
}
```

### Error Handling for Dataset Operations

**Comprehensive error handling with dataset context:**

```go
// Dataset-specific error types
type DatasetNotFoundError struct {
    Dataset string
}

func (e *DatasetNotFoundError) Error() string {
    return fmt.Sprintf("dataset '%s' not found or contains no data", e.Dataset)
}

type DatasetUnauthorizedError struct {
    Dataset string
}

func (e *DatasetUnauthorizedError) Error() string {
    return fmt.Sprintf("access to dataset '%s' is not authorized", e.Dataset)
}

// Enhanced error handling in API layer
func (h *LogHandler) handleDatasetError(resp *restful.Response, err error, dataset string) {
    switch e := err.(type) {
    case *DatasetNotFoundError:
        h.writeErrorResponse(resp, http.StatusNotFound,
            fmt.Sprintf("Dataset '%s' not found. Available datasets can be listed via the datasets endpoint.", dataset))
        h.metrics.RecordDatasetError(dataset, "not_found")
    case *DatasetUnauthorizedError:
        h.writeErrorResponse(resp, http.StatusForbidden,
            "Access to the requested dataset is not authorized")
        h.metrics.RecordDatasetError(dataset, "unauthorized")
    default:
        h.writeErrorResponse(resp, http.StatusBadRequest, err.Error())
        h.metrics.RecordDatasetError(dataset, "validation_error")
    }
}
```

### Dataset-Specific Metrics and Monitoring

**Enhanced metrics collection with dataset dimensions:**

```go
// Enhanced metrics with dataset tracking
type DatasetMetrics struct {
    requestCounter   *prometheus.CounterVec
    requestDuration  *prometheus.HistogramVec
    errorCounter     *prometheus.CounterVec
    datasetSize      *prometheus.GaugeVec
    lastQuery        *prometheus.GaugeVec
}

func NewDatasetMetrics() *DatasetMetrics {
    return &DatasetMetrics{
        requestCounter: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_dataset_requests_total",
                Help: "Total requests per dataset",
            },
            []string{"dataset", "status"},
        ),
        requestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_dataset_request_duration_seconds",
                Help:    "Request duration per dataset",
                Buckets: prometheus.DefBuckets,
            },
            []string{"dataset", "status"},
        ),
        errorCounter: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_dataset_errors_total",
                Help: "Errors per dataset",
            },
            []string{"dataset", "error_type"},
        ),
        datasetSize: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "edge_logs_dataset_size_bytes",
                Help: "Dataset size in bytes",
            },
            []string{"dataset"},
        ),
        lastQuery: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "edge_logs_dataset_last_query_timestamp",
                Help: "Last query timestamp per dataset",
            },
            []string{"dataset"},
        ),
    }
}

func (m *DatasetMetrics) RecordDatasetSuccess(dataset string, resultCount int, duration time.Duration) {
    m.requestCounter.With(prometheus.Labels{
        "dataset": dataset,
        "status":  "success",
    }).Inc()

    m.requestDuration.With(prometheus.Labels{
        "dataset": dataset,
        "status":  "success",
    }).Observe(duration.Seconds())

    m.lastQuery.With(prometheus.Labels{
        "dataset": dataset,
    }).Set(float64(time.Now().Unix()))

    klog.InfoS("Dataset query completed",
        "dataset", dataset,
        "result_count", resultCount,
        "duration_ms", duration.Milliseconds())
}
```

### Performance Optimization for Dataset Routing

**Dataset-specific performance optimizations:**

```go
// Dataset connection pool optimization
type DatasetConnectionPool struct {
    pools map[string]*clickhouse.ConnPool
    mutex sync.RWMutex
    config *config.ClickHouseConfig
}

func (p *DatasetConnectionPool) GetConnection(dataset string) (*clickhouse.Conn, error) {
    p.mutex.RLock()
    pool, exists := p.pools[dataset]
    p.mutex.RUnlock()

    if !exists {
        p.mutex.Lock()
        defer p.mutex.Unlock()

        // Double-check pattern
        if pool, exists = p.pools[dataset]; !exists {
            // Create dataset-specific connection pool with optimized settings
            pool = clickhouse.NewConnPool(&clickhouse.Options{
                Addr:        p.config.Address,
                Database:    p.config.Database,
                MaxConns:    p.config.MaxConns,
                MaxIdleTime: p.config.MaxIdleTime,
                // Dataset-specific optimization settings
                Settings: map[string]interface{}{
                    "max_partitions_per_insert_block": 1000,
                    "optimize_on_insert":               1,
                    "dataset_filter_optimization":     1,
                },
            })
            p.pools[dataset] = pool
        }
    }

    return pool.Get(), nil
}
```

### Testing Strategy

**Comprehensive testing for dataset routing:**

1. **Unit Tests:**
   - Dataset validation logic
   - Query building with dataset scoping
   - Error handling for dataset operations
   - Security validation

2. **Integration Tests:**
   - End-to-end dataset routing
   - Cross-dataset access prevention
   - Performance benchmarking per dataset
   - Error condition handling

3. **Security Tests:**
   - SQL injection prevention via dataset parameter
   - Unauthorized dataset access attempts
   - Dataset parameter sanitization
   - Cross-dataset query blocking

### Project Structure Notes

**File organization building on Story 1-5:**

```
pkg/service/query/
├── service.go              # Enhanced with dataset scoping (modify existing)
├── dataset_validator.go    # Dataset validation logic (new)
└── service_test.go         # Enhanced with dataset tests (modify existing)

pkg/repository/clickhouse/
├── repository.go           # Enhanced with dataset methods (modify existing)
├── dataset_queries.go      # Dataset-specific queries (new)
└── repository_test.go      # Enhanced with dataset tests (modify existing)

pkg/oapis/log/v1alpha1/
├── handler.go              # Enhanced with dataset validation (modify existing)
├── dataset_errors.go       # Dataset error types (new)
├── dataset_metrics.go      # Dataset metrics (new)
└── handler_test.go         # Enhanced with dataset tests (modify existing)

pkg/model/response/
└── log.go                 # Enhanced with dataset metadata (modify existing)
```

**Key Integration Points:**
- Enhances existing pkg/oapis/log/v1alpha1/handler.go from Story 1-5
- Builds upon pkg/service/query/service.go from Story 1-4
- Extends pkg/repository/clickhouse/repository.go from Story 1-3
- Uses existing ClickHouse schema from Story 1-2 with dataset column

### Dependencies and Version Requirements

**No new dependencies required - builds on existing stack:**

```go
// Existing dependencies from previous stories
require (
    github.com/ClickHouse/clickhouse-go/v2 v2.15.0  // Repository layer
    github.com/emicklei/go-restful/v3 v3.11.0        // API framework
    k8s.io/klog/v2 v2.100.1                          // Logging
    github.com/prometheus/client_golang v1.17.0      // Metrics
    github.com/stretchr/testify v1.8.4                // Testing
)
```

### API Usage Examples

**Enhanced API usage with proper dataset routing:**

```bash
# Basic dataset-scoped query
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T01:00:00Z"

# Query specific edge environment
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-cn-hz01/logs?namespace=default&start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T01:00:00Z"

# Query staging environment with filters
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/staging-app/logs?filter=error&namespace=kube-system&start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T01:00:00Z"

# Response includes dataset metadata
{
  "items": [...],
  "dataset": "prod-cluster",
  "total": 1524,
  "has_more": true,
  "metadata": {
    "name": "prod-cluster",
    "total_logs": 15240000,
    "date_range": {
      "earliest": "2024-01-01T00:00:00Z",
      "latest": "2024-01-09T12:00:00Z"
    },
    "partition_count": 45,
    "data_size_bytes": 2847392736
  }
}
```

### Performance Requirements

**Dataset routing performance targets:**

- **Dataset Validation:** < 5ms per request for existence check
- **Query Scoping:** No additional latency vs unscoped queries
- **Response Enhancement:** < 10ms for metadata inclusion
- **Total Routing Overhead:** < 20ms additional latency for dataset features
- **Concurrent Dataset Queries:** Support 50+ different datasets simultaneously

### References

- [Source: _bmad-output/epics.md#Story 2.1] - Complete user story and acceptance criteria
- [Source: _bmad-output/architecture.md#数据隔离] - Dataset as isolation mechanism
- [Source: _bmad-output/architecture.md#API 设计] - URL routing pattern /logdatasets/{dataset}/logs
- [Source: _bmad-output/1-5-core-api-handler-with-go-restful.md] - Base API handler to enhance
- [Source: _bmad-output/1-4-basic-log-query-service.md] - Service layer to enhance with dataset scoping
- [Source: _bmad-output/1-3-clickhouse-repository-layer.md] - Repository layer to enhance with dataset validation
- [Source: _bmad-output/1-2-clickhouse-database-schema-setup.md] - ClickHouse schema with dataset column
- [Source: pkg/oapis/log/v1alpha1/handler.go] - Existing handler to enhance
- [Source: pkg/service/query/service.go] - Service layer to enhance
- [Source: pkg/repository/clickhouse/repository.go] - Repository layer to enhance

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-20250514

### Debug Log References

Enhancing existing API handler (Story 1-5) to provide proper dataset-based query routing and data isolation. Dataset serves as the primary isolation boundary following architecture requirements for multi-tenant log management across edge deployments.

### Completion Notes List

Story 2.1 builds upon the completed foundation stack (Stories 1-1 through 1-5) to implement dataset-based query routing. Enhances existing API handler, service layer, and repository layer with dataset validation, scoping, and security features. Provides comprehensive dataset metadata in responses and prevents cross-dataset access vulnerabilities.

### File List

Primary files to be enhanced/created:
- pkg/service/query/service.go (enhance with dataset scoping)
- pkg/service/query/dataset_validator.go (new)
- pkg/repository/clickhouse/repository.go (enhance with dataset methods)
- pkg/repository/clickhouse/dataset_queries.go (new)
- pkg/oapis/log/v1alpha1/handler.go (enhance with dataset validation)
- pkg/oapis/log/v1alpha1/dataset_errors.go (new)
- pkg/oapis/log/v1alpha1/dataset_metrics.go (new)
- pkg/model/response/log.go (enhance with dataset metadata)
- pkg/service/query/service_test.go (enhance with dataset tests)
- pkg/repository/clickhouse/repository_test.go (enhance with dataset tests)
- pkg/oapis/log/v1alpha1/handler_test.go (enhance with dataset tests)