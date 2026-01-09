# Story 2.4: Content-based log search

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want to search log content using keywords and advanced search patterns,
So that I can find specific log messages or error patterns across edge computing deployments with powerful full-text search capabilities and highlighted results.

## Acceptance Criteria

**Given** Kubernetes metadata filtering is implemented (Story 2-3 completed)
**When** I specify search parameters for content filtering
**Then** Log content is searched using ClickHouse's advanced full-text search capabilities
**And** Multiple search patterns are supported (exact, case-insensitive, regex, wildcard)
**And** ClickHouse text indexing with tokenbf_v1 is leveraged for efficient content matching
**And** Multiple keywords can be searched with complex boolean logic (AND/OR/NOT operators)
**And** Search results highlight matched terms for easy identification
**And** Search performance is optimized with relevance scoring for large datasets
**And** Content search integrates seamlessly with existing dataset, time, and K8s filtering layers
**And** Advanced search patterns include phrase matching and proximity search

## Tasks / Subtasks

- [ ] Implement comprehensive content search validation and parsing (AC: 2, 8)
  - [ ] Create ContentSearchValidator with support for multiple search modes
  - [ ] Support exact phrase matching with quoted strings
  - [ ] Implement case-insensitive and case-sensitive search options
  - [ ] Add regex pattern validation and safety checks for content search
  - [ ] Support wildcard patterns for flexible content matching
  - [ ] Add validation for maximum search complexity to prevent expensive queries
- [ ] Develop advanced boolean search logic with operators (AC: 4)
  - [ ] Implement AND, OR, NOT boolean operators for keyword combinations
  - [ ] Support parenthetical grouping for complex search expressions
  - [ ] Add proximity search for keywords within specified distance
  - [ ] Implement exclude patterns to filter out unwanted content
  - [ ] Support field-specific content search (content vs. severity-specific terms)
- [ ] Optimize ClickHouse full-text search with tokenbf_v1 indexing (AC: 3, 6)
  - [ ] Enhance query building to leverage tokenbf_v1 index for content search
  - [ ] Implement efficient content search patterns for various ClickHouse functions
  - [ ] Add content search query optimization with selectivity estimation
  - [ ] Create specialized search queries for different content patterns
  - [ ] Implement search result ranking and relevance scoring
- [ ] Integrate content search with existing filtering layers (AC: 7)
  - [ ] Enhance service layer to combine dataset, time, K8s, and content filters efficiently
  - [ ] Ensure proper filter precedence: dataset → time → K8s → content filters
  - [ ] Maintain backward compatibility with existing filter parameters
  - [ ] Add content search validation to existing request validation pipeline
  - [ ] Optimize combined filter query performance for complex search scenarios
- [ ] Implement search result highlighting and relevance features (AC: 5, 6)
  - [ ] Add server-side highlighting of matched terms in log content
  - [ ] Implement relevance scoring based on match frequency and position
  - [ ] Support highlighting for multiple search terms with different colors/styles
  - [ ] Add search result snippet extraction for long log messages
  - [ ] Create search statistics (total matches, unique terms, hit distribution)

## Dev Notes

### Architecture Compliance Requirements

**Critical:** This story completes Epic 2 by implementing advanced content-based log search capabilities on top of the comprehensive filtering foundation built in Stories 2-1, 2-2, and 2-3. Content search serves as the final filtering layer after dataset routing, time filtering, and Kubernetes metadata filtering, following the architecture's requirement for "content-based log filtering with full-text search capabilities" using ClickHouse's powerful text search features.

**Key Technical Requirements:**
- **Full-Text Search Optimization:** Leverage existing ClickHouse tokenbf_v1 index for high-performance content matching
- **Advanced Search Patterns:** Support exact, case-insensitive, regex, wildcard, and boolean search operations
- **Performance Optimization:** Maintain sub-2 second response times even with complex content search queries
- **Integrated Filtering:** Seamlessly combine with dataset routing (Story 2-1), time filtering (Story 2-2), and K8s filtering (Story 2-3)
- **Search Highlighting:** Provide server-side highlighting and relevance scoring for enhanced user experience

### Content Search Implementation

**Based on architecture.md specifications and Stories 2-1, 2-2, 2-3 foundation, implementing advanced content search:**

```go
// Advanced content search validator with multiple search modes
package query

import (
    "fmt"
    "regexp"
    "strings"
    "unicode/utf8"
)

type ContentSearchValidator struct {
    maxSearchTerms     int
    maxPatternLength   int
    maxQueryComplexity int
    safetyPatterns     []*regexp.Regexp
}

func NewContentSearchValidator() *ContentSearchValidator {
    return &ContentSearchValidator{
        maxSearchTerms:     20,  // Prevent overly complex queries
        maxPatternLength:   500, // Prevent extremely long search patterns
        maxQueryComplexity: 100, // Complex scoring threshold
        safetyPatterns: []*regexp.Regexp{
            // Prevent expensive regex patterns
            regexp.MustCompile(`\.\*\.\*`),     // Multiple greedy quantifiers
            regexp.MustCompile(`\.\+\.\+`),     // Multiple possessive quantifiers
            regexp.MustCompile(`\(\?\<\!\.\*`), // Negative lookbehind with .*
        },
    }
}

// ContentSearchType defines different search modes
type ContentSearchType string

const (
    ContentSearchExact        ContentSearchType = "exact"
    ContentSearchCaseInsensitive ContentSearchType = "icase"
    ContentSearchRegex        ContentSearchType = "regex"
    ContentSearchWildcard     ContentSearchType = "wildcard"
    ContentSearchPhrase       ContentSearchType = "phrase"
    ContentSearchProximity    ContentSearchType = "proximity"
    ContentSearchBoolean      ContentSearchType = "boolean"
)

// ContentSearchFilter represents a single content search condition
type ContentSearchFilter struct {
    Type            ContentSearchType
    Pattern         string
    CaseInsensitive bool
    BooleanOperator string // AND, OR, NOT
    ProximityDistance int   // For proximity searches
    FieldTarget     string // content, severity, etc.
    Weight          float64 // For relevance scoring
}

// ContentSearchExpression represents a complete search expression
type ContentSearchExpression struct {
    Filters      []ContentSearchFilter
    GlobalOperator string // Default operator for combining filters
    HighlightEnabled bool
    MaxSnippetLength int
    RelevanceScoring bool
}

// ParseContentSearch validates and parses content search parameters
func (v *ContentSearchValidator) ParseContentSearch(searchQuery string, options map[string]string) (*ContentSearchExpression, error) {
    if searchQuery == "" {
        return &ContentSearchExpression{}, nil
    }

    if len(searchQuery) > v.maxPatternLength {
        return nil, fmt.Errorf("search query too long (%d chars), max: %d",
            len(searchQuery), v.maxPatternLength)
    }

    if !utf8.ValidString(searchQuery) {
        return nil, fmt.Errorf("search query contains invalid UTF-8 characters")
    }

    // Parse search expression
    expression, err := v.parseSearchExpression(searchQuery, options)
    if err != nil {
        return nil, fmt.Errorf("search parsing failed: %w", err)
    }

    // Validate complexity
    if err := v.validateComplexity(expression); err != nil {
        return nil, fmt.Errorf("search complexity validation failed: %w", err)
    }

    return expression, nil
}

// parseSearchExpression handles complex search query parsing
func (v *ContentSearchValidator) parseSearchExpression(query string, options map[string]string) (*ContentSearchExpression, error) {
    expression := &ContentSearchExpression{
        GlobalOperator:    "AND", // Default
        HighlightEnabled:  true,
        MaxSnippetLength:  200,
        RelevanceScoring:  true,
    }

    // Apply options
    if operator := options["operator"]; operator != "" {
        if operator == "AND" || operator == "OR" {
            expression.GlobalOperator = operator
        }
    }
    if options["highlight"] == "false" {
        expression.HighlightEnabled = false
    }
    if options["relevance"] == "false" {
        expression.RelevanceScoring = false
    }

    // Parse different query formats
    switch {
    case strings.HasPrefix(query, "boolean:"):
        // Boolean search: boolean:error AND (failed OR timeout) NOT debug
        return v.parseBooleanSearch(strings.TrimPrefix(query, "boolean:"), expression)
    case strings.HasPrefix(query, "regex:"):
        // Regex search: regex:error\s+(failed|timeout)
        return v.parseRegexSearch(strings.TrimPrefix(query, "regex:"), expression)
    case strings.HasPrefix(query, "proximity:"):
        // Proximity search: proximity:5:error timeout
        return v.parseProximitySearch(strings.TrimPrefix(query, "proximity:"), expression)
    case strings.Contains(query, `"`):
        // Phrase search with quoted strings: "connection failed" OR "timeout error"
        return v.parsePhraseSearch(query, expression)
    default:
        // Simple keyword search with optional wildcards
        return v.parseSimpleSearch(query, expression)
    }
}

// parseBooleanSearch handles complex boolean expressions
func (v *ContentSearchValidator) parseBooleanSearch(query string, expression *ContentSearchExpression) (*ContentSearchExpression, error) {
    // Parse boolean expression: error AND (failed OR timeout) NOT debug
    tokens, err := v.tokenizeBooleanQuery(query)
    if err != nil {
        return nil, err
    }

    filters, err := v.convertTokensToFilters(tokens)
    if err != nil {
        return nil, err
    }

    expression.Filters = filters
    return expression, nil
}

// parseRegexSearch handles regex pattern search
func (v *ContentSearchValidator) parseRegexSearch(pattern string, expression *ContentSearchExpression) (*ContentSearchExpression, error) {
    // Validate regex pattern safety
    for _, safetyPattern := range v.safetyPatterns {
        if safetyPattern.MatchString(pattern) {
            return nil, fmt.Errorf("unsafe regex pattern detected")
        }
    }

    // Test regex compilation
    _, err := regexp.Compile(pattern)
    if err != nil {
        return nil, fmt.Errorf("invalid regex pattern: %w", err)
    }

    expression.Filters = []ContentSearchFilter{
        {
            Type:     ContentSearchRegex,
            Pattern:  pattern,
            Weight:   1.0,
        },
    }

    return expression, nil
}

// parseProximitySearch handles proximity-based search
func (v *ContentSearchValidator) parseProximitySearch(query string, expression *ContentSearchExpression) (*ContentSearchExpression, error) {
    // Parse: 5:error timeout (words within 5 positions of each other)
    parts := strings.SplitN(query, ":", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("proximity search format: distance:term1 term2")
    }

    distance := 0
    if _, err := fmt.Sscanf(parts[0], "%d", &distance); err != nil {
        return nil, fmt.Errorf("invalid proximity distance: %s", parts[0])
    }

    if distance <= 0 || distance > 50 {
        return nil, fmt.Errorf("proximity distance must be between 1 and 50")
    }

    terms := strings.Fields(parts[1])
    if len(terms) < 2 {
        return nil, fmt.Errorf("proximity search requires at least 2 terms")
    }

    expression.Filters = []ContentSearchFilter{
        {
            Type:              ContentSearchProximity,
            Pattern:           strings.Join(terms, " "),
            ProximityDistance: distance,
            Weight:            1.5, // Higher weight for proximity matches
        },
    }

    return expression, nil
}

// parsePhraseSearch handles quoted phrase search
func (v *ContentSearchValidator) parsePhraseSearch(query string, expression *ContentSearchExpression) (*ContentSearchExpression, error) {
    var filters []ContentSearchFilter

    // Extract quoted phrases and handle operators between them
    phrases := v.extractQuotedPhrases(query)
    operators := v.extractOperators(query)

    for i, phrase := range phrases {
        operator := "AND"
        if i < len(operators) {
            operator = operators[i]
        }

        filters = append(filters, ContentSearchFilter{
            Type:            ContentSearchPhrase,
            Pattern:         phrase,
            BooleanOperator: operator,
            Weight:          1.2, // Higher weight for exact phrases
        })
    }

    expression.Filters = filters
    return expression, nil
}

// parseSimpleSearch handles basic keyword and wildcard search
func (v *ContentSearchValidator) parseSimpleSearch(query string, expression *ContentSearchExpression) (*ContentSearchExpression, error) {
    terms := strings.Fields(query)
    if len(terms) > v.maxSearchTerms {
        return nil, fmt.Errorf("too many search terms (%d), max: %d", len(terms), v.maxSearchTerms)
    }

    var filters []ContentSearchFilter
    for _, term := range terms {
        searchType := ContentSearchExact
        pattern := term
        caseInsensitive := false

        // Detect search patterns
        switch {
        case strings.HasPrefix(term, "icase:"):
            // Case-insensitive: icase:ERROR
            searchType = ContentSearchCaseInsensitive
            pattern = strings.TrimPrefix(term, "icase:")
            caseInsensitive = true
        case strings.Contains(term, "*") || strings.Contains(term, "?"):
            // Wildcard: error*, fail?d
            searchType = ContentSearchWildcard
        case strings.HasPrefix(term, "NOT:"):
            // Exclusion: NOT:debug
            searchType = ContentSearchExact
            pattern = strings.TrimPrefix(term, "NOT:")
            // Handle NOT operator specially
        }

        filters = append(filters, ContentSearchFilter{
            Type:            searchType,
            Pattern:         pattern,
            CaseInsensitive: caseInsensitive,
            BooleanOperator: expression.GlobalOperator,
            Weight:          1.0,
        })
    }

    expression.Filters = filters
    return expression, nil
}

// validateComplexity ensures search complexity is within acceptable bounds
func (v *ContentSearchValidator) validateComplexity(expression *ContentSearchExpression) error {
    complexity := v.calculateComplexity(expression)
    if complexity > float64(v.maxQueryComplexity) {
        return fmt.Errorf("search complexity (%0.1f) exceeds maximum (%d)", complexity, v.maxQueryComplexity)
    }
    return nil
}

// calculateComplexity provides search complexity scoring
func (v *ContentSearchValidator) calculateComplexity(expression *ContentSearchExpression) float64 {
    complexity := 0.0

    for _, filter := range expression.Filters {
        switch filter.Type {
        case ContentSearchExact:
            complexity += 1.0
        case ContentSearchCaseInsensitive:
            complexity += 1.5
        case ContentSearchWildcard:
            complexity += 3.0
        case ContentSearchRegex:
            complexity += 5.0 + float64(len(filter.Pattern))/10 // Regex complexity scales with pattern length
        case ContentSearchPhrase:
            complexity += 2.0
        case ContentSearchProximity:
            complexity += 4.0 + float64(filter.ProximityDistance)/10
        case ContentSearchBoolean:
            complexity += 6.0
        }
    }

    return complexity
}
```

### Enhanced Service Layer with Content Search Integration

**Enhanced service layer to integrate content search with existing filtering layers:**

```go
// Enhanced LogQueryService with comprehensive content search support
func (s *Service) enhanceQueryWithContentSearch(req *request.LogQueryRequest) error {
    // Skip if no content search specified
    if req.Filter == "" && req.ContentSearch == nil {
        return nil
    }

    // Parse content search from filter parameter (backward compatibility)
    if req.Filter != "" && req.ContentSearch == nil {
        contentSearch, err := s.parseFilterAsContentSearch(req.Filter)
        if err != nil {
            return fmt.Errorf("content search parsing failed: %w", err)
        }
        req.ContentSearch = contentSearch
    }

    // Validate content search expression
    validator := NewContentSearchValidator()
    if err := validator.ValidateContentSearchExpression(req.ContentSearch); err != nil {
        return fmt.Errorf("content search validation failed: %w", err)
    }

    return nil
}

// parseFilterAsContentSearch provides backward compatibility for simple filter parameter
func (s *Service) parseFilterAsContentSearch(filter string) (*ContentSearchExpression, error) {
    validator := NewContentSearchValidator()

    // Simple conversion for backward compatibility
    return validator.ParseContentSearch(filter, map[string]string{
        "operator": "AND",
        "highlight": "true",
    })
}

// Enhanced query building with comprehensive content search
func (s *Service) buildContentSearchQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
    var whereConditions []string
    var args []interface{}

    // Dataset must be first WHERE condition (from Story 2.1)
    whereConditions = append(whereConditions, "dataset = ?")
    args = append(args, req.Dataset)

    // Time range filters (from Story 2.2)
    if req.StartTime != nil {
        whereConditions = append(whereConditions, "timestamp >= ?")
        args = append(args, *req.StartTime)
    }
    if req.EndTime != nil {
        whereConditions = append(whereConditions, "timestamp <= ?")
        args = append(args, *req.EndTime)
    }

    // K8s metadata filters (from Story 2.3)
    k8sConditions, k8sArgs, err := s.buildK8sFilterConditions(req.K8sFilters)
    if err != nil {
        return "", nil, fmt.Errorf("failed to build K8s filter conditions: %w", err)
    }
    whereConditions = append(whereConditions, k8sConditions...)
    args = append(args, k8sArgs...)

    // Advanced content search conditions
    contentConditions, contentArgs, err := s.buildContentSearchConditions(req.ContentSearch)
    if err != nil {
        return "", nil, fmt.Errorf("failed to build content search conditions: %w", err)
    }
    whereConditions = append(whereConditions, contentConditions...)
    args = append(args, contentArgs...)

    // Build optimized query with relevance scoring
    selectFields := s.buildSelectFieldsWithHighlighting(req.ContentSearch)
    orderBy := s.buildOrderByWithRelevance(req.ContentSearch)

    query := fmt.Sprintf(`
        %s
        FROM logs
        WHERE %s
        %s
        LIMIT %d OFFSET %d
    `, selectFields, strings.Join(whereConditions, " AND "), orderBy, req.PageSize, req.Page*req.PageSize)

    return query, args, nil
}

// buildContentSearchConditions creates optimized ClickHouse conditions for content search
func (s *Service) buildContentSearchConditions(contentSearch *ContentSearchExpression) ([]string, []interface{}, error) {
    if contentSearch == nil || len(contentSearch.Filters) == 0 {
        return nil, nil, nil
    }

    var conditions []string
    var args []interface{}

    // Group filters by boolean operators
    andFilters := []ContentSearchFilter{}
    orFilters := []ContentSearchFilter{}
    notFilters := []ContentSearchFilter{}

    for _, filter := range contentSearch.Filters {
        switch filter.BooleanOperator {
        case "OR":
            orFilters = append(orFilters, filter)
        case "NOT":
            notFilters = append(notFilters, filter)
        default: // "AND" or empty
            andFilters = append(andFilters, filter)
        }
    }

    // Build AND conditions
    if len(andFilters) > 0 {
        andConditions, andArgs, err := s.buildFilterConditions(andFilters, "AND")
        if err != nil {
            return nil, nil, err
        }
        conditions = append(conditions, andConditions...)
        args = append(args, andArgs...)
    }

    // Build OR conditions
    if len(orFilters) > 0 {
        orConditions, orArgs, err := s.buildFilterConditions(orFilters, "OR")
        if err != nil {
            return nil, nil, err
        }
        if len(orConditions) > 0 {
            conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(orConditions, " OR ")))
            args = append(args, orArgs...)
        }
    }

    // Build NOT conditions
    if len(notFilters) > 0 {
        notConditions, notArgs, err := s.buildFilterConditions(notFilters, "AND")
        if err != nil {
            return nil, nil, err
        }
        for _, notCondition := range notConditions {
            conditions = append(conditions, fmt.Sprintf("NOT (%s)", notCondition))
            args = append(args, notArgs...)
        }
    }

    return conditions, args, nil
}

// buildFilterConditions creates ClickHouse conditions for specific filter types
func (s *Service) buildFilterConditions(filters []ContentSearchFilter, operator string) ([]string, []interface{}, error) {
    var conditions []string
    var args []interface{}

    for _, filter := range filters {
        condition, arg, err := s.buildSingleFilterCondition(filter)
        if err != nil {
            return nil, nil, err
        }
        conditions = append(conditions, condition)
        args = append(args, arg...)
    }

    return conditions, args, nil
}

// buildSingleFilterCondition creates ClickHouse condition for a single filter
func (s *Service) buildSingleFilterCondition(filter ContentSearchFilter) (string, []interface{}, error) {
    var condition string
    var args []interface{}

    switch filter.Type {
    case ContentSearchExact:
        if filter.CaseInsensitive {
            condition = "positionCaseInsensitive(content, ?) > 0"
        } else {
            condition = "position(content, ?) > 0"
        }
        args = append(args, filter.Pattern)

    case ContentSearchCaseInsensitive:
        condition = "positionCaseInsensitive(content, ?) > 0"
        args = append(args, filter.Pattern)

    case ContentSearchRegex:
        condition = "match(content, ?)"
        args = append(args, filter.Pattern)

    case ContentSearchWildcard:
        // Convert wildcard pattern to SQL LIKE pattern
        likePattern := strings.ReplaceAll(strings.ReplaceAll(filter.Pattern, "*", "%"), "?", "_")
        if filter.CaseInsensitive {
            condition = "content ILIKE ?"
        } else {
            condition = "content LIKE ?"
        }
        args = append(args, likePattern)

    case ContentSearchPhrase:
        // Use multiMatchAny for phrase search
        condition = "multiMatchAny(content, [?])"
        // Escape special characters and create phrase pattern
        phrasePattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(filter.Pattern))
        args = append(args, phrasePattern)

    case ContentSearchProximity:
        // Build proximity search using ClickHouse distance functions
        terms := strings.Fields(filter.Pattern)
        if len(terms) < 2 {
            return "", nil, fmt.Errorf("proximity search requires at least 2 terms")
        }

        // Use multiMatchAny with distance calculation
        patterns := make([]string, len(terms))
        for i, term := range terms {
            patterns[i] = regexp.QuoteMeta(term)
        }

        condition = fmt.Sprintf("multiMatchAny(content, [%s]) AND length(splitByString(' ', content)) - length(splitByString('%s', content)) <= %d",
            strings.Join(patterns, ","), strings.Join(terms, " "), filter.ProximityDistance)
        // Note: This is a simplified proximity calculation; more sophisticated logic would be in a UDF

    case ContentSearchBoolean:
        // Boolean search would be handled at a higher level
        return "", nil, fmt.Errorf("boolean search should be decomposed before reaching this level")

    default:
        return "", nil, fmt.Errorf("unsupported content search type: %s", filter.Type)
    }

    return condition, args, nil
}

// buildSelectFieldsWithHighlighting adds highlighting to SELECT clause
func (s *Service) buildSelectFieldsWithHighlighting(contentSearch *ContentSearchExpression) string {
    baseFields := `
        timestamp,
        content,
        severity,
        k8s_namespace_name,
        k8s_pod_name,
        k8s_node_name,
        host_ip,
        host_name,
        container_name,
        container_id`

    if contentSearch == nil || !contentSearch.HighlightEnabled || len(contentSearch.Filters) == 0 {
        return "SELECT" + baseFields
    }

    // Add highlighting fields
    highlightFields := s.buildHighlightingFields(contentSearch)

    return fmt.Sprintf("SELECT%s,%s", baseFields, highlightFields)
}

// buildHighlightingFields creates highlighting expressions for matched content
func (s *Service) buildHighlightingFields(contentSearch *ContentSearchExpression) string {
    var highlightExpressions []string

    for i, filter := range contentSearch.Filters {
        switch filter.Type {
        case ContentSearchExact, ContentSearchCaseInsensitive:
            // Use replaceRegexpAll for highlighting
            highlightExpr := fmt.Sprintf(
                `replaceRegexpAll(content, '(?i)(%s)', '<mark class="highlight-%d">$1</mark>') AS highlighted_content_%d`,
                regexp.QuoteMeta(filter.Pattern), i, i)
            highlightExpressions = append(highlightExpressions, highlightExpr)

        case ContentSearchRegex:
            highlightExpr := fmt.Sprintf(
                `replaceRegexpAll(content, '(%s)', '<mark class="highlight-%d">$1</mark>') AS highlighted_content_%d`,
                filter.Pattern, i, i)
            highlightExpressions = append(highlightExpressions, highlightExpr)
        }
    }

    // Add search relevance scoring
    relevanceExpr := s.buildRelevanceScoring(contentSearch)
    if relevanceExpr != "" {
        highlightExpressions = append(highlightExpressions, relevanceExpr)
    }

    if len(highlightExpressions) > 0 {
        return "\n        " + strings.Join(highlightExpressions, ",\n        ")
    }
    return ""
}

// buildRelevanceScoring creates relevance scoring expression
func (s *Service) buildRelevanceScoring(contentSearch *ContentSearchExpression) string {
    if !contentSearch.RelevanceScoring || len(contentSearch.Filters) == 0 {
        return ""
    }

    var scoreComponents []string
    for i, filter := range contentSearch.Filters {
        var scoreExpr string
        switch filter.Type {
        case ContentSearchExact, ContentSearchCaseInsensitive:
            scoreExpr = fmt.Sprintf("(position(content, '%s') > 0 ? %f : 0)",
                strings.ReplaceAll(filter.Pattern, "'", "''"), filter.Weight)
        case ContentSearchPhrase:
            scoreExpr = fmt.Sprintf("(multiMatchAny(content, ['%s']) ? %f : 0)",
                strings.ReplaceAll(filter.Pattern, "'", "''"), filter.Weight)
        }
        if scoreExpr != "" {
            scoreComponents = append(scoreComponents, scoreExpr)
        }
    }

    if len(scoreComponents) > 0 {
        return fmt.Sprintf("(%s) AS search_relevance_score", strings.Join(scoreComponents, " + "))
    }
    return ""
}

// buildOrderByWithRelevance adds relevance-based ordering
func (s *Service) buildOrderByWithRelevance(contentSearch *ContentSearchExpression) string {
    if contentSearch != nil && contentSearch.RelevanceScoring && len(contentSearch.Filters) > 0 {
        return "ORDER BY search_relevance_score DESC, timestamp DESC"
    }
    return "ORDER BY timestamp DESC"
}
```

### Repository Layer ClickHouse Query Optimization for Content Search

**Enhanced repository layer for optimal content search querying:**

```go
// Enhanced ClickHouse query execution with content search optimization
func (r *ClickHouseRepository) QueryLogsWithContentSearch(ctx context.Context, req *request.LogQueryRequest) ([]model.LogEntry, int, error) {
    // Build content-search-optimized query
    query, args, err := r.buildContentSearchOptimizedQuery(req)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to build content search query: %w", err)
    }

    // Log query execution details for monitoring
    klog.InfoS("Executing content search query",
        "dataset", req.Dataset,
        "content_filters", len(req.ContentSearch.Filters),
        "search_complexity", r.estimateContentSearchComplexity(req))

    // Execute with context timeout
    queryCtx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
    defer cancel()

    startTime := time.Now()
    rows, err := r.conn.Query(queryCtx, query, args...)
    if err != nil {
        return nil, 0, fmt.Errorf("content search query execution failed: %w", err)
    }
    defer rows.Close()

    // Parse results with content search metadata
    logs, err := r.parseLogsWithContentHighlighting(rows, req.ContentSearch)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to parse content search logs: %w", err)
    }

    // Get total count for pagination
    totalCount, err := r.getTotalCountForContentSearch(ctx, req)
    if err != nil {
        klog.ErrorS(err, "Failed to get content search total count", "dataset", req.Dataset)
        totalCount = len(logs)
    }

    duration := time.Since(startTime)
    klog.InfoS("Content search query completed",
        "dataset", req.Dataset,
        "result_count", len(logs),
        "total_count", totalCount,
        "duration_ms", duration.Milliseconds())

    // Performance monitoring for content search queries
    if duration > 2*time.Second {
        klog.InfoS("Slow content search query detected",
            "dataset", req.Dataset,
            "duration_ms", duration.Milliseconds(),
            "search_complexity", r.estimateContentSearchComplexity(req))
    }

    return logs, totalCount, nil
}

// estimateContentSearchComplexity provides query optimization hints
func (r *ClickHouseRepository) estimateContentSearchComplexity(req *request.LogQueryRequest) float64 {
    if req.ContentSearch == nil {
        return 0.0
    }

    complexity := 0.0
    for _, filter := range req.ContentSearch.Filters {
        switch filter.Type {
        case ContentSearchExact:
            complexity += 1.0
        case ContentSearchCaseInsensitive:
            complexity += 1.5
        case ContentSearchWildcard:
            complexity += 3.0
        case ContentSearchRegex:
            complexity += 5.0
        case ContentSearchPhrase:
            complexity += 2.5
        case ContentSearchProximity:
            complexity += 4.0 + float64(filter.ProximityDistance)/10
        }
    }

    // Factor in boolean complexity
    if len(req.ContentSearch.Filters) > 1 {
        complexity *= 1.5
    }

    return complexity
}

// parseLogsWithContentHighlighting ensures content highlighting is properly extracted
func (r *ClickHouseRepository) parseLogsWithContentHighlighting(rows driver.Rows, contentSearch *ContentSearchExpression) ([]model.LogEntry, error) {
    var logs []model.LogEntry

    for rows.Next() {
        var entry model.LogEntry
        var timestamp time.Time

        // Base fields
        scanFields := []interface{}{
            &timestamp,
            &entry.Content,
            &entry.Severity,
            &entry.K8sNamespaceName,
            &entry.K8sPodName,
            &entry.K8sNodeName,
            &entry.HostIP,
            &entry.HostName,
            &entry.ContainerName,
            &entry.ContainerID,
        }

        // Add highlighting fields if enabled
        var highlightedContent []string
        var relevanceScore float64
        if contentSearch != nil && contentSearch.HighlightEnabled {
            for i := 0; i < len(contentSearch.Filters); i++ {
                var highlighted string
                scanFields = append(scanFields, &highlighted)
                highlightedContent = append(highlightedContent, highlighted)
            }
            if contentSearch.RelevanceScoring {
                scanFields = append(scanFields, &relevanceScore)
            }
        }

        err := rows.Scan(scanFields...)
        if err != nil {
            return nil, fmt.Errorf("failed to scan content search row: %w", err)
        }

        // Set timestamp and highlighting data
        entry.Timestamp = timestamp
        if len(highlightedContent) > 0 {
            entry.HighlightedContent = highlightedContent
            entry.SearchRelevanceScore = relevanceScore
        }

        logs = append(logs, entry)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("content search row iteration error: %w", err)
    }

    return logs, nil
}

// getTotalCountForContentSearch gets accurate count for content search filtered queries
func (r *ClickHouseRepository) getTotalCountForContentSearch(ctx context.Context, req *request.LogQueryRequest) (int, error) {
    // Build count query with same content search filters
    countQuery, args, err := r.buildContentSearchCountQuery(req)
    if err != nil {
        return 0, fmt.Errorf("failed to build content search count query: %w", err)
    }

    var count int64
    err = r.conn.QueryRow(ctx, countQuery, args...).Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("content search count query failed: %w", err)
    }

    return int(count), nil
}
```

### Performance Monitoring for Content Search

**Enhanced metrics collection for content search performance:**

```go
// ContentSearchMetrics tracks performance of content search queries
type ContentSearchMetrics struct {
    contentQueryDuration    *prometheus.HistogramVec
    searchComplexity        *prometheus.HistogramVec
    contentMatchRate        *prometheus.HistogramVec
    slowContentQueries      *prometheus.CounterVec
    searchPatternTypes      *prometheus.CounterVec
    indexEfficiency         *prometheus.HistogramVec
}

func NewContentSearchMetrics() *ContentSearchMetrics {
    return &ContentSearchMetrics{
        contentQueryDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_content_search_duration_seconds",
                Help:    "Duration of content search queries",
                Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0},
            },
            []string{"dataset", "complexity_level", "pattern_count"},
        ),
        searchComplexity: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_content_search_complexity",
                Help:    "Complexity score of content search patterns",
                Buckets: []float64{1, 5, 10, 25, 50, 100},
            },
            []string{"dataset"},
        ),
        contentMatchRate: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_content_match_rate",
                Help:    "Match rate for content search queries (matches/total)",
                Buckets: []float64{0.001, 0.01, 0.1, 0.25, 0.5, 0.75, 1.0},
            },
            []string{"dataset", "search_type"},
        ),
        slowContentQueries: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_slow_content_queries_total",
                Help: "Number of slow content search queries (>2s)",
            },
            []string{"dataset", "reason"},
        ),
        searchPatternTypes: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_search_pattern_types_total",
                Help: "Usage count of different content search pattern types",
            },
            []string{"pattern_type"},
        ),
        indexEfficiency: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_content_index_efficiency",
                Help:    "Efficiency of content search index utilization",
                Buckets: prometheus.DefBuckets,
            },
            []string{"dataset", "index_type"},
        ),
    }
}

// RecordContentSearchQuery records metrics for content search queries
func (m *ContentSearchMetrics) RecordContentSearchQuery(dataset string, duration time.Duration,
    searchExpression *ContentSearchExpression, resultCount, totalScanned int) {

    if searchExpression == nil {
        return
    }

    // Calculate metrics
    complexityLevel := m.categorizeComplexity(searchExpression)
    patternCount := fmt.Sprintf("%d", len(searchExpression.Filters))
    complexity := m.calculateComplexity(searchExpression)
    matchRate := float64(resultCount) / float64(max(totalScanned, 1))

    // Record duration with context
    m.contentQueryDuration.With(prometheus.Labels{
        "dataset":        dataset,
        "complexity_level": complexityLevel,
        "pattern_count":  patternCount,
    }).Observe(duration.Seconds())

    // Record complexity
    m.searchComplexity.With(prometheus.Labels{
        "dataset": dataset,
    }).Observe(complexity)

    // Record match rate by search type
    searchType := m.categorizeSearchType(searchExpression)
    m.contentMatchRate.With(prometheus.Labels{
        "dataset":     dataset,
        "search_type": searchType,
    }).Observe(matchRate)

    // Track pattern type usage
    for _, filter := range searchExpression.Filters {
        m.searchPatternTypes.With(prometheus.Labels{
            "pattern_type": string(filter.Type),
        }).Inc()
    }

    // Track slow queries
    if duration > 2*time.Second {
        reason := m.determineSlowReason(duration, complexity, matchRate)
        m.slowContentQueries.With(prometheus.Labels{
            "dataset": dataset,
            "reason":  reason,
        }).Inc()
    }

    // Estimate index efficiency (simplified)
    indexEfficiency := m.estimateIndexEfficiency(searchExpression, resultCount, totalScanned)
    m.indexEfficiency.With(prometheus.Labels{
        "dataset":    dataset,
        "index_type": "tokenbf_v1",
    }).Observe(indexEfficiency)
}

// categorizeComplexity determines complexity level of content search
func (m *ContentSearchMetrics) categorizeComplexity(searchExpression *ContentSearchExpression) string {
    complexity := m.calculateComplexity(searchExpression)
    switch {
    case complexity <= 2:
        return "simple"
    case complexity <= 10:
        return "moderate"
    case complexity <= 25:
        return "complex"
    default:
        return "very_complex"
    }
}

// calculateComplexity provides numeric complexity scoring
func (m *ContentSearchMetrics) calculateComplexity(searchExpression *ContentSearchExpression) float64 {
    complexity := 0.0
    for _, filter := range searchExpression.Filters {
        switch filter.Type {
        case ContentSearchExact:
            complexity += 1.0
        case ContentSearchCaseInsensitive:
            complexity += 1.5
        case ContentSearchWildcard:
            complexity += 3.0
        case ContentSearchRegex:
            complexity += 5.0 + float64(len(filter.Pattern))/20
        case ContentSearchPhrase:
            complexity += 2.5
        case ContentSearchProximity:
            complexity += 4.0 + float64(filter.ProximityDistance)/5
        case ContentSearchBoolean:
            complexity += 6.0
        }
    }

    // Boolean complexity multiplier
    if len(searchExpression.Filters) > 1 {
        complexity *= 1.5
    }

    return complexity
}

// estimateIndexEfficiency provides index utilization scoring
func (m *ContentSearchMetrics) estimateIndexEfficiency(searchExpression *ContentSearchExpression, resultCount, totalScanned int) float64 {
    if totalScanned == 0 {
        return 1.0
    }

    // Base efficiency from selectivity
    baseEfficiency := float64(resultCount) / float64(totalScanned)

    // Adjust based on search patterns (some patterns benefit more from indexing)
    indexFriendliness := 1.0
    for _, filter := range searchExpression.Filters {
        switch filter.Type {
        case ContentSearchExact:
            indexFriendliness *= 1.2 // Exact matches benefit most from indexing
        case ContentSearchCaseInsensitive:
            indexFriendliness *= 1.1
        case ContentSearchWildcard:
            indexFriendliness *= 0.8 // Wildcards are less index-friendly
        case ContentSearchRegex:
            indexFriendliness *= 0.6 // Regex can be expensive
        case ContentSearchPhrase:
            indexFriendliness *= 1.0 // Neutral
        case ContentSearchProximity:
            indexFriendliness *= 0.7 // Complex proximity calculations
        }
    }

    return baseEfficiency * indexFriendliness
}
```

### API Documentation and Usage Examples

**Enhanced API documentation with advanced content search examples:**

```bash
# Basic content search (exact match)
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?start_time=2024-01-01T10:00:00Z&end_time=2024-01-01T10:30:00Z&filter=error"

# Case-insensitive content search
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-cn-hz01/logs?content_search=icase:ERROR&namespace=production"

# Wildcard content search
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/staging-app/logs?content_search=connect*failure&start_time=2024-01-01T10:00:00Z"

# Regex content search
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?content_search=regex:error\s+(timeout|failed)&pod_names=web-*"

# Phrase search with quotes
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-cluster/logs?content_search=\"connection timeout\"&namespaces=kube-system"

# Boolean search with multiple operators
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?content_search=boolean:error AND (failed OR timeout) NOT debug&start_time=2024-01-01T08:00:00Z"

# Proximity search (words within 5 positions)
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/staging/logs?content_search=proximity:5:database connection&namespace=backend"

# Complex content search with all filters
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-prod/logs?content_search=boolean:(error OR warning) AND timeout&namespaces=prod-*&pods=*api*&start_time=2024-01-01T09:00:00Z&end_time=2024-01-01T10:00:00Z&highlight=true&relevance=true"

# Response includes highlighted content and relevance scoring
{
  "items": [
    {
      "timestamp": "2024-01-01T10:15:30.123Z",
      "content": "Database connection timeout after 30 seconds",
      "highlighted_content": [
        "Database <mark class=\"highlight-0\">connection</mark> <mark class=\"highlight-1\">timeout</mark> after 30 seconds"
      ],
      "search_relevance_score": 2.5,
      "severity": "ERROR",
      "k8s_namespace_name": "production",
      "k8s_pod_name": "api-service-abc123",
      "k8s_node_name": "edge-node-01"
    }
  ],
  "dataset": "edge-prod",
  "total": 89,
  "has_more": true,
  "search_metadata": {
    "patterns_matched": 2,
    "total_highlights": 156,
    "search_complexity": 4.5,
    "query_time_ms": 234
  }
}
```

### Error Handling for Content Search Operations

**Comprehensive error handling for content search:**

```go
// Content-search-specific error types
type ContentSearchValidationError struct {
    Pattern     string
    SearchType  ContentSearchType
    Reason      string
    Suggestion  string
}

func (e *ContentSearchValidationError) Error() string {
    return fmt.Sprintf("content search validation failed for pattern '%s' (%s): %s",
        e.Pattern, e.SearchType, e.Reason)
}

type ContentSearchComplexityError struct {
    Complexity float64
    MaxAllowed float64
    Patterns   []string
}

func (e *ContentSearchComplexityError) Error() string {
    return fmt.Sprintf("content search complexity (%.1f) exceeds maximum (%.1f) for patterns: %v",
        e.Complexity, e.MaxAllowed, e.Patterns)
}

// Enhanced error handling in API layer
func (h *LogHandler) handleContentSearchError(resp *restful.Response, err error, dataset string) {
    switch e := err.(type) {
    case *ContentSearchValidationError:
        errorResp := map[string]interface{}{
            "error":        "Invalid content search pattern",
            "pattern":      e.Pattern,
            "search_type":  e.SearchType,
            "reason":       e.Reason,
            "suggestion":   e.Suggestion,
            "dataset":      dataset,
            "supported_patterns": map[string][]string{
                "exact": {
                    "error",
                    "timeout",
                    "connection failed",
                },
                "case_insensitive": {
                    "icase:ERROR",
                    "icase:warning",
                },
                "wildcard": {
                    "error*",
                    "*timeout*",
                    "connect?on",
                },
                "regex": {
                    "regex:error\\s+(failed|timeout)",
                    "regex:^[0-9]{4}-[0-9]{2}-[0-9]{2}",
                },
                "phrase": {
                    "\"connection timeout\"",
                    "\"database error\"",
                },
                "boolean": {
                    "boolean:error AND failed",
                    "boolean:warning OR error NOT debug",
                },
                "proximity": {
                    "proximity:5:database connection",
                    "proximity:10:error timeout",
                },
            },
            "examples": []string{
                "filter=error",
                "content_search=icase:WARNING",
                "content_search=*timeout*",
                "content_search=regex:error\\s+(failed|timeout)",
                "content_search=\"connection failed\"",
                "content_search=boolean:error AND (timeout OR failed)",
                "content_search=proximity:5:database connection",
            },
        }
        h.writeErrorResponse(resp, http.StatusBadRequest, errorResp)

    case *ContentSearchComplexityError:
        errorResp := map[string]interface{}{
            "error":       "Content search complexity too high",
            "complexity":  e.Complexity,
            "max_allowed": e.MaxAllowed,
            "patterns":    e.Patterns,
            "dataset":     dataset,
            "optimization_tips": []string{
                "Use exact matches instead of wildcards when possible",
                "Avoid complex regex patterns with multiple quantifiers",
                "Limit boolean expressions to essential terms",
                "Consider breaking complex searches into multiple queries",
                "Use phrase search for exact multi-word matches",
            },
        }
        h.writeErrorResponse(resp, http.StatusBadRequest, errorResp)

    default:
        h.writeErrorResponse(resp, http.StatusBadRequest, "Content search error: "+err.Error())
    }

    h.metrics.RecordContentSearchError(dataset, "validation_error")
}
```

### Testing Strategy for Content Search

**Comprehensive testing strategy for content search functionality:**

1. **Unit Tests:**
   - Content search pattern parsing and validation
   - Boolean expression parsing and compilation
   - Regex pattern safety validation
   - Query building with various search combinations
   - Highlighting and relevance scoring logic
   - Search complexity calculation

2. **Integration Tests:**
   - End-to-end content search with real ClickHouse data
   - Complex boolean search expressions
   - Performance testing with large content datasets
   - Search highlighting and relevance scoring
   - Integration with existing filter layers (dataset, time, K8s)

3. **Performance Tests:**
   - Content search query performance benchmarking
   - tokenbf_v1 index effectiveness measurement
   - Memory usage for complex search operations
   - Concurrent content search query handling
   - Large-scale content search optimization

### Project Structure Notes

**File organization completing Epic 2 foundation:**

```
pkg/service/query/
├── service.go                     # Enhanced with content search (modify existing)
├── content_search_validator.go    # Content search validation logic (new)
├── content_search_builder.go      # Content search query building (new)
├── k8s_validator.go               # Existing from Story 2.3
├── time_validator.go              # Existing from Story 2.2
├── dataset_validator.go           # Existing from Story 2.1
└── service_test.go               # Enhanced with content search tests (modify existing)

pkg/repository/clickhouse/
├── repository.go                 # Enhanced with content search optimization (modify existing)
├── content_search_queries.go     # Content search-specific query patterns (new)
├── k8s_queries.go                # Existing from Story 2.3
├── time_queries.go               # Existing from Story 2.2
└── repository_test.go            # Enhanced with content search tests (modify existing)

pkg/oapis/log/v1alpha1/
├── handler.go                    # Enhanced with content search parameters (modify existing)
├── content_search_errors.go      # Content search-specific error types (new)
├── content_search_metrics.go     # Content search metrics (new)
├── k8s_errors.go                 # Existing from Story 2.3
└── handler_test.go               # Enhanced with content search tests (modify existing)

pkg/model/request/
└── log.go                        # Enhanced with content search fields (modify existing)

pkg/model/response/
└── log.go                        # Enhanced with highlighting and relevance (modify existing)
```

**Key Integration Points:**
- Completes Epic 2 by building upon dataset routing (Story 2-1), time filtering (Story 2-2), and K8s filtering (Story 2-3)
- Enhances existing service layer with comprehensive content search integration
- Leverages existing ClickHouse tokenbf_v1 index from Story 1-2 schema
- Provides foundation for Epic 3 advanced analytics with enriched search capabilities

### Dependencies and Version Requirements

**No new dependencies required - leverages existing stack:**

```go
// Existing dependencies from previous stories
require (
    github.com/ClickHouse/clickhouse-go/v2 v2.15.0  // Full-text search support
    github.com/emicklei/go-restful/v3 v3.11.0        // API framework
    k8s.io/klog/v2 v2.100.1                          // Structured logging
    github.com/prometheus/client_golang v1.17.0      // Content search metrics
    github.com/stretchr/testify v1.8.4                // Testing framework
)
```

### Performance Requirements

**Content search performance targets:**

- **Search Validation:** < 3ms per request for complex pattern parsing and validation
- **Query Building:** < 15ms additional latency for complex content search query construction
- **ClickHouse Execution:** < 2 seconds for bounded content search queries with up to 10 patterns
- **tokenbf_v1 Optimization:** Effective utilization of ClickHouse full-text index
- **Memory Usage:** < 30MB additional memory for complex content search operations
- **Highlighting Performance:** < 50ms for search result highlighting and relevance scoring
- **Concurrent Content Searches:** Support 150+ simultaneous content search queries

### References

- [Source: _bmad-output/epics.md#Story 2.4] - Complete user story and acceptance criteria
- [Source: _bmad-output/architecture.md#内容搜索] - Content search requirements with tokenbf_v1 indexing
- [Source: _bmad-output/2-3-namespace-and-pod-filtering.md] - Foundation for content search enhancement
- [Source: _bmad-output/2-2-time-range-filtering-with-millisecond-precision.md] - Time filtering foundation
- [Source: _bmad-output/2-1-dataset-based-query-routing.md] - Dataset routing foundation
- [Source: sqlscripts/clickhouse/01_tables.sql#tokenbf_v1] - ClickHouse content search index
- [Source: pkg/model/request/log.go#ContentSearch] - Content search field definitions
- [Source: pkg/service/query/service.go] - Service layer to enhance with content search
- [Source: pkg/repository/clickhouse/repository.go] - Repository layer for content search queries

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-20250514

### Debug Log References

Completing Epic 2 by implementing comprehensive content-based log search capabilities on top of the filtering foundation built in Stories 2-1, 2-2, and 2-3. Content search serves as the final filtering layer after dataset routing, time filtering, and Kubernetes metadata filtering, leveraging ClickHouse's powerful tokenbf_v1 index for high-performance full-text search with advanced patterns, boolean logic, and search result highlighting.

### Completion Notes List

Story 2-4 completes Epic 2: Dataset Management and Data Isolation by implementing advanced content-based log search functionality. Builds upon the comprehensive filtering foundation from Stories 2-1 (dataset routing), 2-2 (time filtering), and 2-3 (K8s filtering) to provide powerful full-text search capabilities with multiple pattern types (exact, case-insensitive, regex, wildcard, phrase, proximity, boolean), search result highlighting, relevance scoring, and performance optimization using ClickHouse's tokenbf_v1 index. This story provides the complete filtering stack needed for Epic 3 advanced analytics and Epic 4 web interface development.

### File List

Primary files to be enhanced/created:
- pkg/service/query/service.go (enhance with content search integration)
- pkg/service/query/content_search_validator.go (new)
- pkg/service/query/content_search_builder.go (new)
- pkg/repository/clickhouse/repository.go (enhance with content search optimization)
- pkg/repository/clickhouse/content_search_queries.go (new)
- pkg/oapis/log/v1alpha1/handler.go (enhance with content search parameters)
- pkg/oapis/log/v1alpha1/content_search_errors.go (new)
- pkg/oapis/log/v1alpha1/content_search_metrics.go (new)
- pkg/model/request/log.go (enhance with content search fields)
- pkg/model/response/log.go (enhance with highlighting and relevance)
- pkg/service/query/service_test.go (enhance with content search tests)
- pkg/repository/clickhouse/repository_test.go (enhance with content search tests)
- pkg/oapis/log/v1alpha1/handler_test.go (enhance with content search tests)