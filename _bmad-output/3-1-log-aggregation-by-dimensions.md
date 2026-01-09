# Story 3.1: Log aggregation by dimensions

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want to aggregate logs by different dimensions (severity, namespace, host, time buckets) with multiple aggregation functions,
So that I can understand patterns, trends, and operational insights across my edge computing infrastructure for data-driven troubleshooting and capacity planning.

## Acceptance Criteria

**Given** Epic 2 comprehensive filtering foundation is implemented (dataset routing, time filtering, K8s filtering, content search)
**When** I use the aggregation API endpoint with various dimension configurations
**Then** I can group logs by single or multiple dimensions including severity, namespace, host_name, container_name, time buckets, and custom intervals
**And** Aggregation results include count, sum, average, min, max aggregations for log metrics with statistical analysis
**And** Time-based bucketing provides trend analysis with configurable intervals (hourly, daily, custom periods)
**And** Multiple dimensions can be combined in a single aggregation query with hierarchical grouping
**And** Aggregation queries leverage ClickHouse GROUP BY optimization for large-scale performance
**And** Results are properly formatted for visualization and dashboard integration with metadata
**And** Aggregation integrates seamlessly with all Epic 2 filtering capabilities (dataset, time, K8s, content search)
**And** Aggregation result caching provides fast repeated query performance
**And** Aggregation queries maintain sub-2 second response times even for complex multi-dimensional analysis

## Tasks / Subtasks

- [ ] Implement comprehensive aggregation dimension framework (AC: 1, 3)
  - [ ] Create AggregationDimensionValidator with support for multiple dimension types
  - [ ] Support single dimension aggregation (severity, namespace, host, container)
  - [ ] Implement multi-dimensional aggregation with hierarchical grouping
  - [ ] Add time-based dimension support with configurable bucket intervals
  - [ ] Support custom time intervals (1min, 5min, 15min, 1hour, 6hour, 12hour, 1day, 1week)
  - [ ] Add dimension combination validation to prevent expensive query patterns
  - [ ] Implement dimension cardinality estimation for query optimization

- [ ] Develop comprehensive aggregation function support (AC: 2, 8)
  - [ ] Implement count aggregation for log entry counting by dimensions
  - [ ] Add sum aggregation for numeric log attributes (message length, processing time)
  - [ ] Support average aggregation for performance metrics and durations
  - [ ] Implement min/max aggregation for timestamp ranges and numeric values
  - [ ] Add percentile aggregation (p50, p95, p99) for performance analysis
  - [ ] Support distinct count aggregation for unique value counting
  - [ ] Implement rate aggregation for time-based trend calculation
  - [ ] Add custom aggregation expressions for advanced analytics

- [ ] Build optimized ClickHouse aggregation queries with GROUP BY optimization (AC: 5, 8)
  - [ ] Create AggregationQueryBuilder for efficient ClickHouse GROUP BY queries
  - [ ] Implement optimal column ordering for aggregation performance
  - [ ] Add query optimization hints for large-scale aggregation operations
  - [ ] Support materialized view leveraging for frequently-used aggregations
  - [ ] Implement incremental aggregation for real-time dashboard updates
  - [ ] Add aggregation query parallelization for multi-dataset scenarios
  - [ ] Create aggregation result compression for memory efficiency

- [ ] Integrate aggregation with Epic 2 comprehensive filtering foundation (AC: 7)
  - [ ] Enhance service layer to combine dataset routing with aggregation dimensions
  - [ ] Integrate time filtering as pre-aggregation filter for optimal performance
  - [ ] Combine K8s metadata filtering with namespace/pod aggregation dimensions
  - [ ] Support content search filtering as aggregation input for focused analysis
  - [ ] Maintain filter precedence: dataset → time → K8s → content → aggregation
  - [ ] Ensure aggregation works with all existing filter combinations
  - [ ] Add aggregation validation to existing request validation pipeline

- [ ] Implement aggregation result caching and performance optimization (AC: 8, 9)
  - [ ] Create AggregationCache with configurable TTL for different aggregation types
  - [ ] Implement cache key generation based on filters and aggregation dimensions
  - [ ] Add cache invalidation strategies for real-time data updates
  - [ ] Support cache warming for frequently-accessed aggregation patterns
  - [ ] Implement aggregation result compression for large result sets
  - [ ] Add performance monitoring for aggregation query execution times
  - [ ] Create aggregation complexity scoring for optimization guidance

- [ ] Develop aggregation visualization and dashboard integration support (AC: 6)
  - [ ] Format aggregation results for time-series visualization frameworks
  - [ ] Add chart-ready data structures (labels, datasets, time series)
  - [ ] Support multiple output formats (JSON, CSV, metrics format)
  - [ ] Implement data aggregation summarization for high-level dashboard views
  - [ ] Add trend calculation and percentage change analysis
  - [ ] Support drill-down capabilities from aggregated data to raw logs
  - [ ] Create aggregation metadata for visualization guidance

## Dev Notes

### Architecture Compliance Requirements

**Critical:** Story 3-1 initiates Epic 3: Advanced Query and Analytics by implementing powerful log aggregation capabilities on top of the comprehensive filtering foundation built in Epic 2. This story leverages the established dataset routing (2-1), time filtering (2-2), K8s filtering (2-3), and content search (2-4) to provide advanced analytics capabilities for edge computing log analysis, following the architecture's requirement for "log aggregation by dimensions with multiple aggregation functions and time-based bucketing for trend analysis."

**Key Technical Requirements:**
- **Multi-Dimensional Aggregation:** Support flexible dimension combinations with hierarchical grouping
- **Advanced Analytics Functions:** Comprehensive aggregation functions (count, sum, avg, min, max, percentiles, rates)
- **Time-Based Bucketing:** Configurable time intervals for trend analysis and operational insights
- **Performance Optimization:** Leverage ClickHouse GROUP BY optimization for sub-2 second aggregation response times
- **Filtering Integration:** Seamless integration with all Epic 2 filtering capabilities
- **Caching Strategy:** Intelligent result caching for improved repeated query performance

### Aggregation Framework Implementation

**Based on Epic 2 filtering foundation and architecture.md specifications, implementing comprehensive aggregation framework:**

```go
// Advanced aggregation framework for edge-logs analytics
package aggregation

import (
    "fmt"
    "regexp"
    "strings"
    "time"
    "unicode/utf8"
)

// AggregationDimensionType defines supported aggregation dimensions
type AggregationDimensionType string

const (
    DimensionSeverity       AggregationDimensionType = "severity"
    DimensionNamespace      AggregationDimensionType = "namespace"
    DimensionPodName        AggregationDimensionType = "pod_name"
    DimensionNodeName       AggregationDimensionType = "node_name"
    DimensionHostName       AggregationDimensionType = "host_name"
    DimensionContainerName  AggregationDimensionType = "container_name"
    DimensionTimestamp      AggregationDimensionType = "timestamp"
    DimensionDataset        AggregationDimensionType = "dataset"
    DimensionCustomField    AggregationDimensionType = "custom_field"
)

// AggregationFunctionType defines supported aggregation functions
type AggregationFunctionType string

const (
    FunctionCount          AggregationFunctionType = "count"
    FunctionSum            AggregationFunctionType = "sum"
    FunctionAvg            AggregationFunctionType = "avg"
    FunctionMin            AggregationFunctionType = "min"
    FunctionMax            AggregationFunctionType = "max"
    FunctionPercentile50   AggregationFunctionType = "p50"
    FunctionPercentile95   AggregationFunctionType = "p95"
    FunctionPercentile99   AggregationFunctionType = "p99"
    FunctionDistinctCount  AggregationFunctionType = "distinct_count"
    FunctionRate           AggregationFunctionType = "rate"
    FunctionStdDev         AggregationFunctionType = "stddev"
    FunctionCustom         AggregationFunctionType = "custom"
)

// TimeBucketInterval defines time bucketing intervals
type TimeBucketInterval string

const (
    IntervalMinute     TimeBucketInterval = "1m"
    Interval5Minutes   TimeBucketInterval = "5m"
    Interval15Minutes  TimeBucketInterval = "15m"
    IntervalHour       TimeBucketInterval = "1h"
    Interval6Hours     TimeBucketInterval = "6h"
    Interval12Hours    TimeBucketInterval = "12h"
    IntervalDay        TimeBucketInterval = "1d"
    IntervalWeek       TimeBucketInterval = "1w"
    IntervalCustom     TimeBucketInterval = "custom"
)

// AggregationDimension represents a single aggregation dimension
type AggregationDimension struct {
    Type            AggregationDimensionType
    Field           string        // For custom fields
    TimeBucket      TimeBucketInterval // For timestamp dimensions
    CustomInterval  time.Duration      // For custom time intervals
    Alias           string        // Output field alias
    SortOrder       string        // ASC, DESC
    Limit           int          // Limit results for this dimension
}

// AggregationFunction represents a single aggregation function
type AggregationFunction struct {
    Type              AggregationFunctionType
    Field             string        // Field to aggregate
    CustomExpression  string        // For custom aggregation functions
    Alias             string        // Output field alias
    Parameters        map[string]interface{} // Function-specific parameters
}

// AggregationRequest represents a complete aggregation query
type AggregationRequest struct {
    // Inherited filtering from Epic 2
    Dataset        string                    // From Story 2-1: Dataset routing
    StartTime      *time.Time               // From Story 2-2: Time filtering
    EndTime        *time.Time               // From Story 2-2: Time filtering
    K8sFilters     *K8sFilterExpression     // From Story 2-3: K8s filtering
    ContentSearch  *ContentSearchExpression  // From Story 2-4: Content search

    // New aggregation-specific fields
    Dimensions     []AggregationDimension
    Functions      []AggregationFunction
    GroupBy        []string                 // Explicit GROUP BY clause control
    Having         string                   // HAVING clause for post-aggregation filtering
    OrderBy        []string                 // Result ordering
    Limit          int                      // Result limit
    Offset         int                      // Result pagination

    // Performance and caching
    UseCache       bool                     // Enable result caching
    CacheTTL       time.Duration           // Cache time-to-live
    MaxComplexity  float64                 // Maximum allowed query complexity

    // Output formatting
    OutputFormat   string                   // json, csv, metrics
    TimeZone       string                   // Timezone for timestamp formatting
    Precision      int                      // Decimal precision for numeric results
}

// AggregationDimensionValidator validates aggregation dimensions and functions
type AggregationDimensionValidator struct {
    maxDimensions      int
    maxFunctions       int
    maxComplexity      float64
    allowedCustomFields map[string]bool
    cardinalityLimits  map[AggregationDimensionType]int
}

func NewAggregationDimensionValidator() *AggregationDimensionValidator {
    return &AggregationDimensionValidator{
        maxDimensions:     8,  // Prevent overly complex aggregations
        maxFunctions:      12, // Comprehensive function support
        maxComplexity:     100.0,
        allowedCustomFields: map[string]bool{
            "message_length": true,
            "processing_time": true,
            "error_code": true,
            "trace_id": true,
        },
        cardinalityLimits: map[AggregationDimensionType]int{
            DimensionSeverity:      10,     // ERROR, WARNING, INFO, DEBUG, etc.
            DimensionNamespace:     1000,   // Reasonable K8s namespace limit
            DimensionPodName:       10000,  // Large pod environments
            DimensionNodeName:      500,    // Edge node cluster size
            DimensionHostName:      1000,   // Edge host limits
            DimensionContainerName: 5000,   // Container variety
            DimensionDataset:       100,    // Multiple edge datasets
        },
    }
}

// ValidateAggregationRequest validates comprehensive aggregation request
func (v *AggregationDimensionValidator) ValidateAggregationRequest(req *AggregationRequest) error {
    if req == nil {
        return fmt.Errorf("aggregation request cannot be nil")
    }

    // Validate dimension count and complexity
    if len(req.Dimensions) > v.maxDimensions {
        return fmt.Errorf("too many dimensions (%d), maximum allowed: %d",
            len(req.Dimensions), v.maxDimensions)
    }

    if len(req.Functions) == 0 {
        return fmt.Errorf("at least one aggregation function must be specified")
    }

    if len(req.Functions) > v.maxFunctions {
        return fmt.Errorf("too many functions (%d), maximum allowed: %d",
            len(req.Functions), v.maxFunctions)
    }

    // Validate dimensions
    for i, dim := range req.Dimensions {
        if err := v.validateDimension(dim); err != nil {
            return fmt.Errorf("dimension %d validation failed: %w", i, err)
        }
    }

    // Validate functions
    for i, fn := range req.Functions {
        if err := v.validateFunction(fn); err != nil {
            return fmt.Errorf("function %d validation failed: %w", i, err)
        }
    }

    // Validate complexity
    complexity := v.calculateRequestComplexity(req)
    if complexity > v.maxComplexity {
        return fmt.Errorf("aggregation complexity (%.1f) exceeds maximum (%.1f)",
            complexity, v.maxComplexity)
    }

    // Validate time dimension consistency
    if err := v.validateTimeDimensions(req); err != nil {
        return fmt.Errorf("time dimension validation failed: %w", err)
    }

    // Validate cardinality estimates
    if err := v.validateDimensionCardinality(req); err != nil {
        return fmt.Errorf("dimension cardinality validation failed: %w", err)
    }

    return nil
}

// validateDimension validates a single aggregation dimension
func (v *AggregationDimensionValidator) validateDimension(dim AggregationDimension) error {
    // Validate dimension type
    switch dim.Type {
    case DimensionSeverity, DimensionNamespace, DimensionPodName,
         DimensionNodeName, DimensionHostName, DimensionContainerName, DimensionDataset:
        // Standard dimensions - no additional validation needed
    case DimensionTimestamp:
        // Time dimension requires bucket interval
        if dim.TimeBucket == "" {
            return fmt.Errorf("timestamp dimension requires time_bucket specification")
        }
        if err := v.validateTimeBucket(dim.TimeBucket, dim.CustomInterval); err != nil {
            return fmt.Errorf("time bucket validation failed: %w", err)
        }
    case DimensionCustomField:
        // Custom dimension requires field specification
        if dim.Field == "" {
            return fmt.Errorf("custom dimension requires field specification")
        }
        if !v.allowedCustomFields[dim.Field] {
            return fmt.Errorf("custom field '%s' not allowed", dim.Field)
        }
    default:
        return fmt.Errorf("unsupported dimension type: %s", dim.Type)
    }

    // Validate alias
    if dim.Alias != "" {
        if !isValidIdentifier(dim.Alias) {
            return fmt.Errorf("invalid dimension alias: %s", dim.Alias)
        }
    }

    // Validate sort order
    if dim.SortOrder != "" && dim.SortOrder != "ASC" && dim.SortOrder != "DESC" {
        return fmt.Errorf("invalid sort order: %s (must be ASC or DESC)", dim.SortOrder)
    }

    // Validate limit
    if dim.Limit < 0 || dim.Limit > 10000 {
        return fmt.Errorf("dimension limit must be between 0 and 10000, got: %d", dim.Limit)
    }

    return nil
}

// validateFunction validates a single aggregation function
func (v *AggregationDimensionValidator) validateFunction(fn AggregationFunction) error {
    // Validate function type
    switch fn.Type {
    case FunctionCount:
        // Count function doesn't require field specification
    case FunctionSum, FunctionAvg, FunctionMin, FunctionMax, FunctionStdDev:
        // Numeric functions require field specification
        if fn.Field == "" {
            return fmt.Errorf("%s function requires field specification", fn.Type)
        }
        if !isNumericField(fn.Field) {
            return fmt.Errorf("%s function requires numeric field, got: %s", fn.Type, fn.Field)
        }
    case FunctionPercentile50, FunctionPercentile95, FunctionPercentile99:
        // Percentile functions require numeric field
        if fn.Field == "" {
            return fmt.Errorf("%s function requires field specification", fn.Type)
        }
        if !isNumericField(fn.Field) {
            return fmt.Errorf("%s function requires numeric field, got: %s", fn.Type, fn.Field)
        }
    case FunctionDistinctCount:
        // Distinct count requires field specification
        if fn.Field == "" {
            return fmt.Errorf("distinct_count function requires field specification")
        }
    case FunctionRate:
        // Rate function requires time-based aggregation
        if fn.Field == "" {
            fn.Field = "count(*)" // Default to count rate
        }
    case FunctionCustom:
        // Custom function requires expression
        if fn.CustomExpression == "" {
            return fmt.Errorf("custom function requires expression specification")
        }
        if err := v.validateCustomExpression(fn.CustomExpression); err != nil {
            return fmt.Errorf("custom expression validation failed: %w", err)
        }
    default:
        return fmt.Errorf("unsupported function type: %s", fn.Type)
    }

    // Validate alias
    if fn.Alias != "" {
        if !isValidIdentifier(fn.Alias) {
            return fmt.Errorf("invalid function alias: %s", fn.Alias)
        }
    }

    return nil
}

// validateTimeBucket validates time bucket configuration
func (v *AggregationDimensionValidator) validateTimeBucket(bucket TimeBucketInterval, customInterval time.Duration) error {
    switch bucket {
    case IntervalMinute, Interval5Minutes, Interval15Minutes,
         IntervalHour, Interval6Hours, Interval12Hours,
         IntervalDay, IntervalWeek:
        // Standard intervals are valid
        return nil
    case IntervalCustom:
        // Custom interval requires duration specification
        if customInterval <= 0 {
            return fmt.Errorf("custom time bucket requires positive duration")
        }
        if customInterval < time.Second {
            return fmt.Errorf("custom time bucket minimum is 1 second")
        }
        if customInterval > 30*24*time.Hour {
            return fmt.Errorf("custom time bucket maximum is 30 days")
        }
        return nil
    default:
        return fmt.Errorf("unsupported time bucket interval: %s", bucket)
    }
}

// validateTimeDimensions ensures consistent time handling
func (v *AggregationDimensionValidator) validateTimeDimensions(req *AggregationRequest) error {
    hasTimeDimension := false
    for _, dim := range req.Dimensions {
        if dim.Type == DimensionTimestamp {
            hasTimeDimension = true
            break
        }
    }

    // If using rate function, time dimension is required
    for _, fn := range req.Functions {
        if fn.Type == FunctionRate && !hasTimeDimension {
            return fmt.Errorf("rate function requires time dimension for proper calculation")
        }
    }

    // Validate time range for time-based aggregations
    if hasTimeDimension && (req.StartTime == nil || req.EndTime == nil) {
        return fmt.Errorf("time dimensions require start_time and end_time specification")
    }

    return nil
}

// validateDimensionCardinality estimates and validates dimension cardinality
func (v *AggregationDimensionValidator) validateDimensionCardinality(req *AggregationRequest) error {
    estimatedCardinality := 1

    for _, dim := range req.Dimensions {
        if limit, exists := v.cardinalityLimits[dim.Type]; exists {
            estimatedCardinality *= limit
        } else {
            estimatedCardinality *= 1000 // Default estimate for custom dimensions
        }

        // Early exit if estimated cardinality is too high
        if estimatedCardinality > 1000000 { // 1M combinations limit
            return fmt.Errorf("estimated result cardinality too high (%d combinations), consider reducing dimensions or adding limits",
                estimatedCardinality)
        }
    }

    return nil
}

// calculateRequestComplexity provides aggregation complexity scoring
func (v *AggregationDimensionValidator) calculateRequestComplexity(req *AggregationRequest) float64 {
    complexity := 0.0

    // Dimension complexity
    for _, dim := range req.Dimensions {
        switch dim.Type {
        case DimensionSeverity, DimensionDataset:
            complexity += 1.0 // Low cardinality dimensions
        case DimensionNamespace, DimensionHostName:
            complexity += 2.0 // Medium cardinality dimensions
        case DimensionPodName, DimensionContainerName:
            complexity += 3.0 // High cardinality dimensions
        case DimensionTimestamp:
            complexity += 1.5 // Time dimensions are moderately complex
        case DimensionCustomField:
            complexity += 2.5 // Custom fields unknown cardinality
        }
    }

    // Function complexity
    for _, fn := range req.Functions {
        switch fn.Type {
        case FunctionCount:
            complexity += 0.5 // Simple count operation
        case FunctionSum, FunctionAvg, FunctionMin, FunctionMax:
            complexity += 1.0 // Basic aggregation functions
        case FunctionPercentile50, FunctionPercentile95, FunctionPercentile99:
            complexity += 2.0 // Percentile calculations are more expensive
        case FunctionDistinctCount:
            complexity += 3.0 // Distinct count can be expensive
        case FunctionRate, FunctionStdDev:
            complexity += 1.5 // Moderate complexity calculations
        case FunctionCustom:
            complexity += 4.0 // Custom expressions unknown complexity
        }
    }

    // Multi-dimensional complexity multiplier
    if len(req.Dimensions) > 1 {
        complexity *= 1.5 + float64(len(req.Dimensions)-1)*0.3
    }

    // Filter complexity (inherited from Epic 2)
    if req.ContentSearch != nil && len(req.ContentSearch.Filters) > 0 {
        complexity += float64(len(req.ContentSearch.Filters)) * 0.8
    }
    if req.K8sFilters != nil {
        complexity += 1.2 // K8s filtering adds some complexity
    }

    return complexity
}

// isValidIdentifier validates SQL identifier naming
func isValidIdentifier(name string) bool {
    if name == "" {
        return false
    }

    // Must start with letter or underscore
    if !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z') || name[0] == '_') {
        return false
    }

    // Check remaining characters
    for _, r := range name[1:] {
        if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
            return false
        }
    }

    return len(name) <= 64 // Reasonable length limit
}

// isNumericField determines if a field contains numeric data
func isNumericField(field string) bool {
    numericFields := map[string]bool{
        "message_length":   true,
        "processing_time":  true,
        "timestamp":        true,
        "error_code":       true,
        "response_time":    true,
        "bytes_processed":  true,
        "memory_usage":     true,
        "cpu_usage":        true,
    }
    return numericFields[field]
}

// validateCustomExpression validates custom aggregation expressions
func (v *AggregationDimensionValidator) validateCustomExpression(expr string) error {
    // Basic SQL injection protection
    dangerousPatterns := []string{
        "DROP", "DELETE", "INSERT", "UPDATE", "CREATE", "ALTER",
        "EXEC", "EXECUTE", "UNION", "DECLARE", "--", "/*", "*/",
        "xp_", "sp_", "proc", "procedure",
    }

    upperExpr := strings.ToUpper(expr)
    for _, pattern := range dangerousPatterns {
        if strings.Contains(upperExpr, pattern) {
            return fmt.Errorf("custom expression contains potentially dangerous pattern: %s", pattern)
        }
    }

    // Length validation
    if len(expr) > 500 {
        return fmt.Errorf("custom expression too long (%d chars), maximum 500", len(expr))
    }

    // UTF-8 validation
    if !utf8.ValidString(expr) {
        return fmt.Errorf("custom expression contains invalid UTF-8 characters")
    }

    return nil
}
```

### Enhanced Service Layer with Aggregation Integration

**Enhanced service layer to integrate aggregation with Epic 2 filtering foundation:**

```go
// Enhanced LogQueryService with comprehensive aggregation support
func (s *Service) QueryLogsWithAggregation(ctx context.Context, req *request.AggregationRequest) (*response.AggregationResponse, error) {
    // Validate aggregation request
    validator := NewAggregationDimensionValidator()
    if err := validator.ValidateAggregationRequest(req); err != nil {
        return nil, fmt.Errorf("aggregation validation failed: %w", err)
    }

    // Check cache first (if enabled)
    cacheKey := s.generateAggregationCacheKey(req)
    if req.UseCache && s.cache != nil {
        if cached, found := s.cache.Get(cacheKey); found {
            if aggResp, ok := cached.(*response.AggregationResponse); ok {
                klog.InfoS("Aggregation cache hit", "dataset", req.Dataset, "cache_key", cacheKey)
                return aggResp, nil
            }
        }
    }

    // Build aggregation query
    query, args, err := s.buildAggregationQuery(req)
    if err != nil {
        return nil, fmt.Errorf("failed to build aggregation query: %w", err)
    }

    // Execute aggregation query
    startTime := time.Now()
    result, err := s.repository.ExecuteAggregationQuery(ctx, query, args)
    if err != nil {
        return nil, fmt.Errorf("aggregation query execution failed: %w", err)
    }
    queryDuration := time.Since(startTime)

    // Parse aggregation results
    response, err := s.parseAggregationResults(result, req)
    if err != nil {
        return nil, fmt.Errorf("aggregation result parsing failed: %w", err)
    }

    // Add execution metadata
    response.Metadata = &response.AggregationMetadata{
        QueryDurationMs:      queryDuration.Milliseconds(),
        QueryComplexity:      validator.calculateRequestComplexity(req),
        DimensionCount:       len(req.Dimensions),
        FunctionCount:        len(req.Functions),
        ResultSetSize:        len(response.Results),
        CacheKey:            cacheKey,
        FromCache:           false,
        EstimatedCardinality: s.estimateResultCardinality(req),
    }

    // Store in cache (if enabled)
    if req.UseCache && s.cache != nil && req.CacheTTL > 0 {
        s.cache.Set(cacheKey, response, req.CacheTTL)
        klog.InfoS("Aggregation result cached", "dataset", req.Dataset, "cache_key", cacheKey, "ttl", req.CacheTTL)
    }

    // Record performance metrics
    s.metrics.RecordAggregationQuery(req.Dataset, queryDuration, req, len(response.Results))

    klog.InfoS("Aggregation query completed",
        "dataset", req.Dataset,
        "dimensions", len(req.Dimensions),
        "functions", len(req.Functions),
        "result_count", len(response.Results),
        "duration_ms", queryDuration.Milliseconds(),
        "complexity", response.Metadata.QueryComplexity)

    return response, nil
}

// buildAggregationQuery creates optimized ClickHouse aggregation query
func (s *Service) buildAggregationQuery(req *request.AggregationRequest) (string, []interface{}, error) {
    var whereConditions []string
    var args []interface{}

    // Build base filters (from Epic 2 foundation)
    baseFilters, baseArgs, err := s.buildBaseFilters(req)
    if err != nil {
        return "", nil, fmt.Errorf("failed to build base filters: %w", err)
    }
    whereConditions = append(whereConditions, baseFilters...)
    args = append(args, baseArgs...)

    // Build SELECT clause with aggregation functions
    selectClause, err := s.buildAggregationSelectClause(req)
    if err != nil {
        return "", nil, fmt.Errorf("failed to build aggregation SELECT clause: %w", err)
    }

    // Build GROUP BY clause
    groupByClause, err := s.buildAggregationGroupByClause(req)
    if err != nil {
        return "", nil, fmt.Errorf("failed to build aggregation GROUP BY clause: %w", err)
    }

    // Build HAVING clause (post-aggregation filtering)
    havingClause := ""
    if req.Having != "" {
        havingClause = fmt.Sprintf("HAVING %s", req.Having)
    }

    // Build ORDER BY clause
    orderByClause, err := s.buildAggregationOrderByClause(req)
    if err != nil {
        return "", nil, fmt.Errorf("failed to build aggregation ORDER BY clause: %w", err)
    }

    // Build LIMIT clause
    limitClause := ""
    if req.Limit > 0 {
        limitClause = fmt.Sprintf("LIMIT %d", req.Limit)
        if req.Offset > 0 {
            limitClause += fmt.Sprintf(" OFFSET %d", req.Offset)
        }
    }

    // Assemble final query
    query := fmt.Sprintf(`
        %s
        FROM logs
        WHERE %s
        %s
        %s
        %s
        %s
    `,
        selectClause,
        strings.Join(whereConditions, " AND "),
        groupByClause,
        havingClause,
        orderByClause,
        limitClause)

    return strings.TrimSpace(query), args, nil
}

// buildBaseFilters creates base filtering conditions from Epic 2 foundation
func (s *Service) buildBaseFilters(req *request.AggregationRequest) ([]string, []interface{}, error) {
    var conditions []string
    var args []interface{}

    // Dataset filter (from Story 2.1)
    if req.Dataset != "" {
        conditions = append(conditions, "dataset = ?")
        args = append(args, req.Dataset)
    }

    // Time range filters (from Story 2.2)
    if req.StartTime != nil {
        conditions = append(conditions, "timestamp >= ?")
        args = append(args, *req.StartTime)
    }
    if req.EndTime != nil {
        conditions = append(conditions, "timestamp <= ?")
        args = append(args, *req.EndTime)
    }

    // K8s metadata filters (from Story 2.3)
    if req.K8sFilters != nil {
        k8sConditions, k8sArgs, err := s.buildK8sFilterConditions(req.K8sFilters)
        if err != nil {
            return nil, nil, fmt.Errorf("failed to build K8s filter conditions: %w", err)
        }
        conditions = append(conditions, k8sConditions...)
        args = append(args, k8sArgs...)
    }

    // Content search filters (from Story 2.4)
    if req.ContentSearch != nil {
        contentConditions, contentArgs, err := s.buildContentSearchConditions(req.ContentSearch)
        if err != nil {
            return nil, nil, fmt.Errorf("failed to build content search conditions: %w", err)
        }
        conditions = append(conditions, contentConditions...)
        args = append(args, contentArgs...)
    }

    return conditions, args, nil
}

// buildAggregationSelectClause creates SELECT clause with aggregation functions
func (s *Service) buildAggregationSelectClause(req *request.AggregationRequest) (string, error) {
    var selectParts []string

    // Add dimension fields
    for _, dim := range req.Dimensions {
        dimField, err := s.buildDimensionField(dim)
        if err != nil {
            return "", fmt.Errorf("failed to build dimension field: %w", err)
        }
        selectParts = append(selectParts, dimField)
    }

    // Add aggregation functions
    for _, fn := range req.Functions {
        fnField, err := s.buildAggregationFunction(fn)
        if err != nil {
            return "", fmt.Errorf("failed to build aggregation function: %w", err)
        }
        selectParts = append(selectParts, fnField)
    }

    return "SELECT " + strings.Join(selectParts, ",\n        "), nil
}

// buildDimensionField creates dimension field expression
func (s *Service) buildDimensionField(dim AggregationDimension) (string, error) {
    var fieldExpr string
    var alias string

    switch dim.Type {
    case DimensionSeverity:
        fieldExpr = "severity"
        alias = "severity"
    case DimensionNamespace:
        fieldExpr = "k8s_namespace_name"
        alias = "namespace"
    case DimensionPodName:
        fieldExpr = "k8s_pod_name"
        alias = "pod_name"
    case DimensionNodeName:
        fieldExpr = "k8s_node_name"
        alias = "node_name"
    case DimensionHostName:
        fieldExpr = "host_name"
        alias = "host_name"
    case DimensionContainerName:
        fieldExpr = "container_name"
        alias = "container_name"
    case DimensionDataset:
        fieldExpr = "dataset"
        alias = "dataset"
    case DimensionTimestamp:
        bucketExpr, err := s.buildTimeBucketExpression(dim.TimeBucket, dim.CustomInterval)
        if err != nil {
            return "", fmt.Errorf("failed to build time bucket expression: %w", err)
        }
        fieldExpr = bucketExpr
        alias = "time_bucket"
    case DimensionCustomField:
        // Validate custom field exists and is safe
        if !s.isValidCustomField(dim.Field) {
            return "", fmt.Errorf("invalid custom field: %s", dim.Field)
        }
        fieldExpr = dim.Field
        alias = dim.Field
    default:
        return "", fmt.Errorf("unsupported dimension type: %s", dim.Type)
    }

    // Use custom alias if provided
    if dim.Alias != "" {
        alias = dim.Alias
    }

    return fmt.Sprintf("%s AS %s", fieldExpr, alias), nil
}

// buildTimeBucketExpression creates time bucketing expression
func (s *Service) buildTimeBucketExpression(bucket TimeBucketInterval, customInterval time.Duration) (string, error) {
    switch bucket {
    case IntervalMinute:
        return "toStartOfMinute(timestamp)", nil
    case Interval5Minutes:
        return "toStartOfFiveMinutes(timestamp)", nil
    case Interval15Minutes:
        return "toStartOfFifteenMinutes(timestamp)", nil
    case IntervalHour:
        return "toStartOfHour(timestamp)", nil
    case Interval6Hours:
        return "toDateTime64(intDiv(toUInt64(timestamp), 21600) * 21600, 9)", nil
    case Interval12Hours:
        return "toDateTime64(intDiv(toUInt64(timestamp), 43200) * 43200, 9)", nil
    case IntervalDay:
        return "toStartOfDay(timestamp)", nil
    case IntervalWeek:
        return "toStartOfWeek(timestamp)", nil
    case IntervalCustom:
        if customInterval <= 0 {
            return "", fmt.Errorf("custom interval must be positive")
        }
        seconds := int64(customInterval.Seconds())
        return fmt.Sprintf("toDateTime64(intDiv(toUInt64(timestamp), %d) * %d, 9)", seconds, seconds), nil
    default:
        return "", fmt.Errorf("unsupported time bucket interval: %s", bucket)
    }
}

// buildAggregationFunction creates aggregation function expression
func (s *Service) buildAggregationFunction(fn AggregationFunction) (string, error) {
    var fnExpr string
    var alias string

    switch fn.Type {
    case FunctionCount:
        fnExpr = "count(*)"
        alias = "count"
    case FunctionSum:
        if fn.Field == "" {
            return "", fmt.Errorf("sum function requires field specification")
        }
        fnExpr = fmt.Sprintf("sum(%s)", fn.Field)
        alias = fmt.Sprintf("sum_%s", fn.Field)
    case FunctionAvg:
        if fn.Field == "" {
            return "", fmt.Errorf("avg function requires field specification")
        }
        fnExpr = fmt.Sprintf("avg(%s)", fn.Field)
        alias = fmt.Sprintf("avg_%s", fn.Field)
    case FunctionMin:
        if fn.Field == "" {
            return "", fmt.Errorf("min function requires field specification")
        }
        fnExpr = fmt.Sprintf("min(%s)", fn.Field)
        alias = fmt.Sprintf("min_%s", fn.Field)
    case FunctionMax:
        if fn.Field == "" {
            return "", fmt.Errorf("max function requires field specification")
        }
        fnExpr = fmt.Sprintf("max(%s)", fn.Field)
        alias = fmt.Sprintf("max_%s", fn.Field)
    case FunctionPercentile50:
        if fn.Field == "" {
            return "", fmt.Errorf("percentile function requires field specification")
        }
        fnExpr = fmt.Sprintf("quantile(0.5)(%s)", fn.Field)
        alias = fmt.Sprintf("p50_%s", fn.Field)
    case FunctionPercentile95:
        if fn.Field == "" {
            return "", fmt.Errorf("percentile function requires field specification")
        }
        fnExpr = fmt.Sprintf("quantile(0.95)(%s)", fn.Field)
        alias = fmt.Sprintf("p95_%s", fn.Field)
    case FunctionPercentile99:
        if fn.Field == "" {
            return "", fmt.Errorf("percentile function requires field specification")
        }
        fnExpr = fmt.Sprintf("quantile(0.99)(%s)", fn.Field)
        alias = fmt.Sprintf("p99_%s", fn.Field)
    case FunctionDistinctCount:
        if fn.Field == "" {
            return "", fmt.Errorf("distinct_count function requires field specification")
        }
        fnExpr = fmt.Sprintf("uniqExact(%s)", fn.Field)
        alias = fmt.Sprintf("distinct_%s", fn.Field)
    case FunctionRate:
        // Rate calculation requires time dimension
        timeField := s.extractTimeFieldFromDimensions(fn)
        if timeField == "" {
            timeField = "timestamp"
        }
        if fn.Field == "" {
            fnExpr = fmt.Sprintf("count(*) / (max(%s) - min(%s))", timeField, timeField)
        } else {
            fnExpr = fmt.Sprintf("sum(%s) / (max(%s) - min(%s))", fn.Field, timeField, timeField)
        }
        alias = fmt.Sprintf("rate_%s", strings.ReplaceAll(fn.Field, "*", "all"))
    case FunctionStdDev:
        if fn.Field == "" {
            return "", fmt.Errorf("stddev function requires field specification")
        }
        fnExpr = fmt.Sprintf("stddevPop(%s)", fn.Field)
        alias = fmt.Sprintf("stddev_%s", fn.Field)
    case FunctionCustom:
        if fn.CustomExpression == "" {
            return "", fmt.Errorf("custom function requires expression")
        }
        fnExpr = fn.CustomExpression
        alias = "custom_result"
    default:
        return "", fmt.Errorf("unsupported function type: %s", fn.Type)
    }

    // Use custom alias if provided
    if fn.Alias != "" {
        alias = fn.Alias
    }

    return fmt.Sprintf("%s AS %s", fnExpr, alias), nil
}

// buildAggregationGroupByClause creates GROUP BY clause
func (s *Service) buildAggregationGroupByClause(req *request.AggregationRequest) (string, error) {
    if len(req.Dimensions) == 0 {
        return "", nil
    }

    var groupByFields []string
    for i, dim := range req.Dimensions {
        // Use dimension alias or field position
        if dim.Alias != "" {
            groupByFields = append(groupByFields, dim.Alias)
        } else {
            groupByFields = append(groupByFields, fmt.Sprintf("%d", i+1))
        }
    }

    return "GROUP BY " + strings.Join(groupByFields, ", "), nil
}

// buildAggregationOrderByClause creates ORDER BY clause
func (s *Service) buildAggregationOrderByClause(req *request.AggregationRequest) (string, error) {
    var orderByParts []string

    // Add explicit order by fields
    for _, orderField := range req.OrderBy {
        orderByParts = append(orderByParts, orderField)
    }

    // Add dimension-specific ordering
    for _, dim := range req.Dimensions {
        if dim.SortOrder != "" {
            field := dim.Alias
            if field == "" {
                field = s.getDimensionDefaultAlias(dim.Type)
            }
            orderByParts = append(orderByParts, fmt.Sprintf("%s %s", field, dim.SortOrder))
        }
    }

    // Default ordering for time dimensions
    if s.hasTimeDimension(req.Dimensions) && len(orderByParts) == 0 {
        orderByParts = append(orderByParts, "time_bucket DESC")
    }

    if len(orderByParts) > 0 {
        return "ORDER BY " + strings.Join(orderByParts, ", "), nil
    }

    return "", nil
}
```

### Aggregation Result Caching and Performance Optimization

**Comprehensive caching and performance optimization for aggregation queries:**

```go
// AggregationCache provides intelligent caching for aggregation results
type AggregationCache struct {
    cache     map[string]*CachedAggregation
    mutex     sync.RWMutex
    maxSize   int
    ttlMap    map[string]time.Time
    hitRate   *prometheus.CounterVec
    missRate  *prometheus.CounterVec
}

type CachedAggregation struct {
    Result      *response.AggregationResponse
    GeneratedAt time.Time
    TTL         time.Duration
    AccessCount int64
    LastAccess  time.Time
    CacheKey    string
    DataHash    string // Hash of underlying data for invalidation
}

func NewAggregationCache(maxSize int) *AggregationCache {
    cache := &AggregationCache{
        cache:   make(map[string]*CachedAggregation),
        maxSize: maxSize,
        ttlMap:  make(map[string]time.Time),
        hitRate: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_aggregation_cache_hits_total",
                Help: "Total number of aggregation cache hits",
            },
            []string{"dataset", "cache_type"},
        ),
        missRate: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_aggregation_cache_misses_total",
                Help: "Total number of aggregation cache misses",
            },
            []string{"dataset", "cache_type"},
        ),
    }

    // Start cache maintenance goroutine
    go cache.maintenanceLoop()

    return cache
}

// generateAggregationCacheKey creates cache key from aggregation request
func (s *Service) generateAggregationCacheKey(req *request.AggregationRequest) string {
    var keyParts []string

    // Dataset
    keyParts = append(keyParts, "dataset:"+req.Dataset)

    // Time range
    if req.StartTime != nil && req.EndTime != nil {
        keyParts = append(keyParts, fmt.Sprintf("time:%d-%d", req.StartTime.Unix(), req.EndTime.Unix()))
    }

    // Dimensions
    var dimParts []string
    for _, dim := range req.Dimensions {
        dimKey := string(dim.Type)
        if dim.Field != "" {
            dimKey += ":" + dim.Field
        }
        if dim.TimeBucket != "" {
            dimKey += ":" + string(dim.TimeBucket)
        }
        if dim.CustomInterval > 0 {
            dimKey += fmt.Sprintf(":%d", dim.CustomInterval.Seconds())
        }
        dimParts = append(dimParts, dimKey)
    }
    if len(dimParts) > 0 {
        keyParts = append(keyParts, "dims:"+strings.Join(dimParts, ","))
    }

    // Functions
    var funcParts []string
    for _, fn := range req.Functions {
        funcKey := string(fn.Type)
        if fn.Field != "" {
            funcKey += ":" + fn.Field
        }
        if fn.CustomExpression != "" {
            funcKey += ":custom"
        }
        funcParts = append(funcParts, funcKey)
    }
    if len(funcParts) > 0 {
        keyParts = append(keyParts, "funcs:"+strings.Join(funcParts, ","))
    }

    // Filters (simplified representation)
    if req.K8sFilters != nil {
        keyParts = append(keyParts, "k8s:filtered")
    }
    if req.ContentSearch != nil && len(req.ContentSearch.Filters) > 0 {
        keyParts = append(keyParts, fmt.Sprintf("content:%d", len(req.ContentSearch.Filters)))
    }

    // Pagination and ordering
    if req.Limit > 0 {
        keyParts = append(keyParts, fmt.Sprintf("limit:%d:%d", req.Limit, req.Offset))
    }
    if len(req.OrderBy) > 0 {
        keyParts = append(keyParts, "order:"+strings.Join(req.OrderBy, ","))
    }

    // Generate final cache key
    fullKey := strings.Join(keyParts, "|")

    // Hash for reasonable key length
    hasher := sha256.New()
    hasher.Write([]byte(fullKey))
    return fmt.Sprintf("agg:%x", hasher.Sum(nil)[:16])
}

// Get retrieves aggregation result from cache
func (c *AggregationCache) Get(key string) (*response.AggregationResponse, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()

    cached, exists := c.cache[key]
    if !exists {
        return nil, false
    }

    // Check TTL
    if time.Since(cached.GeneratedAt) > cached.TTL {
        // Expired - clean up in background
        go c.removeExpired(key)
        return nil, false
    }

    // Update access statistics
    atomic.AddInt64(&cached.AccessCount, 1)
    cached.LastAccess = time.Now()

    return cached.Result, true
}

// Set stores aggregation result in cache
func (c *AggregationCache) Set(key string, result *response.AggregationResponse, ttl time.Duration) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    // Evict if cache is full
    if len(c.cache) >= c.maxSize {
        c.evictLeastRecentlyUsed()
    }

    cached := &CachedAggregation{
        Result:      result,
        GeneratedAt: time.Now(),
        TTL:         ttl,
        AccessCount: 1,
        LastAccess:  time.Now(),
        CacheKey:    key,
        DataHash:    c.generateDataHash(result),
    }

    c.cache[key] = cached
    c.ttlMap[key] = cached.GeneratedAt.Add(ttl)
}

// maintenanceLoop handles cache cleanup and optimization
func (c *AggregationCache) maintenanceLoop() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        c.cleanupExpired()
        c.optimizeCache()
    }
}

// cleanupExpired removes expired cache entries
func (c *AggregationCache) cleanupExpired() {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    now := time.Now()
    var expiredKeys []string

    for key, expiration := range c.ttlMap {
        if now.After(expiration) {
            expiredKeys = append(expiredKeys, key)
        }
    }

    for _, key := range expiredKeys {
        delete(c.cache, key)
        delete(c.ttlMap, key)
    }

    if len(expiredKeys) > 0 {
        klog.InfoS("Cache cleanup completed", "expired_entries", len(expiredKeys), "total_entries", len(c.cache))
    }
}

// AggregationPerformanceMetrics tracks aggregation performance
type AggregationPerformanceMetrics struct {
    queryDuration       *prometheus.HistogramVec
    queryComplexity     *prometheus.HistogramVec
    resultCardinality   *prometheus.HistogramVec
    cacheHitRate       *prometheus.GaugeVec
    slowQueries        *prometheus.CounterVec
    dimensionUsage     *prometheus.CounterVec
    functionUsage      *prometheus.CounterVec
}

func NewAggregationPerformanceMetrics() *AggregationPerformanceMetrics {
    return &AggregationPerformanceMetrics{
        queryDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_aggregation_duration_seconds",
                Help:    "Duration of aggregation queries",
                Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
            },
            []string{"dataset", "complexity_level", "dimension_count", "function_count"},
        ),
        queryComplexity: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_aggregation_complexity",
                Help:    "Complexity score of aggregation queries",
                Buckets: []float64{1, 5, 10, 25, 50, 100},
            },
            []string{"dataset"},
        ),
        resultCardinality: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_aggregation_result_cardinality",
                Help:    "Number of result rows from aggregation queries",
                Buckets: []float64{1, 10, 100, 1000, 10000, 100000},
            },
            []string{"dataset", "dimension_count"},
        ),
        cacheHitRate: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "edge_logs_aggregation_cache_hit_rate",
                Help: "Cache hit rate for aggregation queries",
            },
            []string{"dataset"},
        ),
        slowQueries: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_slow_aggregation_queries_total",
                Help: "Number of slow aggregation queries (>2s)",
            },
            []string{"dataset", "reason"},
        ),
        dimensionUsage: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_aggregation_dimension_usage_total",
                Help: "Usage count of aggregation dimensions",
            },
            []string{"dimension_type"},
        ),
        functionUsage: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_aggregation_function_usage_total",
                Help: "Usage count of aggregation functions",
            },
            []string{"function_type"},
        ),
    }
}

// RecordAggregationQuery records metrics for aggregation queries
func (m *AggregationPerformanceMetrics) RecordAggregationQuery(dataset string, duration time.Duration,
    req *request.AggregationRequest, resultCount int) {

    // Categorize complexity
    complexity := calculateAggregationComplexity(req)
    complexityLevel := categorizeComplexity(complexity)

    // Record duration with context
    m.queryDuration.With(prometheus.Labels{
        "dataset":        dataset,
        "complexity_level": complexityLevel,
        "dimension_count": fmt.Sprintf("%d", len(req.Dimensions)),
        "function_count":  fmt.Sprintf("%d", len(req.Functions)),
    }).Observe(duration.Seconds())

    // Record complexity
    m.queryComplexity.With(prometheus.Labels{
        "dataset": dataset,
    }).Observe(complexity)

    // Record result cardinality
    m.resultCardinality.With(prometheus.Labels{
        "dataset":         dataset,
        "dimension_count": fmt.Sprintf("%d", len(req.Dimensions)),
    }).Observe(float64(resultCount))

    // Track dimension usage
    for _, dim := range req.Dimensions {
        m.dimensionUsage.With(prometheus.Labels{
            "dimension_type": string(dim.Type),
        }).Inc()
    }

    // Track function usage
    for _, fn := range req.Functions {
        m.functionUsage.With(prometheus.Labels{
            "function_type": string(fn.Type),
        }).Inc()
    }

    // Track slow queries
    if duration > 2*time.Second {
        reason := determineSlowAggregationReason(duration, complexity, len(req.Dimensions), len(req.Functions))
        m.slowQueries.With(prometheus.Labels{
            "dataset": dataset,
            "reason":  reason,
        }).Inc()
    }
}
```

### Project Structure Notes

**File organization enhancing existing Epic 2 foundation to support Epic 3 aggregation capabilities:**

```
pkg/service/query/
├── service.go                        # Enhanced with aggregation support (modify existing)
├── aggregation_validator.go          # Aggregation dimension and function validation (new)
├── aggregation_builder.go            # Aggregation query building logic (new)
├── aggregation_cache.go               # Aggregation result caching (new)
├── content_search_validator.go        # Existing from Story 2.4
├── k8s_validator.go                   # Existing from Story 2.3
├── time_validator.go                  # Existing from Story 2.2
├── dataset_validator.go               # Existing from Story 2.1
└── service_test.go                   # Enhanced with aggregation tests (modify existing)

pkg/repository/clickhouse/
├── repository.go                     # Enhanced with aggregation query support (modify existing)
├── aggregation_queries.go            # Aggregation-specific query patterns (new)
├── aggregation_optimization.go       # Aggregation performance optimization (new)
├── content_search_queries.go         # Existing from Story 2.4
├── k8s_queries.go                    # Existing from Story 2.3
├── time_queries.go                   # Existing from Story 2.2
└── repository_test.go                # Enhanced with aggregation tests (modify existing)

pkg/oapis/log/v1alpha1/
├── handler.go                        # Enhanced with aggregation endpoints (modify existing)
├── aggregation_handler.go            # Dedicated aggregation endpoint handlers (new)
├── aggregation_errors.go             # Aggregation-specific error handling (new)
├── aggregation_metrics.go            # Aggregation performance metrics (new)
├── content_search_errors.go          # Existing from Story 2.4
├── k8s_errors.go                     # Existing from Story 2.3
└── handler_test.go                   # Enhanced with aggregation tests (modify existing)

pkg/model/request/
├── log.go                           # Enhanced with aggregation request fields (modify existing)
└── aggregation.go                   # Dedicated aggregation request models (new)

pkg/model/response/
├── log.go                           # Existing log response models
└── aggregation.go                   # Aggregation response models with visualization support (new)
```

### References

- [Source: _bmad-output/epics.md#Story 3.1] - Complete user story and acceptance criteria
- [Source: _bmad-output/architecture.md#聚合分析] - Aggregation requirements with ClickHouse GROUP BY optimization
- [Source: _bmad-output/2-4-content-based-log-search.md] - Epic 2 content search foundation
- [Source: _bmad-output/2-3-namespace-and-pod-filtering.md] - Epic 2 K8s filtering foundation
- [Source: _bmad-output/2-2-time-range-filtering-with-millisecond-precision.md] - Epic 2 time filtering foundation
- [Source: _bmad-output/2-1-dataset-based-query-routing.md] - Epic 2 dataset routing foundation
- [Source: sqlscripts/clickhouse/01_tables.sql] - ClickHouse schema for aggregation optimization
- [Source: pkg/model/request/log.go] - Request models to enhance with aggregation fields
- [Source: pkg/service/query/service.go] - Service layer to enhance with aggregation support
- [Source: pkg/repository/clickhouse/repository.go] - Repository layer for aggregation queries

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-20250514

### Debug Log References

Initiating Epic 3: Advanced Query and Analytics by implementing comprehensive log aggregation capabilities on top of the complete filtering foundation established in Epic 2. This story leverages dataset routing (2-1), time filtering (2-2), K8s filtering (2-3), and content search (2-4) to provide powerful multi-dimensional aggregation with time-based bucketing, multiple aggregation functions (count, sum, avg, min, max, percentiles), and intelligent result caching for sub-2 second performance on large-scale edge computing log datasets.

### Completion Notes List

Story 3-1 launches Epic 3 by implementing advanced log aggregation by dimensions, building upon the comprehensive filtering foundation from Epic 2. Provides multi-dimensional aggregation capabilities with flexible dimension combinations (severity, namespace, host, time buckets), comprehensive aggregation functions (count, sum, avg, min, max, percentiles, rates), time-based bucketing for trend analysis, performance optimization through ClickHouse GROUP BY optimization, intelligent result caching, and seamless integration with all Epic 2 filtering capabilities (dataset routing, time filtering, K8s metadata filtering, content search) to deliver powerful analytics for edge computing operational insights.

### File List

Primary files to be enhanced/created:
- pkg/service/query/service.go (enhance with aggregation support)
- pkg/service/query/aggregation_validator.go (new)
- pkg/service/query/aggregation_builder.go (new)
- pkg/service/query/aggregation_cache.go (new)
- pkg/repository/clickhouse/repository.go (enhance with aggregation query support)
- pkg/repository/clickhouse/aggregation_queries.go (new)
- pkg/repository/clickhouse/aggregation_optimization.go (new)
- pkg/oapis/log/v1alpha1/handler.go (enhance with aggregation endpoints)
- pkg/oapis/log/v1alpha1/aggregation_handler.go (new)
- pkg/oapis/log/v1alpha1/aggregation_errors.go (new)
- pkg/oapis/log/v1alpha1/aggregation_metrics.go (new)
- pkg/model/request/log.go (enhance with aggregation request fields)
- pkg/model/request/aggregation.go (new)
- pkg/model/response/aggregation.go (new)
- pkg/service/query/service_test.go (enhance with aggregation tests)
- pkg/repository/clickhouse/repository_test.go (enhance with aggregation tests)
- pkg/oapis/log/v1alpha1/handler_test.go (enhance with aggregation tests)