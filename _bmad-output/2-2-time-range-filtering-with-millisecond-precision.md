# Story 2.2: Time-range filtering with millisecond precision

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want to filter logs by precise time ranges with millisecond accuracy,
So that I can investigate incidents that happened at specific moments and correlate events across distributed edge systems with high temporal precision.

## Acceptance Criteria

**Given** Dataset routing is implemented (Story 2-1 completed)
**When** I specify start_time and end_time parameters
**Then** Only logs within the exact time range are returned
**And** Time filtering supports millisecond precision using DateTime64(9) from ClickHouse schema
**And** Time parameters accept ISO 8601 format with optional millisecond component
**And** Time zone handling is consistent with UTC normalization
**And** Proper error messages are shown for invalid time formats
**And** Query performance is optimized for time-range filtering with proper indexing
**And** Time boundaries are inclusive (start <= timestamp <= end)

## Tasks / Subtasks

- [ ] Enhance time parameter validation and parsing (AC: 3, 5)
  - [ ] Create TimeRangeValidator with comprehensive ISO 8601 parsing support
  - [ ] Support multiple time format patterns (RFC3339, ISO 8601 with/without milliseconds)
  - [ ] Implement timezone normalization to UTC for consistent storage/queries
  - [ ] Add validation for maximum time range spans to prevent expensive queries
  - [ ] Create clear error messages for malformed time strings
- [ ] Optimize time-range query building for ClickHouse (AC: 2, 6)
  - [ ] Enhance service layer to use millisecond-precision time comparisons
  - [ ] Ensure time queries leverage ClickHouse DateTime64(9) optimization
  - [ ] Add time-range specific query patterns for optimal partition pruning
  - [ ] Implement time boundary validation to prevent future time queries
  - [ ] Add query plan analysis for time-range performance monitoring
- [ ] Implement time boundary semantics and edge case handling (AC: 7)
  - [ ] Define inclusive time boundaries (start_time <= timestamp <= end_time)
  - [ ] Handle edge cases with identical start/end times (microsecond queries)
  - [ ] Add validation for maximum query time span (prevent > 24 hour queries)
  - [ ] Implement proper handling of partial millisecond timestamps
  - [ ] Add support for relative time expressions (last 1h, last 30m)
- [ ] Enhance API layer time parameter processing (AC: 3, 4, 5)
  - [ ] Update request parsing to handle millisecond precision timestamps
  - [ ] Add comprehensive time format documentation to API specs
  - [ ] Implement query parameter normalization for time inputs
  - [ ] Add time range validation middleware to prevent expensive queries
  - [ ] Create standardized time format error responses
- [ ] Add performance monitoring for time-range queries (AC: 6)
  - [ ] Create metrics for time-range query duration distribution
  - [ ] Monitor partition scan efficiency for time-based queries
  - [ ] Add alerting for slow time-range queries (> 500ms for bounded ranges)
  - [ ] Track time range query patterns and optimization opportunities
  - [ ] Implement query complexity scoring based on time span

## Dev Notes

### Architecture Compliance Requirements

**Critical:** This story enhances the existing dataset-based query system (Story 2-1) to provide high-precision time filtering capabilities. Time filtering serves as the secondary filtering mechanism after dataset scoping, following the architecture's requirement for "time-range based log filtering with millisecond precision" (FR6).

**Key Technical Requirements:**
- **Millisecond Precision:** Leverage existing ClickHouse DateTime64(9) schema for nanosecond-level precision
- **Performance Optimization:** Time queries must maintain sub-2 second response times with proper indexing
- **UTC Normalization:** All time inputs normalized to UTC for consistent querying across edge deployments
- **Query Safety:** Prevent expensive unbounded time queries that could impact system performance
- **ISO 8601 Compliance:** Support standard time formats for international edge deployments

### Time Range Filtering Implementation

**Based on architecture.md specifications and Story 2-1 foundation, implementing high-precision time filtering:**

```go
// Enhanced time range validator for millisecond precision
package query

import (
    "fmt"
    "time"
    "regexp"
)

type TimeRangeValidator struct {
    maxTimeSpan     time.Duration
    timeFormats     []string
    iso8601Regex    *regexp.Regexp
}

func NewTimeRangeValidator() *TimeRangeValidator {
    return &TimeRangeValidator{
        maxTimeSpan: 24 * time.Hour, // Prevent expensive queries
        timeFormats: []string{
            time.RFC3339,                    // 2006-01-02T15:04:05Z07:00
            "2006-01-02T15:04:05.000Z07:00", // With milliseconds
            "2006-01-02T15:04:05.000000Z",   // With microseconds UTC
            "2006-01-02T15:04:05.000000000Z", // With nanoseconds UTC
            "2006-01-02T15:04:05",           // Local time (converted to UTC)
            "2006-01-02 15:04:05",           // SQL format
        },
        iso8601Regex: regexp.MustCompile(
            `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?(Z|[+-]\d{2}:\d{2})?$`,
        ),
    }
}

// ValidateAndParseTimeRange validates and normalizes time range inputs
func (v *TimeRangeValidator) ValidateAndParseTimeRange(startStr, endStr string) (*time.Time, *time.Time, error) {
    var startTime, endTime *time.Time
    var err error

    // Parse start time if provided
    if startStr != "" {
        startTime, err = v.parseTimeString(startStr)
        if err != nil {
            return nil, nil, fmt.Errorf("invalid start_time format '%s': %w", startStr, err)
        }
    }

    // Parse end time if provided
    if endStr != "" {
        endTime, err = v.parseTimeString(endStr)
        if err != nil {
            return nil, nil, fmt.Errorf("invalid end_time format '%s': %w", endStr, err)
        }
    }

    // Validate time range logic
    if err := v.validateTimeRange(startTime, endTime); err != nil {
        return nil, nil, err
    }

    return startTime, endTime, nil
}

// parseTimeString attempts to parse time string in multiple formats
func (v *TimeRangeValidator) parseTimeString(timeStr string) (*time.Time, error) {
    // Validate format using regex first
    if !v.iso8601Regex.MatchString(timeStr) {
        return nil, fmt.Errorf("time format must be ISO 8601 compliant")
    }

    // Try parsing with multiple formats
    for _, format := range v.timeFormats {
        if t, err := time.Parse(format, timeStr); err == nil {
            // Normalize to UTC for consistent storage/querying
            utcTime := t.UTC()
            return &utcTime, nil
        }
    }

    // If standard parsing fails, try parsing with custom millisecond handling
    if t, err := v.parseWithMillisecondHandling(timeStr); err == nil {
        utcTime := t.UTC()
        return &utcTime, nil
    }

    return nil, fmt.Errorf("unsupported time format, supported formats: RFC3339, ISO 8601 with optional milliseconds")
}

// parseWithMillisecondHandling handles edge cases in millisecond parsing
func (v *TimeRangeValidator) parseWithMillisecondHandling(timeStr string) (time.Time, error) {
    // Handle different millisecond precision formats
    formats := []string{
        "2006-01-02T15:04:05.999999999Z", // Nanoseconds
        "2006-01-02T15:04:05.999999Z",    // Microseconds
        "2006-01-02T15:04:05.999Z",       // Milliseconds
        "2006-01-02T15:04:05.99Z",        // Centiseconds
        "2006-01-02T15:04:05.9Z",         // Deciseconds
    }

    for _, format := range formats {
        if t, err := time.Parse(format, timeStr); err == nil {
            return t, nil
        }
    }

    return time.Time{}, fmt.Errorf("unable to parse time with millisecond precision")
}

// validateTimeRange ensures time range is logical and within limits
func (v *TimeRangeValidator) validateTimeRange(startTime, endTime *time.Time) error {
    now := time.Now().UTC()

    // Check for future times
    if startTime != nil && startTime.After(now) {
        return fmt.Errorf("start_time cannot be in the future")
    }
    if endTime != nil && endTime.After(now) {
        return fmt.Errorf("end_time cannot be in the future")
    }

    // Check time order
    if startTime != nil && endTime != nil {
        if startTime.After(*endTime) {
            return fmt.Errorf("start_time (%v) must be before or equal to end_time (%v)",
                startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))
        }

        // Check maximum time span to prevent expensive queries
        timeSpan := endTime.Sub(*startTime)
        if timeSpan > v.maxTimeSpan {
            return fmt.Errorf("time range span (%v) exceeds maximum allowed span (%v)",
                timeSpan, v.maxTimeSpan)
        }

        // Warn for very small time ranges (may indicate precision issues)
        if timeSpan < time.Millisecond {
            // Allow but log warning for sub-millisecond queries
            klog.V(2).InfoS("Sub-millisecond time range query",
                "start_time", startTime.Format(time.RFC3339Nano),
                "end_time", endTime.Format(time.RFC3339Nano),
                "span_ns", timeSpan.Nanoseconds())
        }
    }

    return nil
}
```

### Enhanced Service Layer with Millisecond Precision

**Enhanced service layer to support high-precision time filtering:**

```go
// Enhanced LogQueryRequest validation with time precision
func (s *Service) validateQueryRequest(req *request.LogQueryRequest) error {
    // Existing dataset validation from Story 2.1
    if err := s.datasetValidator.ValidateDataset(req.Dataset); err != nil {
        return fmt.Errorf("dataset validation failed: %w", err)
    }

    // Enhanced time range validation with millisecond precision
    if err := s.validateTimeRange(req); err != nil {
        return fmt.Errorf("time range validation failed: %w", err)
    }

    // Additional validation logic...
    return s.validatePagination(req)
}

// validateTimeRange provides comprehensive time range validation
func (s *Service) validateTimeRange(req *request.LogQueryRequest) error {
    validator := NewTimeRangeValidator()

    // Convert time.Time back to string for validation if needed
    var startStr, endStr string
    if req.StartTime != nil {
        startStr = req.StartTime.Format(time.RFC3339Nano)
    }
    if req.EndTime != nil {
        endStr = req.EndTime.Format(time.RFC3339Nano)
    }

    // Validate and normalize time range
    normalizedStart, normalizedEnd, err := validator.ValidateAndParseTimeRange(startStr, endStr)
    if err != nil {
        return err
    }

    // Update request with normalized times
    req.StartTime = normalizedStart
    req.EndTime = normalizedEnd

    return nil
}

// Enhanced query building with millisecond precision support
func (s *Service) buildTimeRangeQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
    var whereConditions []string
    var args []interface{}

    // Dataset must be first WHERE condition (from Story 2.1)
    whereConditions = append(whereConditions, "dataset = ?")
    args = append(args, req.Dataset)

    // Add high-precision time range filters
    if req.StartTime != nil {
        // Use >= for inclusive start boundary
        whereConditions = append(whereConditions, "timestamp >= ?")
        args = append(args, *req.StartTime)
    }

    if req.EndTime != nil {
        // Use <= for inclusive end boundary
        whereConditions = append(whereConditions, "timestamp <= ?")
        args = append(args, *req.EndTime)
    }

    // Additional filters (namespace, pod, content, etc.) from Story 2.1
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

    // Build optimized query with proper ordering for time-range queries
    query := fmt.Sprintf(`
        SELECT
            timestamp,
            content,
            severity,
            k8s_namespace_name,
            k8s_pod_name,
            k8s_node_name,
            host_ip,
            host_name,
            container_name,
            container_id
        FROM logs
        WHERE %s
        ORDER BY timestamp DESC, host_ip ASC
        LIMIT %d OFFSET %d
    `, strings.Join(whereConditions, " AND "), req.PageSize, req.Page*req.PageSize)

    return query, args, nil
}
```

### API Layer Enhancement for Time Parameter Processing

**Enhanced API handler to support millisecond-precision time parsing:**

```go
// Enhanced time parameter parsing in API handler
func (h *LogHandler) parseTimeParameters(req *restful.Request) (*time.Time, *time.Time, error) {
    var startTime, endTime *time.Time
    var err error

    // Parse start_time parameter
    startTimeStr := req.QueryParameter("start_time")
    if startTimeStr != "" {
        validator := NewTimeRangeValidator()
        startTime, err = validator.parseTimeString(startTimeStr)
        if err != nil {
            return nil, nil, fmt.Errorf("invalid start_time parameter: %w", err)
        }
    }

    // Parse end_time parameter
    endTimeStr := req.QueryParameter("end_time")
    if endTimeStr != "" {
        validator := NewTimeRangeValidator()
        endTime, err = validator.parseTimeString(endTimeStr)
        if err != nil {
            return nil, nil, fmt.Errorf("invalid end_time parameter: %w", err)
        }
    }

    // Apply time range validation
    validator := NewTimeRangeValidator()
    if err := validator.validateTimeRange(startTime, endTime); err != nil {
        return nil, nil, fmt.Errorf("time range validation failed: %w", err)
    }

    return startTime, endTime, nil
}

// Enhanced query parsing with time precision support
func (h *LogHandler) parseQueryRequest(req *restful.Request, dataset string) (*request.LogQueryRequest, error) {
    // Parse time parameters with millisecond precision
    startTime, endTime, err := h.parseTimeParameters(req)
    if err != nil {
        return nil, err
    }

    // Build request with enhanced time handling
    queryReq := &request.LogQueryRequest{
        Dataset:   dataset,
        StartTime: startTime,
        EndTime:   endTime,

        // Other parameters from existing implementation
        Filter:        req.QueryParameter("filter"),
        Severity:      req.QueryParameter("severity"),
        Namespace:     req.QueryParameter("namespace"),
        PodName:       req.QueryParameter("pod_name"),
        NodeName:      req.QueryParameter("node_name"),
        HostIP:        req.QueryParameter("host_ip"),
        HostName:      req.QueryParameter("host_name"),
        ContainerName: req.QueryParameter("container_name"),
    }

    // Parse pagination parameters
    if err := h.parsePaginationParameters(req, queryReq); err != nil {
        return nil, fmt.Errorf("pagination parsing failed: %w", err)
    }

    return queryReq, nil
}
```

### Repository Layer ClickHouse Query Optimization

**Enhanced repository layer for optimal time-range querying:**

```go
// Enhanced ClickHouse query execution with time optimization
func (r *ClickHouseRepository) QueryLogs(ctx context.Context, req *request.LogQueryRequest) ([]clickhouse.LogEntry, int, error) {
    // Build optimized query with time range
    query, args, err := r.buildOptimizedTimeQuery(req)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to build time query: %w", err)
    }

    // Log query execution details for monitoring
    klog.InfoS("Executing time-range query",
        "dataset", req.Dataset,
        "start_time", r.formatTimeForLog(req.StartTime),
        "end_time", r.formatTimeForLog(req.EndTime),
        "estimated_partitions", r.estimatePartitionCount(req))

    // Execute with context timeout
    queryCtx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
    defer cancel()

    startTime := time.Now()
    rows, err := r.conn.Query(queryCtx, query, args...)
    if err != nil {
        return nil, 0, fmt.Errorf("query execution failed: %w", err)
    }
    defer rows.Close()

    // Parse results with time precision preservation
    logs, err := r.parseLogsWithTimePrecision(rows)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to parse logs: %w", err)
    }

    // Get total count for pagination
    totalCount, err := r.getTotalCountForTimeRange(ctx, req)
    if err != nil {
        klog.ErrorS(err, "Failed to get total count", "dataset", req.Dataset)
        // Continue with results but set count to length
        totalCount = len(logs)
    }

    duration := time.Since(startTime)
    klog.InfoS("Time-range query completed",
        "dataset", req.Dataset,
        "result_count", len(logs),
        "total_count", totalCount,
        "duration_ms", duration.Milliseconds())

    // Performance monitoring
    if duration > 500*time.Millisecond {
        klog.InfoS("Slow time-range query detected",
            "dataset", req.Dataset,
            "duration_ms", duration.Milliseconds(),
            "time_span", r.calculateTimeSpan(req))
    }

    return logs, totalCount, nil
}

// buildOptimizedTimeQuery creates ClickHouse-optimized queries for time ranges
func (r *ClickHouseRepository) buildOptimizedTimeQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
    var whereConditions []string
    var args []interface{}

    // Dataset first for partition pruning (Story 2.1)
    whereConditions = append(whereConditions, "dataset = ?")
    args = append(args, req.Dataset)

    // Time range conditions with inclusive boundaries
    if req.StartTime != nil {
        whereConditions = append(whereConditions, "timestamp >= toDateTime64(?, 9)")
        args = append(args, req.StartTime.Unix())
    }

    if req.EndTime != nil {
        whereConditions = append(whereConditions, "timestamp <= toDateTime64(?, 9)")
        args = append(args, req.EndTime.Unix())
    }

    // Additional filters...

    // Build query with optimal ordering for time-range scans
    query := fmt.Sprintf(`
        SELECT
            timestamp,
            content,
            severity,
            k8s_namespace_name,
            k8s_pod_name,
            k8s_node_name,
            host_ip,
            host_name,
            container_name
        FROM logs
        WHERE %s
        ORDER BY timestamp DESC
        LIMIT %d OFFSET %d
    `, strings.Join(whereConditions, " AND "), req.PageSize, req.Page*req.PageSize)

    return query, args, nil
}

// parseLogsWithTimePrecision ensures millisecond precision is preserved
func (r *ClickHouseRepository) parseLogsWithTimePrecision(rows driver.Rows) ([]clickhouse.LogEntry, error) {
    var logs []clickhouse.LogEntry

    for rows.Next() {
        var entry clickhouse.LogEntry
        var timestamp time.Time

        err := rows.Scan(
            &timestamp,
            &entry.Content,
            &entry.Severity,
            &entry.K8sNamespaceName,
            &entry.K8sPodName,
            &entry.K8sNodeName,
            &entry.HostIP,
            &entry.HostName,
            &entry.ContainerName,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan row: %w", err)
        }

        // Preserve full timestamp precision
        entry.Timestamp = timestamp
        logs = append(logs, entry)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("row iteration error: %w", err)
    }

    return logs, nil
}
```

### Performance Monitoring and Query Optimization

**Enhanced metrics and monitoring for time-range queries:**

```go
// TimeRangeMetrics tracks performance of time-based queries
type TimeRangeMetrics struct {
    queryDuration    *prometheus.HistogramVec
    partitionScans   *prometheus.CounterVec
    timeSpanRequests *prometheus.HistogramVec
    slowQueries      *prometheus.CounterVec
}

func NewTimeRangeMetrics() *TimeRangeMetrics {
    return &TimeRangeMetrics{
        queryDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_time_query_duration_seconds",
                Help:    "Duration of time-range queries",
                Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
            },
            []string{"dataset", "time_span_category"},
        ),
        partitionScans: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_partition_scans_total",
                Help: "Number of partitions scanned for time queries",
            },
            []string{"dataset"},
        ),
        timeSpanRequests: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_time_span_hours",
                Help:    "Time span of queries in hours",
                Buckets: []float64{0.25, 0.5, 1, 6, 12, 24},
            },
            []string{"dataset"},
        ),
        slowQueries: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_slow_time_queries_total",
                Help: "Number of slow time-range queries (>500ms)",
            },
            []string{"dataset", "reason"},
        ),
    }
}

// Record time-range query metrics
func (m *TimeRangeMetrics) RecordTimeQuery(dataset string, duration time.Duration, timeSpan time.Duration) {
    // Categorize time span
    var spanCategory string
    hours := timeSpan.Hours()
    switch {
    case hours <= 0.25:
        spanCategory = "sub_hour"
    case hours <= 1:
        spanCategory = "hourly"
    case hours <= 6:
        spanCategory = "multi_hour"
    case hours <= 24:
        spanCategory = "daily"
    default:
        spanCategory = "extended"
    }

    m.queryDuration.With(prometheus.Labels{
        "dataset":            dataset,
        "time_span_category": spanCategory,
    }).Observe(duration.Seconds())

    m.timeSpanRequests.With(prometheus.Labels{
        "dataset": dataset,
    }).Observe(hours)

    // Track slow queries
    if duration > 500*time.Millisecond {
        reason := "unknown"
        if hours > 12 {
            reason = "large_time_span"
        } else if duration > 2*time.Second {
            reason = "timeout_risk"
        } else {
            reason = "processing_heavy"
        }

        m.slowQueries.With(prometheus.Labels{
            "dataset": dataset,
            "reason":  reason,
        }).Inc()
    }
}
```

### API Documentation and Usage Examples

**Enhanced API documentation with millisecond precision examples:**

```bash
# Basic time-range query with millisecond precision
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?start_time=2024-01-01T10:30:45.123Z&end_time=2024-01-01T10:30:45.456Z"

# Sub-second incident investigation
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-cn-hz01/logs?start_time=2024-01-01T14:25:30.000Z&end_time=2024-01-01T14:25:30.999Z&filter=error"

# Microsecond precision query for high-frequency events
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/staging-app/logs?start_time=2024-01-01T09:15:22.123456Z&end_time=2024-01-01T09:15:22.123789Z&namespace=monitoring"

# Response includes millisecond-precise timestamps
{
  "logs": [
    {
      "timestamp": "2024-01-01T14:25:30.123456789Z",
      "content": "Connection timeout to database",
      "severity": "ERROR",
      "k8s_namespace_name": "production",
      "k8s_pod_name": "web-server-abc123"
    }
  ],
  "total_count": 1,
  "page": 0,
  "page_size": 100,
  "has_more": false
}
```

### Error Handling for Time Operations

**Comprehensive error handling for time-related operations:**

```go
// Time-specific error types
type TimeValidationError struct {
    Field   string
    Value   string
    Reason  string
}

func (e *TimeValidationError) Error() string {
    return fmt.Sprintf("time validation failed for %s='%s': %s", e.Field, e.Value, e.Reason)
}

type TimeRangeError struct {
    StartTime *time.Time
    EndTime   *time.Time
    Issue     string
}

func (e *TimeRangeError) Error() string {
    return fmt.Sprintf("time range error: %s (start: %v, end: %v)",
        e.Issue,
        formatOptionalTime(e.StartTime),
        formatOptionalTime(e.EndTime))
}

// Enhanced error responses for time operations
func (h *LogHandler) handleTimeError(resp *restful.Response, err error, dataset string) {
    switch e := err.(type) {
    case *TimeValidationError:
        errorResp := map[string]interface{}{
            "error":   "Invalid time format",
            "field":   e.Field,
            "value":   e.Value,
            "reason":  e.Reason,
            "dataset": dataset,
            "supported_formats": []string{
                "RFC3339: 2006-01-02T15:04:05Z",
                "With milliseconds: 2006-01-02T15:04:05.123Z",
                "With microseconds: 2006-01-02T15:04:05.123456Z",
                "With nanoseconds: 2006-01-02T15:04:05.123456789Z",
            },
            "examples": []string{
                "start_time=2024-01-01T10:30:45.123Z",
                "end_time=2024-01-01T10:30:45.456Z",
            },
        }
        h.writeErrorResponse(resp, http.StatusBadRequest, errorResp)
        h.metrics.RecordTimeError(dataset, "validation_error")

    case *TimeRangeError:
        errorResp := map[string]interface{}{
            "error":      "Invalid time range",
            "issue":      e.Issue,
            "start_time": formatOptionalTime(e.StartTime),
            "end_time":   formatOptionalTime(e.EndTime),
            "dataset":    dataset,
            "guidelines": map[string]string{
                "max_span":      "24 hours",
                "boundary":      "inclusive (start <= timestamp <= end)",
                "timezone":      "all times converted to UTC",
                "future_times":  "not allowed",
            },
        }
        h.writeErrorResponse(resp, http.StatusBadRequest, errorResp)
        h.metrics.RecordTimeError(dataset, "range_error")

    default:
        h.writeErrorResponse(resp, http.StatusBadRequest, "Time parameter error: "+err.Error())
        h.metrics.RecordTimeError(dataset, "unknown_error")
    }
}
```

### Testing Strategy for Time Precision

**Comprehensive testing strategy for millisecond precision:**

1. **Unit Tests:**
   - Time format parsing with various ISO 8601 formats
   - Timezone conversion and UTC normalization
   - Boundary condition testing (inclusive ranges)
   - Error handling for invalid time formats
   - Performance testing for time validation logic

2. **Integration Tests:**
   - End-to-end time-range queries with millisecond precision
   - ClickHouse DateTime64(9) integration testing
   - Query performance with various time spans
   - Partition pruning effectiveness for time-range queries

3. **Performance Tests:**
   - Time-range query performance benchmarking
   - Memory usage for high-precision time operations
   - Concurrent time-range query handling
   - Large time span query optimization

### Project Structure Notes

**File organization building on Story 2-1 foundation:**

```
pkg/service/query/
├── service.go                  # Enhanced with time validation (modify existing)
├── time_validator.go           # Time range validation logic (new)
├── dataset_validator.go        # Existing from Story 2.1
└── service_test.go            # Enhanced with time precision tests (modify existing)

pkg/repository/clickhouse/
├── repository.go              # Enhanced with time query optimization (modify existing)
├── time_queries.go            # Time-specific query patterns (new)
└── repository_test.go         # Enhanced with time precision tests (modify existing)

pkg/oapis/log/v1alpha1/
├── handler.go                 # Enhanced with time parameter processing (modify existing)
├── time_errors.go             # Time-specific error types (new)
├── time_metrics.go            # Time query metrics (new)
└── handler_test.go            # Enhanced with time endpoint tests (modify existing)

pkg/model/request/
└── log.go                     # Enhanced with time validation (modify existing)

pkg/model/response/
└── log.go                     # Enhanced with precise timestamp format (modify existing)
```

**Key Integration Points:**
- Enhances existing pkg/service/query/service.go from Story 2-1 with time precision
- Builds upon pkg/oapis/log/v1alpha1/handler.go time parameter parsing
- Extends pkg/repository/clickhouse/repository.go with time-optimized queries
- Uses existing ClickHouse DateTime64(9) schema from Story 1-2
- Integrates with dataset validation from Story 2-1

### Dependencies and Version Requirements

**No new dependencies required - leverages existing stack:**

```go
// Existing dependencies from previous stories
require (
    github.com/ClickHouse/clickhouse-go/v2 v2.15.0  // DateTime64 support
    github.com/emicklei/go-restful/v3 v3.11.0        // API framework
    k8s.io/klog/v2 v2.100.1                          // Structured logging
    github.com/prometheus/client_golang v1.17.0      // Time metrics
    github.com/stretchr/testify v1.8.4                // Testing framework
)
```

### Performance Requirements

**Time-range filtering performance targets:**

- **Time Validation:** < 1ms per request for format parsing and validation
- **Query Building:** < 5ms additional latency for time-range query construction
- **ClickHouse Execution:** < 500ms for bounded time-range queries (< 1 hour span)
- **Partition Pruning:** Effective pruning for time-based partitions
- **Memory Usage:** < 10MB additional memory for time precision operations
- **Concurrent Time Queries:** Support 100+ simultaneous time-range queries

### References

- [Source: _bmad-output/epics.md#Story 2.2] - Complete user story and acceptance criteria
- [Source: _bmad-output/architecture.md#FR6] - Time-range filtering with millisecond precision requirement
- [Source: _bmad-output/2-1-dataset-based-query-routing.md] - Foundation for time filtering enhancement
- [Source: sqlscripts/clickhouse/01_tables.sql#timestamp] - ClickHouse DateTime64(9) schema
- [Source: pkg/model/request/log.go#StartTime] - Existing time field definitions
- [Source: pkg/service/query/service.go] - Service layer to enhance with time precision
- [Source: pkg/repository/clickhouse/repository.go] - Repository layer for time-optimized queries

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-20250514

### Debug Log References

Enhancing existing dataset-based query system (Story 2-1) to provide high-precision time filtering capabilities with millisecond accuracy. Time filtering serves as the secondary filtering mechanism after dataset scoping, leveraging existing ClickHouse DateTime64(9) schema for nanosecond-level precision.

### Completion Notes List

Story 2.2 builds upon the completed dataset routing foundation (Story 2-1) to implement high-precision time-range filtering. Enhances existing service, repository, and API layers with comprehensive time validation, UTC normalization, and performance optimization. Provides millisecond-accurate incident investigation capabilities while maintaining sub-2 second query performance requirements.

### File List

Primary files to be enhanced/created:
- pkg/service/query/service.go (enhance with time precision validation)
- pkg/service/query/time_validator.go (new)
- pkg/repository/clickhouse/repository.go (enhance with time-optimized queries)
- pkg/repository/clickhouse/time_queries.go (new)
- pkg/oapis/log/v1alpha1/handler.go (enhance with time parameter processing)
- pkg/oapis/log/v1alpha1/time_errors.go (new)
- pkg/oapis/log/v1alpha1/time_metrics.go (new)
- pkg/model/request/log.go (enhance with time validation)
- pkg/model/response/log.go (enhance with timestamp precision)
- pkg/service/query/service_test.go (enhance with time precision tests)
- pkg/repository/clickhouse/repository_test.go (enhance with time integration tests)
- pkg/oapis/log/v1alpha1/handler_test.go (enhance with time endpoint tests)