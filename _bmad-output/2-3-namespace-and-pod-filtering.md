# Story 2.3: Namespace and pod filtering

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want to filter logs by Kubernetes namespace and pod names with advanced matching patterns,
So that I can efficiently focus on specific applications, services, or workloads when troubleshooting issues across edge computing deployments.

## Acceptance Criteria

**Given** Time-range filtering is implemented (Story 2-2 completed)
**When** I specify namespace or pod_name query parameters
**Then** Only logs matching the specified Kubernetes metadata are returned
**And** Filtering uses k8s_namespace_name and k8s_pod_name columns efficiently with proper indexing
**And** Multiple filter matching patterns are supported (exact, prefix, regex)
**And** Multiple namespaces and pods can be specified in a single query
**And** Performance is optimized for K8s metadata indexing in ClickHouse
**And** API parameter validation ensures K8s-compliant resource names
**And** Proper error handling for non-existent namespaces or pods
**And** Integration with existing dataset and time-range filtering maintains query performance

## Tasks / Subtasks

- [ ] Implement K8s resource name validation and parsing (AC: 6)
  - [ ] Create K8sResourceValidator with DNS-1123 compliant validation
  - [ ] Support multiple namespace specification (comma-separated, array)
  - [ ] Support multiple pod name specification with various matching patterns
  - [ ] Implement K8s resource name sanitization to prevent injection
  - [ ] Add validation for maximum filter complexity to prevent expensive queries
- [ ] Enhance K8s metadata filtering with pattern matching (AC: 3)
  - [ ] Implement exact matching for namespace and pod names
  - [ ] Add prefix matching support for pod names (starts-with pattern)
  - [ ] Add regex pattern matching for advanced pod name filtering
  - [ ] Support wildcard patterns for namespace and pod selection
  - [ ] Add case-insensitive matching options for pod names
- [ ] Optimize ClickHouse queries for K8s metadata performance (AC: 5)
  - [ ] Enhance query building to leverage LowCardinality optimization
  - [ ] Implement efficient IN clause for multiple namespace/pod filters
  - [ ] Add K8s metadata indexing strategy for optimal partition pruning
  - [ ] Create specialized query patterns for common K8s filtering scenarios
  - [ ] Add query plan analysis for K8s metadata query optimization
- [ ] Integrate with existing filtering layers (AC: 8)
  - [ ] Enhance service layer to combine dataset, time, and K8s filters efficiently
  - [ ] Ensure proper filter precedence: dataset → time → K8s → content
  - [ ] Maintain backward compatibility with existing API parameters
  - [ ] Add K8s filter validation to existing request validation pipeline
  - [ ] Optimize combined filter query performance for complex scenarios
- [ ] Add comprehensive error handling for K8s filtering (AC: 7)
  - [ ] Create K8s-specific error types for validation failures
  - [ ] Implement user-friendly error messages for K8s resource name issues
  - [ ] Add helpful suggestions for invalid namespace/pod patterns
  - [ ] Handle edge cases with special K8s characters and naming rules
  - [ ] Add monitoring for K8s filter error patterns and optimization opportunities

## Dev Notes

### Architecture Compliance Requirements

**Critical:** This story enhances the existing dataset and time-range filtering system (Stories 2-1, 2-2) to provide comprehensive Kubernetes-native log filtering capabilities. K8s filtering serves as the third filtering layer after dataset and time scoping, following the architecture's requirement for "namespace and pod based log filtering" aligned with edge computing K8s deployments.

**Key Technical Requirements:**
- **K8s-Native Filtering:** Leverage existing k8s_namespace_name and k8s_pod_name LowCardinality columns for optimal performance
- **Edge Computing Patterns:** Support multi-namespace and cross-pod queries common in edge deployments
- **Performance Optimization:** Maintain sub-2 second response times with complex K8s metadata filtering
- **Multiple Pattern Matching:** Support exact, prefix, regex, and wildcard matching for flexible pod selection
- **Query Integration:** Seamlessly combine with dataset routing and time filtering without performance degradation

### K8s Namespace and Pod Filtering Implementation

**Based on architecture.md specifications and Stories 2-1, 2-2 foundation, implementing K8s-native filtering:**

```go
// Enhanced K8s resource validator with DNS-1123 compliance
package query

import (
    "fmt"
    "regexp"
    "strings"
)

type K8sResourceValidator struct {
    namespaceRegex    *regexp.Regexp
    podNameRegex      *regexp.Regexp
    maxFilterCount    int
    maxPatternLength  int
}

func NewK8sResourceValidator() *K8sResourceValidator {
    return &K8sResourceValidator{
        // DNS-1123 compliant validation for namespaces
        namespaceRegex: regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`),
        // More permissive for pod names (allows uppercase, dots)
        podNameRegex: regexp.MustCompile(`^[a-z0-9A-Z]([-a-z0-9A-Z._]*[a-z0-9A-Z])?$`),
        maxFilterCount:   50,  // Prevent overly complex queries
        maxPatternLength: 255, // Prevent extremely long patterns
    }
}

// K8sFilterType defines different matching patterns
type K8sFilterType string

const (
    K8sFilterExact     K8sFilterType = "exact"
    K8sFilterPrefix    K8sFilterType = "prefix"
    K8sFilterSuffix    K8sFilterType = "suffix"
    K8sFilterContains  K8sFilterType = "contains"
    K8sFilterRegex     K8sFilterType = "regex"
    K8sFilterWildcard  K8sFilterType = "wildcard"
)

// K8sFilter represents a single K8s filtering condition
type K8sFilter struct {
    Type      K8sFilterType
    Pattern   string
    Field     string // "namespace" or "pod"
    CaseInsensitive bool
}

// ParseK8sFilters validates and parses K8s filtering parameters
func (v *K8sResourceValidator) ParseK8sFilters(namespaces, pods []string) ([]K8sFilter, error) {
    var filters []K8sFilter

    // Parse namespace filters
    for _, ns := range namespaces {
        if ns == "" {
            continue
        }

        filter, err := v.parseNamespaceFilter(ns)
        if err != nil {
            return nil, fmt.Errorf("invalid namespace filter '%s': %w", ns, err)
        }
        filters = append(filters, filter)
    }

    // Parse pod name filters
    for _, pod := range pods {
        if pod == "" {
            continue
        }

        filter, err := v.parsePodFilter(pod)
        if err != nil {
            return nil, fmt.Errorf("invalid pod filter '%s': %w", pod, err)
        }
        filters = append(filters, filter)
    }

    // Validate total filter count
    if len(filters) > v.maxFilterCount {
        return nil, fmt.Errorf("too many K8s filters (%d), maximum allowed: %d",
            len(filters), v.maxFilterCount)
    }

    return filters, nil
}

// parseNamespaceFilter parses and validates namespace filter patterns
func (v *K8sResourceValidator) parseNamespaceFilter(namespace string) (K8sFilter, error) {
    if len(namespace) > v.maxPatternLength {
        return K8sFilter{}, fmt.Errorf("namespace pattern too long (%d chars), max: %d",
            len(namespace), v.maxPatternLength)
    }

    // Detect filter type based on pattern
    filter := K8sFilter{Field: "namespace"}

    switch {
    case strings.HasPrefix(namespace, "regex:"):
        // Regex pattern: regex:^kube-.*
        filter.Type = K8sFilterRegex
        filter.Pattern = strings.TrimPrefix(namespace, "regex:")
        if err := v.validateRegexPattern(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid regex pattern: %w", err)
        }

    case strings.Contains(namespace, "*") || strings.Contains(namespace, "?"):
        // Wildcard pattern: kube-* or test-?-env
        filter.Type = K8sFilterWildcard
        filter.Pattern = namespace
        if err := v.validateWildcardPattern(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid wildcard pattern: %w", err)
        }

    case strings.HasSuffix(namespace, "*"):
        // Prefix pattern: kube-*
        filter.Type = K8sFilterPrefix
        filter.Pattern = strings.TrimSuffix(namespace, "*")
        if err := v.validateNamespaceFormat(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid prefix pattern: %w", err)
        }

    default:
        // Exact match
        filter.Type = K8sFilterExact
        filter.Pattern = namespace
        if err := v.validateNamespaceFormat(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid namespace name: %w", err)
        }
    }

    return filter, nil
}

// parsePodFilter parses and validates pod name filter patterns
func (v *K8sResourceValidator) parsePodFilter(podName string) (K8sFilter, error) {
    if len(podName) > v.maxPatternLength {
        return K8sFilter{}, fmt.Errorf("pod pattern too long (%d chars), max: %d",
            len(podName), v.maxPatternLength)
    }

    filter := K8sFilter{Field: "pod"}

    // Check for case-insensitive prefix
    if strings.HasPrefix(strings.ToLower(podName), "icase:") {
        filter.CaseInsensitive = true
        podName = podName[6:] // Remove "icase:" prefix
    }

    switch {
    case strings.HasPrefix(podName, "regex:"):
        // Regex pattern: regex:^app-.*-[0-9]+$
        filter.Type = K8sFilterRegex
        filter.Pattern = strings.TrimPrefix(podName, "regex:")
        if err := v.validateRegexPattern(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid regex pattern: %w", err)
        }

    case strings.Contains(podName, "*") || strings.Contains(podName, "?"):
        // Wildcard pattern: app-* or web-??-prod
        filter.Type = K8sFilterWildcard
        filter.Pattern = podName

    case strings.HasPrefix(podName, "*") && strings.HasSuffix(podName, "*"):
        // Contains pattern: *web-server*
        filter.Type = K8sFilterContains
        filter.Pattern = strings.Trim(podName, "*")
        if err := v.validatePodNameFormat(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid contains pattern: %w", err)
        }

    case strings.HasSuffix(podName, "*"):
        // Prefix pattern: web-app-*
        filter.Type = K8sFilterPrefix
        filter.Pattern = strings.TrimSuffix(podName, "*")
        if err := v.validatePodNameFormat(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid prefix pattern: %w", err)
        }

    case strings.HasPrefix(podName, "*"):
        // Suffix pattern: *-worker
        filter.Type = K8sFilterSuffix
        filter.Pattern = strings.TrimPrefix(podName, "*")
        if err := v.validatePodNameFormat(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid suffix pattern: %w", err)
        }

    default:
        // Exact match
        filter.Type = K8sFilterExact
        filter.Pattern = podName
        if err := v.validatePodNameFormat(filter.Pattern); err != nil {
            return filter, fmt.Errorf("invalid pod name: %w", err)
        }
    }

    return filter, nil
}

// validateNamespaceFormat ensures namespace follows DNS-1123 rules
func (v *K8sResourceValidator) validateNamespaceFormat(namespace string) error {
    if len(namespace) == 0 {
        return fmt.Errorf("namespace cannot be empty")
    }
    if len(namespace) > 63 {
        return fmt.Errorf("namespace too long (%d chars), max 63", len(namespace))
    }
    if !v.namespaceRegex.MatchString(namespace) {
        return fmt.Errorf("namespace format invalid: must be DNS-1123 compliant (lowercase alphanumeric and hyphens)")
    }
    return nil
}

// validatePodNameFormat ensures pod name follows K8s naming rules
func (v *K8sResourceValidator) validatePodNameFormat(podName string) error {
    if len(podName) == 0 {
        return fmt.Errorf("pod name cannot be empty")
    }
    if len(podName) > 253 {
        return fmt.Errorf("pod name too long (%d chars), max 253", len(podName))
    }
    if !v.podNameRegex.MatchString(podName) {
        return fmt.Errorf("pod name format invalid: must follow K8s naming conventions")
    }
    return nil
}

// validateRegexPattern ensures regex patterns are safe and valid
func (v *K8sResourceValidator) validateRegexPattern(pattern string) error {
    _, err := regexp.Compile(pattern)
    if err != nil {
        return fmt.Errorf("invalid regex pattern: %w", err)
    }

    // Additional safety checks for potentially expensive regex patterns
    if strings.Contains(pattern, ".*.*") || strings.Contains(pattern, ".+.+") {
        return fmt.Errorf("regex pattern may be too expensive (multiple greedy quantifiers)")
    }

    return nil
}
```

### Enhanced Service Layer with K8s Filtering Integration

**Enhanced service layer to integrate K8s filters with existing filtering:**

```go
// Enhanced LogQueryRequest with K8s filtering support
func (s *Service) enhanceQueryWithK8sFilters(req *request.LogQueryRequest) error {
    // Parse and validate K8s filters
    k8sValidator := NewK8sResourceValidator()

    // Parse namespaces from various input formats
    namespaces := s.parseNamespaceInput(req.Namespace, req.Namespaces)
    pods := s.parsePodInput(req.PodName, req.PodNames)

    k8sFilters, err := k8sValidator.ParseK8sFilters(namespaces, pods)
    if err != nil {
        return fmt.Errorf("K8s filter validation failed: %w", err)
    }

    // Store parsed filters in request for query building
    req.K8sFilters = k8sFilters

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

// Enhanced query building with K8s metadata optimization
func (s *Service) buildK8sOptimizedQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
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

    // Build K8s metadata filters with optimized patterns
    k8sConditions, k8sArgs, err := s.buildK8sFilterConditions(req.K8sFilters)
    if err != nil {
        return "", nil, fmt.Errorf("failed to build K8s filter conditions: %w", err)
    }
    whereConditions = append(whereConditions, k8sConditions...)
    args = append(args, k8sArgs...)

    // Additional filters (content, severity, etc.)
    if req.Filter != "" {
        whereConditions = append(whereConditions, "positionCaseInsensitive(content, ?) > 0")
        args = append(args, req.Filter)
    }

    // Build optimized query with proper ordering for K8s metadata queries
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
        ORDER BY timestamp DESC, k8s_namespace_name ASC, k8s_pod_name ASC
        LIMIT %d OFFSET %d
    `, strings.Join(whereConditions, " AND "), req.PageSize, req.Page*req.PageSize)

    return query, args, nil
}

// buildK8sFilterConditions creates optimized ClickHouse conditions for K8s filters
func (s *Service) buildK8sFilterConditions(filters []K8sFilter) ([]string, []interface{}, error) {
    var conditions []string
    var args []interface{}

    // Group filters by field and type for optimization
    namespaceFilters := make(map[K8sFilterType][]string)
    podFilters := make(map[K8sFilterType][]string)

    for _, filter := range filters {
        if filter.Field == "namespace" {
            namespaceFilters[filter.Type] = append(namespaceFilters[filter.Type], filter.Pattern)
        } else if filter.Field == "pod" {
            podFilters[filter.Type] = append(podFilters[filter.Type], filter.Pattern)
        }
    }

    // Build optimized namespace conditions
    if nsConditions, nsArgs, err := s.buildNamespaceConditions(namespaceFilters); err != nil {
        return nil, nil, err
    } else if len(nsConditions) > 0 {
        conditions = append(conditions, nsConditions...)
        args = append(args, nsArgs...)
    }

    // Build optimized pod conditions
    if podConditions, podArgs, err := s.buildPodConditions(podFilters); err != nil {
        return nil, nil, err
    } else if len(podConditions) > 0 {
        conditions = append(conditions, podConditions...)
        args = append(args, podArgs...)
    }

    return conditions, args, nil
}

// buildNamespaceConditions creates efficient ClickHouse conditions for namespace filtering
func (s *Service) buildNamespaceConditions(filters map[K8sFilterType][]string) ([]string, []interface{}, error) {
    var conditions []string
    var args []interface{}

    // Handle exact matches with IN clause for efficiency
    if exactFilters := filters[K8sFilterExact]; len(exactFilters) > 0 {
        if len(exactFilters) == 1 {
            conditions = append(conditions, "k8s_namespace_name = ?")
            args = append(args, exactFilters[0])
        } else {
            placeholders := make([]string, len(exactFilters))
            for i, ns := range exactFilters {
                placeholders[i] = "?"
                args = append(args, ns)
            }
            conditions = append(conditions, fmt.Sprintf("k8s_namespace_name IN (%s)",
                strings.Join(placeholders, ",")))
        }
    }

    // Handle prefix matches
    if prefixFilters := filters[K8sFilterPrefix]; len(prefixFilters) > 0 {
        var prefixConditions []string
        for _, prefix := range prefixFilters {
            prefixConditions = append(prefixConditions, "startsWith(k8s_namespace_name, ?)")
            args = append(args, prefix)
        }
        if len(prefixConditions) == 1 {
            conditions = append(conditions, prefixConditions[0])
        } else {
            conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(prefixConditions, " OR ")))
        }
    }

    // Handle regex matches
    if regexFilters := filters[K8sFilterRegex]; len(regexFilters) > 0 {
        var regexConditions []string
        for _, pattern := range regexFilters {
            regexConditions = append(regexConditions, "match(k8s_namespace_name, ?)")
            args = append(args, pattern)
        }
        if len(regexConditions) == 1 {
            conditions = append(conditions, regexConditions[0])
        } else {
            conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(regexConditions, " OR ")))
        }
    }

    // Handle wildcard matches
    if wildcardFilters := filters[K8sFilterWildcard]; len(wildcardFilters) > 0 {
        var wildcardConditions []string
        for _, pattern := range wildcardFilters {
            // Convert wildcard pattern to SQL LIKE pattern
            likePattern := strings.ReplaceAll(strings.ReplaceAll(pattern, "*", "%"), "?", "_")
            wildcardConditions = append(wildcardConditions, "k8s_namespace_name LIKE ?")
            args = append(args, likePattern)
        }
        if len(wildcardConditions) == 1 {
            conditions = append(conditions, wildcardConditions[0])
        } else {
            conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(wildcardConditions, " OR ")))
        }
    }

    return conditions, args, nil
}

// buildPodConditions creates efficient ClickHouse conditions for pod filtering
func (s *Service) buildPodConditions(filters map[K8sFilterType][]string) ([]string, []interface{}, error) {
    var conditions []string
    var args []interface{}

    // Handle exact matches
    if exactFilters := filters[K8sFilterExact]; len(exactFilters) > 0 {
        if len(exactFilters) == 1 {
            conditions = append(conditions, "k8s_pod_name = ?")
            args = append(args, exactFilters[0])
        } else {
            placeholders := make([]string, len(exactFilters))
            for i, pod := range exactFilters {
                placeholders[i] = "?"
                args = append(args, pod)
            }
            conditions = append(conditions, fmt.Sprintf("k8s_pod_name IN (%s)",
                strings.Join(placeholders, ",")))
        }
    }

    // Handle prefix matches
    if prefixFilters := filters[K8sFilterPrefix]; len(prefixFilters) > 0 {
        var prefixConditions []string
        for _, prefix := range prefixFilters {
            prefixConditions = append(prefixConditions, "startsWith(k8s_pod_name, ?)")
            args = append(args, prefix)
        }
        conditions = append(conditions, s.combineConditions(prefixConditions))
    }

    // Handle suffix matches
    if suffixFilters := filters[K8sFilterSuffix]; len(suffixFilters) > 0 {
        var suffixConditions []string
        for _, suffix := range suffixFilters {
            suffixConditions = append(suffixConditions, "endsWith(k8s_pod_name, ?)")
            args = append(args, suffix)
        }
        conditions = append(conditions, s.combineConditions(suffixConditions))
    }

    // Handle contains matches
    if containsFilters := filters[K8sFilterContains]; len(containsFilters) > 0 {
        var containsConditions []string
        for _, contains := range containsFilters {
            containsConditions = append(containsConditions, "position(k8s_pod_name, ?) > 0")
            args = append(args, contains)
        }
        conditions = append(conditions, s.combineConditions(containsConditions))
    }

    // Handle regex matches
    if regexFilters := filters[K8sFilterRegex]; len(regexFilters) > 0 {
        var regexConditions []string
        for _, pattern := range regexFilters {
            regexConditions = append(regexConditions, "match(k8s_pod_name, ?)")
            args = append(args, pattern)
        }
        conditions = append(conditions, s.combineConditions(regexConditions))
    }

    // Handle wildcard matches
    if wildcardFilters := filters[K8sFilterWildcard]; len(wildcardFilters) > 0 {
        var wildcardConditions []string
        for _, pattern := range wildcardFilters {
            likePattern := strings.ReplaceAll(strings.ReplaceAll(pattern, "*", "%"), "?", "_")
            wildcardConditions = append(wildcardConditions, "k8s_pod_name LIKE ?")
            args = append(args, likePattern)
        }
        conditions = append(conditions, s.combineConditions(wildcardConditions))
    }

    return conditions, args, nil
}

// combineConditions efficiently combines multiple conditions with OR logic
func (s *Service) combineConditions(conditions []string) string {
    if len(conditions) == 1 {
        return conditions[0]
    }
    return fmt.Sprintf("(%s)", strings.Join(conditions, " OR "))
}
```

### API Layer Enhancement for K8s Parameter Processing

**Enhanced API handler to support advanced K8s filtering parameters:**

```go
// Enhanced K8s parameter parsing in API handler
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

// Enhanced query parsing with K8s filtering support
func (h *LogHandler) parseQueryRequest(req *restful.Request, dataset string) (*request.LogQueryRequest, error) {
    // Parse existing parameters (time, dataset, etc. from Stories 2-1, 2-2)
    startTime, endTime, err := h.parseTimeParameters(req)
    if err != nil {
        return nil, err
    }

    // Parse K8s parameters with advanced patterns
    namespaces, pods, err := h.parseK8sParameters(req)
    if err != nil {
        return nil, fmt.Errorf("K8s parameter parsing failed: %w", err)
    }

    // Build enhanced request with K8s filtering
    queryReq := &request.LogQueryRequest{
        Dataset:   dataset,
        StartTime: startTime,
        EndTime:   endTime,

        // Legacy single parameters for backward compatibility
        Namespace: req.QueryParameter("namespace"),
        PodName:   req.QueryParameter("pod_name"),

        // Enhanced multi-value parameters
        Namespaces: namespaces,
        PodNames:   pods,

        // Other parameters
        Filter:        req.QueryParameter("filter"),
        Severity:      req.QueryParameter("severity"),
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

### Repository Layer ClickHouse Query Optimization for K8s

**Enhanced repository layer for optimal K8s metadata querying:**

```go
// Enhanced ClickHouse query execution with K8s optimization
func (r *ClickHouseRepository) QueryLogsWithK8sFilters(ctx context.Context, req *request.LogQueryRequest) ([]model.LogEntry, int, error) {
    // Build K8s-optimized query
    query, args, err := r.buildK8sOptimizedQuery(req)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to build K8s query: %w", err)
    }

    // Log query execution details for monitoring
    klog.InfoS("Executing K8s metadata query",
        "dataset", req.Dataset,
        "namespaces", req.Namespaces,
        "pods", req.PodNames,
        "estimated_selectivity", r.estimateK8sSelectivity(req))

    // Execute with context timeout
    queryCtx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
    defer cancel()

    startTime := time.Now()
    rows, err := r.conn.Query(queryCtx, query, args...)
    if err != nil {
        return nil, 0, fmt.Errorf("K8s query execution failed: %w", err)
    }
    defer rows.Close()

    // Parse results with K8s metadata
    logs, err := r.parseLogsWithK8sMetadata(rows)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to parse K8s logs: %w", err)
    }

    // Get total count for pagination
    totalCount, err := r.getTotalCountForK8sQuery(ctx, req)
    if err != nil {
        klog.ErrorS(err, "Failed to get K8s total count", "dataset", req.Dataset)
        totalCount = len(logs)
    }

    duration := time.Since(startTime)
    klog.InfoS("K8s metadata query completed",
        "dataset", req.Dataset,
        "result_count", len(logs),
        "total_count", totalCount,
        "duration_ms", duration.Milliseconds())

    // Performance monitoring for K8s queries
    if duration > 1*time.Second {
        klog.InfoS("Slow K8s query detected",
            "dataset", req.Dataset,
            "duration_ms", duration.Milliseconds(),
            "filter_complexity", len(req.K8sFilters))
    }

    return logs, totalCount, nil
}

// estimateK8sSelectivity provides query optimization hints
func (r *ClickHouseRepository) estimateK8sSelectivity(req *request.LogQueryRequest) float64 {
    selectivity := 1.0

    // Estimate selectivity based on filter types
    for _, filter := range req.K8sFilters {
        switch filter.Type {
        case K8sFilterExact:
            selectivity *= 0.01 // Exact matches are highly selective
        case K8sFilterPrefix:
            selectivity *= 0.1 // Prefix matches are moderately selective
        case K8sFilterRegex:
            selectivity *= 0.3 // Regex can vary widely
        case K8sFilterWildcard:
            selectivity *= 0.2 // Wildcards are moderately selective
        default:
            selectivity *= 0.5 // Conservative estimate
        }
    }

    return selectivity
}

// parseLogsWithK8sMetadata ensures K8s metadata is properly extracted
func (r *ClickHouseRepository) parseLogsWithK8sMetadata(rows driver.Rows) ([]model.LogEntry, error) {
    var logs []model.LogEntry

    for rows.Next() {
        var entry model.LogEntry
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
            &entry.ContainerID,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan K8s metadata row: %w", err)
        }

        entry.Timestamp = timestamp
        logs = append(logs, entry)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("K8s metadata row iteration error: %w", err)
    }

    return logs, nil
}

// getTotalCountForK8sQuery gets accurate count for K8s filtered queries
func (r *ClickHouseRepository) getTotalCountForK8sQuery(ctx context.Context, req *request.LogQueryRequest) (int, error) {
    // Build count query with same K8s filters
    countQuery, args, err := r.buildK8sCountQuery(req)
    if err != nil {
        return 0, fmt.Errorf("failed to build K8s count query: %w", err)
    }

    var count int64
    err = r.conn.QueryRow(ctx, countQuery, args...).Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("K8s count query failed: %w", err)
    }

    return int(count), nil
}

// buildK8sCountQuery creates optimized count query for K8s filters
func (r *ClickHouseRepository) buildK8sCountQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
    var whereConditions []string
    var args []interface{}

    // Use same filter building logic as main query
    whereConditions = append(whereConditions, "dataset = ?")
    args = append(args, req.Dataset)

    if req.StartTime != nil {
        whereConditions = append(whereConditions, "timestamp >= ?")
        args = append(args, *req.StartTime)
    }
    if req.EndTime != nil {
        whereConditions = append(whereConditions, "timestamp <= ?")
        args = append(args, *req.EndTime)
    }

    // Add K8s filter conditions
    k8sConditions, k8sArgs, err := r.buildK8sFilterConditionsForRepository(req.K8sFilters)
    if err != nil {
        return "", nil, err
    }
    whereConditions = append(whereConditions, k8sConditions...)
    args = append(args, k8sArgs...)

    query := fmt.Sprintf(`
        SELECT COUNT(*)
        FROM logs
        WHERE %s
    `, strings.Join(whereConditions, " AND "))

    return query, args, nil
}
```

### Performance Monitoring for K8s Metadata Queries

**Enhanced metrics collection for K8s filtering performance:**

```go
// K8sFilteringMetrics tracks performance of K8s metadata queries
type K8sFilteringMetrics struct {
    k8sQueryDuration   *prometheus.HistogramVec
    filterComplexity   *prometheus.HistogramVec
    k8sSelectivity     *prometheus.HistogramVec
    slowK8sQueries     *prometheus.CounterVec
    k8sFilterTypes     *prometheus.CounterVec
}

func NewK8sFilteringMetrics() *K8sFilteringMetrics {
    return &K8sFilteringMetrics{
        k8sQueryDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_k8s_query_duration_seconds",
                Help:    "Duration of K8s metadata filtered queries",
                Buckets: prometheus.DefBuckets,
            },
            []string{"dataset", "filter_count", "complexity_level"},
        ),
        filterComplexity: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_k8s_filter_complexity",
                Help:    "Complexity score of K8s filters",
                Buckets: []float64{1, 2, 5, 10, 20, 50},
            },
            []string{"dataset"},
        ),
        k8sSelectivity: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "edge_logs_k8s_query_selectivity",
                Help:    "Selectivity ratio of K8s queries (results/scanned)",
                Buckets: []float64{0.001, 0.01, 0.1, 0.25, 0.5, 0.75, 1.0},
            },
            []string{"dataset", "filter_types"},
        ),
        slowK8sQueries: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_slow_k8s_queries_total",
                Help: "Number of slow K8s filtered queries (>1s)",
            },
            []string{"dataset", "reason"},
        ),
        k8sFilterTypes: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "edge_logs_k8s_filter_types_total",
                Help: "Usage count of different K8s filter types",
            },
            []string{"filter_type", "field"},
        ),
    }
}

// RecordK8sQuery records metrics for K8s filtered queries
func (m *K8sFilteringMetrics) RecordK8sQuery(dataset string, duration time.Duration,
    filters []K8sFilter, resultCount, scannedCount int) {

    // Calculate complexity and filter metrics
    filterCount := len(filters)
    complexityLevel := m.categorizeComplexity(filters)
    filterTypes := m.categorizeFilterTypes(filters)

    // Record duration with context
    m.k8sQueryDuration.With(prometheus.Labels{
        "dataset":        dataset,
        "filter_count":   fmt.Sprintf("%d", filterCount),
        "complexity_level": complexityLevel,
    }).Observe(duration.Seconds())

    // Record filter complexity
    complexity := m.calculateComplexityScore(filters)
    m.filterComplexity.With(prometheus.Labels{
        "dataset": dataset,
    }).Observe(complexity)

    // Record selectivity
    selectivity := float64(resultCount) / float64(max(scannedCount, 1))
    m.k8sSelectivity.With(prometheus.Labels{
        "dataset":      dataset,
        "filter_types": filterTypes,
    }).Observe(selectivity)

    // Track filter type usage
    for _, filter := range filters {
        m.k8sFilterTypes.With(prometheus.Labels{
            "filter_type": string(filter.Type),
            "field":      filter.Field,
        }).Inc()
    }

    // Track slow queries
    if duration > 1*time.Second {
        reason := m.determineSlowestReason(duration, filterCount, selectivity)
        m.slowK8sQueries.With(prometheus.Labels{
            "dataset": dataset,
            "reason":  reason,
        }).Inc()
    }
}

// categorizeComplexity determines complexity level of K8s filters
func (m *K8sFilteringMetrics) categorizeComplexity(filters []K8sFilter) string {
    if len(filters) == 0 {
        return "none"
    }
    if len(filters) == 1 && filters[0].Type == K8sFilterExact {
        return "simple"
    }
    if len(filters) <= 3 {
        return "moderate"
    }
    return "complex"
}

// calculateComplexityScore provides numeric complexity scoring
func (m *K8sFilteringMetrics) calculateComplexityScore(filters []K8sFilter) float64 {
    score := 0.0
    for _, filter := range filters {
        switch filter.Type {
        case K8sFilterExact:
            score += 1.0
        case K8sFilterPrefix, K8sFilterSuffix:
            score += 2.0
        case K8sFilterContains:
            score += 3.0
        case K8sFilterWildcard:
            score += 4.0
        case K8sFilterRegex:
            score += 5.0
        }
    }
    return score
}
```

### API Documentation and Usage Examples

**Enhanced API documentation with advanced K8s filtering examples:**

```bash
# Basic namespace filtering (exact match)
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?start_time=2024-01-01T10:00:00Z&end_time=2024-01-01T10:30:00Z&namespace=kube-system"

# Multiple namespace filtering
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-cn-hz01/logs?namespaces=kube-system,default,monitoring&start_time=2024-01-01T10:00:00Z"

# Pod prefix filtering
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/staging-app/logs?pod_names=web-app-*,api-service-*&namespace=production"

# Advanced pod pattern matching
curl -k "https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?pods=regex:^app-.*-[0-9]+$,*worker*,icase:WEB-*&start_time=2024-01-01T09:00:00Z"

# Complex K8s filtering with time and content
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-cluster/logs?namespaces=kube-*,prod-*&pods=*api*,*worker*&filter=error&start_time=2024-01-01T08:00:00Z&end_time=2024-01-01T09:00:00Z"

# Response includes K8s metadata in structured format
{
  "items": [
    {
      "timestamp": "2024-01-01T10:15:30.123Z",
      "content": "Failed to connect to database",
      "severity": "ERROR",
      "k8s_namespace_name": "production",
      "k8s_pod_name": "web-app-deployment-abc123",
      "k8s_node_name": "edge-node-01",
      "host_ip": "10.0.1.15",
      "container_name": "web-app"
    }
  ],
  "dataset": "prod-cluster",
  "total": 156,
  "has_more": true,
  "query_info": {
    "k8s_filters_applied": 3,
    "namespaces_matched": ["production", "staging"],
    "pods_matched": ["web-app-*", "api-service-*"]
  }
}
```

### Error Handling for K8s Filtering Operations

**Comprehensive error handling for K8s filtering:**

```go
// K8s-specific error types
type K8sValidationError struct {
    Field       string
    Value       string
    Reason      string
    FilterType  K8sFilterType
}

func (e *K8sValidationError) Error() string {
    return fmt.Sprintf("K8s %s filter validation failed for '%s' (%s): %s",
        e.Field, e.Value, e.FilterType, e.Reason)
}

type K8sFilterComplexityError struct {
    FilterCount int
    MaxAllowed  int
}

func (e *K8sFilterComplexityError) Error() string {
    return fmt.Sprintf("K8s filter complexity too high: %d filters (max: %d)",
        e.FilterCount, e.MaxAllowed)
}

// Enhanced error handling in API layer
func (h *LogHandler) handleK8sError(resp *restful.Response, err error, dataset string) {
    switch e := err.(type) {
    case *K8sValidationError:
        errorResp := map[string]interface{}{
            "error":       "Invalid K8s filter",
            "field":       e.Field,
            "value":       e.Value,
            "filter_type": e.FilterType,
            "reason":      e.Reason,
            "dataset":     dataset,
            "supported_patterns": map[string][]string{
                "namespace": {
                    "exact: production",
                    "prefix: kube-*",
                    "wildcard: *-system",
                    "regex: regex:^[a-z]+-env$",
                },
                "pod": {
                    "exact: web-app-123",
                    "prefix: api-*",
                    "suffix: *-worker",
                    "contains: *database*",
                    "wildcard: web-??-prod",
                    "regex: regex:^app-[0-9]+$",
                    "case_insensitive: icase:WEB-*",
                },
            },
            "examples": []string{
                "namespace=production",
                "namespaces=kube-system,default",
                "pod_names=web-*,api-service-*",
                "pods=regex:^app-.*,*worker*",
            },
        }
        h.writeErrorResponse(resp, http.StatusBadRequest, errorResp)

    case *K8sFilterComplexityError:
        errorResp := map[string]interface{}{
            "error":         "K8s filter complexity too high",
            "filter_count":  e.FilterCount,
            "max_allowed":   e.MaxAllowed,
            "dataset":       dataset,
            "recommendation": "Use fewer filters or more specific patterns",
            "optimization_tips": []string{
                "Use exact matches when possible",
                "Combine multiple exact matches with comma separation",
                "Avoid complex regex patterns",
                "Use prefix/suffix patterns instead of contains",
            },
        }
        h.writeErrorResponse(resp, http.StatusBadRequest, errorResp)

    default:
        h.writeErrorResponse(resp, http.StatusBadRequest, "K8s filtering error: "+err.Error())
    }

    h.metrics.RecordK8sError(dataset, "validation_error")
}
```

### Testing Strategy for K8s Filtering

**Comprehensive testing strategy for K8s metadata filtering:**

1. **Unit Tests:**
   - K8s resource name validation (DNS-1123 compliance)
   - Filter pattern parsing and validation
   - Query building with various K8s filter combinations
   - Performance testing for filter complexity scoring
   - Edge cases with special K8s characters

2. **Integration Tests:**
   - End-to-end K8s filtering with real ClickHouse data
   - Multiple namespace and pod filtering combinations
   - Complex pattern matching (regex, wildcard, prefix)
   - Performance testing with large K8s metadata datasets
   - Error condition handling

3. **Performance Tests:**
   - K8s filter query performance benchmarking
   - Memory usage for complex K8s filter operations
   - Concurrent K8s filtered query handling
   - Large-scale K8s metadata query optimization

### Project Structure Notes

**File organization building on Stories 2-1, 2-2 foundation:**

```
pkg/service/query/
├── service.go                    # Enhanced with K8s filtering (modify existing)
├── k8s_validator.go              # K8s resource validation logic (new)
├── k8s_filter_builder.go         # K8s filter query building (new)
├── time_validator.go             # Existing from Story 2.2
├── dataset_validator.go          # Existing from Story 2.1
└── service_test.go              # Enhanced with K8s filtering tests (modify existing)

pkg/repository/clickhouse/
├── repository.go                # Enhanced with K8s query optimization (modify existing)
├── k8s_queries.go               # K8s-specific query patterns (new)
├── time_queries.go              # Existing from Story 2.2
└── repository_test.go           # Enhanced with K8s integration tests (modify existing)

pkg/oapis/log/v1alpha1/
├── handler.go                   # Enhanced with K8s parameter processing (modify existing)
├── k8s_errors.go                # K8s-specific error types (new)
├── k8s_metrics.go               # K8s filtering metrics (new)
├── time_errors.go               # Existing from Story 2.2
└── handler_test.go              # Enhanced with K8s endpoint tests (modify existing)

pkg/model/request/
└── log.go                       # Enhanced with K8s filtering fields (modify existing)

pkg/model/response/
└── log.go                       # Enhanced with K8s query metadata (modify existing)
```

**Key Integration Points:**
- Enhances existing pkg/service/query/service.go from Stories 2-1, 2-2 with K8s filtering
- Builds upon pkg/oapis/log/v1alpha1/handler.go parameter parsing capabilities
- Extends pkg/repository/clickhouse/repository.go with K8s-optimized queries
- Uses existing ClickHouse k8s_namespace_name and k8s_pod_name LowCardinality columns
- Integrates with dataset validation (Story 2-1) and time filtering (Story 2-2)

### Dependencies and Version Requirements

**No new dependencies required - leverages existing stack:**

```go
// Existing dependencies from previous stories
require (
    github.com/ClickHouse/clickhouse-go/v2 v2.15.0  // LowCardinality support
    github.com/emicklei/go-restful/v3 v3.11.0        # API framework
    k8s.io/klog/v2 v2.100.1                          # Structured logging
    github.com/prometheus/client_golang v1.17.0      # K8s filtering metrics
    github.com/stretchr/testify v1.8.4                # Testing framework
)
```

### Performance Requirements

**K8s metadata filtering performance targets:**

- **K8s Validation:** < 2ms per request for resource name validation and pattern parsing
- **Filter Building:** < 10ms additional latency for complex K8s filter query construction
- **ClickHouse Execution:** < 1 second for bounded K8s metadata queries with up to 20 filters
- **LowCardinality Optimization:** Effective utilization of ClickHouse LowCardinality columns
- **Memory Usage:** < 20MB additional memory for complex K8s filtering operations
- **Concurrent K8s Queries:** Support 200+ simultaneous K8s-filtered queries across different namespaces

### References

- [Source: _bmad-output/epics.md#Story 2.3] - Complete user story and acceptance criteria
- [Source: _bmad-output/architecture.md#K8s元数据] - K8s metadata schema with LowCardinality columns
- [Source: _bmad-output/2-2-time-range-filtering-with-millisecond-precision.md] - Foundation for K8s filtering enhancement
- [Source: _bmad-output/2-1-dataset-based-query-routing.md] - Dataset routing foundation
- [Source: sqlscripts/clickhouse/01_tables.sql#k8s_namespace_name] - ClickHouse K8s metadata schema
- [Source: pkg/model/request/log.go#K8sFilters] - K8s filter field definitions
- [Source: pkg/service/query/service.go] - Service layer to enhance with K8s filtering
- [Source: pkg/repository/clickhouse/repository.go] - Repository layer for K8s-optimized queries

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-20250514

### Debug Log References

Enhancing existing dataset routing and time filtering system (Stories 2-1, 2-2) to provide comprehensive Kubernetes-native log filtering capabilities. K8s filtering serves as the third filtering layer after dataset and time scoping, leveraging existing ClickHouse LowCardinality columns for optimal performance.

### Completion Notes List

Story 2-3 builds upon the completed dataset routing (Story 2-1) and time filtering (Story 2-2) foundations to implement advanced Kubernetes namespace and pod filtering. Enhances existing service, repository, and API layers with comprehensive K8s resource validation, multiple pattern matching (exact, prefix, regex, wildcard), and performance optimization. Provides flexible K8s-native filtering capabilities while maintaining sub-2 second query performance requirements and preparing for content-based search integration in Story 2-4.

### File List

Primary files to be enhanced/created:
- pkg/service/query/service.go (enhance with K8s filtering integration)
- pkg/service/query/k8s_validator.go (new)
- pkg/service/query/k8s_filter_builder.go (new)
- pkg/repository/clickhouse/repository.go (enhance with K8s query optimization)
- pkg/repository/clickhouse/k8s_queries.go (new)
- pkg/oapis/log/v1alpha1/handler.go (enhance with K8s parameter processing)
- pkg/oapis/log/v1alpha1/k8s_errors.go (new)
- pkg/oapis/log/v1alpha1/k8s_metrics.go (new)
- pkg/model/request/log.go (enhance with K8s filtering fields)
- pkg/model/response/log.go (enhance with K8s query metadata)
- pkg/service/query/service_test.go (enhance with K8s filtering tests)
- pkg/repository/clickhouse/repository_test.go (enhance with K8s integration tests)
- pkg/oapis/log/v1alpha1/handler_test.go (enhance with K8s endpoint tests)