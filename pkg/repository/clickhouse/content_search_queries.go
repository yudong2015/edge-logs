package clickhouse

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/model/search"
)

// ContentSearchQueryBuilder builds optimized ClickHouse queries for content search
type ContentSearchQueryBuilder struct {
	// tokenbf_v1 optimization settings
	tokenBFOptimization bool
	maxTokenLength      int
	skipGramSize        int
}

// NewContentSearchQueryBuilder creates a new content search query builder
func NewContentSearchQueryBuilder() *ContentSearchQueryBuilder {
	return &ContentSearchQueryBuilder{
		tokenBFOptimization: true, // Enable tokenbf_v1 index utilization
		maxTokenLength:      64,   // Maximum token length for bloom filter
		skipGramSize:        8,    // Skip-gram size for tokenbf_v1
	}
}

// BuildContentSearchQuery creates an optimized ClickHouse query for content search
func (b *ContentSearchQueryBuilder) BuildContentSearchQuery(req *request.LogQueryRequest, contentSearch *search.ContentSearchExpression) (string, []interface{}, error) {
	if contentSearch == nil || len(contentSearch.Filters) == 0 {
		// Fallback to basic query without content search
		return b.buildBasicQuery(req)
	}

	klog.V(4).InfoS("Building content search query",
		"dataset", req.Dataset,
		"filters", len(contentSearch.Filters),
		"highlight", contentSearch.HighlightEnabled,
		"relevance", contentSearch.RelevanceScoring)

	// Build SELECT clause with highlighting and relevance
	selectClause := b.buildSelectClause(contentSearch)

	// Build WHERE clause with content search conditions
	whereConditions, args, err := b.buildWhereClause(req, contentSearch)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
	}

	// Build ORDER BY clause with relevance scoring
	orderByClause := b.buildOrderByClause(contentSearch)

	// Build LIMIT and OFFSET
	limitClause := fmt.Sprintf("LIMIT %d OFFSET %d", req.PageSize, req.Page*req.PageSize)

	// Construct final query (unified logs table)
	query := fmt.Sprintf(`
		%s
		FROM ` + "`logs_mv`" + `
		WHERE %s
		%s
		%s
	`, selectClause, strings.Join(whereConditions, " AND "), orderByClause, limitClause)

	klog.V(4).InfoS("Content search query built",
		"dataset", req.Dataset,
		"query_length", len(query),
		"args_count", len(args))

	return strings.TrimSpace(query), args, nil
}

// BuildContentSearchCountQuery creates a count query for pagination
func (b *ContentSearchQueryBuilder) BuildContentSearchCountQuery(req *request.LogQueryRequest, contentSearch *search.ContentSearchExpression) (string, []interface{}, error) {
	if contentSearch == nil || len(contentSearch.Filters) == 0 {
		return b.buildBasicCountQuery(req)
	}

	// Build WHERE clause with content search conditions
	whereConditions, args, err := b.buildWhereClause(req, contentSearch)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build WHERE clause for count: %w", err)
	}

	// Construct count query (unified logs table)
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM ` + "`logs_mv`" + `
		WHERE %s
	`, strings.Join(whereConditions, " AND "))

	return strings.TrimSpace(query), args, nil
}

// buildSelectClause creates the SELECT clause with highlighting and relevance scoring (OTEL format)
func (b *ContentSearchQueryBuilder) buildSelectClause(contentSearch *search.ContentSearchExpression) string {
	baseFields := []string{
		"Timestamp",
		"SeverityText",
		"SeverityNumber",
		"ServiceName",
		"Content",
		"pod_name",
		"namespace_name",
		"container_name",
		"container_id",
	}

	selectFields := baseFields

	// Add highlighting fields if enabled
	if contentSearch.HighlightEnabled {
		highlightFields := b.buildHighlightingFields(contentSearch)
		selectFields = append(selectFields, highlightFields...)
	}

	// Add relevance scoring if enabled
	if contentSearch.RelevanceScoring {
		relevanceField := b.buildRelevanceScoring(contentSearch)
		if relevanceField != "" {
			selectFields = append(selectFields, relevanceField)
		}
	}

	return "SELECT " + strings.Join(selectFields, ",\n       ")
}

// buildWhereClause creates the WHERE clause with ServiceName, time, K8s, and content conditions (OTEL format)
func (b *ContentSearchQueryBuilder) buildWhereClause(req *request.LogQueryRequest, contentSearch *search.ContentSearchExpression) ([]string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	// Dataset condition
	if req.Dataset != "" {
		conditions = append(conditions, "dataset = ?")
		args = append(args, req.Dataset)
	}

	// Time range conditions
	if req.StartTime != nil {
		conditions = append(conditions, "Timestamp >= ?")
		args = append(args, *req.StartTime)
	}
	if req.EndTime != nil {
		conditions = append(conditions, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	// Legacy filter handling for backward compatibility
	if req.Filter != "" && len(contentSearch.Filters) == 0 {
		conditions = append(conditions, "positionCaseInsensitive(Content, ?) > 0")
		args = append(args, req.Filter)
	}

	// Severity filtering
	if req.Severity != "" {
		conditions = append(conditions, "SeverityText = ?")
		args = append(args, req.Severity)
	}

	// K8s metadata conditions
	k8sConditions, k8sArgs := b.buildK8sConditions(req)
	conditions = append(conditions, k8sConditions...)
	args = append(args, k8sArgs...)

	// Content search conditions
	contentConditions, contentArgs, err := b.buildContentSearchConditions(contentSearch)
	if err != nil {
		return nil, nil, err
	}
	conditions = append(conditions, contentConditions...)
	args = append(args, contentArgs...)

	return conditions, args, nil
}

// buildK8sConditions creates K8s metadata filtering conditions (OTEL format)
func (b *ContentSearchQueryBuilder) buildK8sConditions(req *request.LogQueryRequest) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

	// Namespace support (from LogAttributes)
	if req.Namespace != "" {
		conditions = append(conditions, "LogAttributes['k8s.namespace.name'] = ?")
		args = append(args, req.Namespace)
	}

	// Pod name support (from LogAttributes)
	if req.PodName != "" {
		conditions = append(conditions, "LogAttributes['k8s.pod.name'] = ?")
		args = append(args, req.PodName)
	}

	// Node name support (from LogAttributes)
	if req.NodeName != "" {
		conditions = append(conditions, "LogAttributes['k8s.node.name'] = ?")
		args = append(args, req.NodeName)
	}

	// Container name support (from LogAttributes)
	if req.ContainerName != "" {
		conditions = append(conditions, "LogAttributes['k8s.container.name'] = ?")
		args = append(args, req.ContainerName)
	}

	// Host filtering (from ResourceAttributes)
	if req.HostIP != "" {
		conditions = append(conditions, "ResourceAttributes['host.ip'] = ?")
		args = append(args, req.HostIP)
	}
	if req.HostName != "" {
		conditions = append(conditions, "ResourceAttributes['host.name'] = ?")
		args = append(args, req.HostName)
	}

	// Enhanced K8s filters from parsed filters
	if len(req.K8sFilters) > 0 {
		for _, filter := range req.K8sFilters {
			condition, arg := b.buildSingleK8sCondition(filter)
			if condition != "" {
				conditions = append(conditions, condition)
				args = append(args, arg...)
			}
		}
	}

	return conditions, args
}

// buildSingleK8sCondition creates a condition for a single K8s filter (OTEL format)
func (b *ContentSearchQueryBuilder) buildSingleK8sCondition(filter request.K8sFilter) (string, []interface{}) {
	var condition string
	var args []interface{}

	fieldName := ""
	switch filter.Field {
	case "namespace":
		fieldName = "LogAttributes['k8s.namespace.name']"
	case "pod":
		fieldName = "LogAttributes['k8s.pod.name']"
	default:
		return "", nil
	}

	switch filter.Type {
	case request.K8sFilterExact:
		if filter.CaseInsensitive {
			condition = fmt.Sprintf("lower(%s) = lower(?)", fieldName)
		} else {
			condition = fmt.Sprintf("%s = ?", fieldName)
		}
		args = append(args, filter.Pattern)

	case request.K8sFilterPrefix:
		if filter.CaseInsensitive {
			condition = fmt.Sprintf("lower(%s) LIKE lower(?)", fieldName)
			args = append(args, filter.Pattern+"%")
		} else {
			condition = fmt.Sprintf("%s LIKE ?", fieldName)
			args = append(args, filter.Pattern+"%")
		}

	case request.K8sFilterSuffix:
		if filter.CaseInsensitive {
			condition = fmt.Sprintf("lower(%s) LIKE lower(?)", fieldName)
			args = append(args, "%"+filter.Pattern)
		} else {
			condition = fmt.Sprintf("%s LIKE ?", fieldName)
			args = append(args, "%"+filter.Pattern)
		}

	case request.K8sFilterContains:
		if filter.CaseInsensitive {
			condition = fmt.Sprintf("positionCaseInsensitive(%s, ?) > 0", fieldName)
		} else {
			condition = fmt.Sprintf("position(%s, ?) > 0", fieldName)
		}
		args = append(args, filter.Pattern)

	case request.K8sFilterWildcard:
		// Convert wildcard pattern to SQL LIKE pattern
		likePattern := strings.ReplaceAll(strings.ReplaceAll(filter.Pattern, "*", "%"), "?", "_")
		if filter.CaseInsensitive {
			condition = fmt.Sprintf("lower(%s) LIKE lower(?)", fieldName)
		} else {
			condition = fmt.Sprintf("%s LIKE ?", fieldName)
		}
		args = append(args, likePattern)

	case request.K8sFilterRegex:
		condition = fmt.Sprintf("match(%s, ?)", fieldName)
		args = append(args, filter.Pattern)
	}

	return condition, args
}

// buildContentSearchConditions creates optimized content search conditions
func (b *ContentSearchQueryBuilder) buildContentSearchConditions(contentSearch *search.ContentSearchExpression) ([]string, []interface{}, error) {
	if contentSearch == nil || len(contentSearch.Filters) == 0 {
		return nil, nil, nil
	}

	var conditions []string
	var args []interface{}

	// Group filters by boolean operators
	andFilters := []search.ContentSearchFilter{}
	orFilters := []search.ContentSearchFilter{}
	notFilters := []search.ContentSearchFilter{}

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
		andConditions, andArgs, err := b.buildFilterConditions(andFilters)
		if err != nil {
			return nil, nil, err
		}
		conditions = append(conditions, andConditions...)
		args = append(args, andArgs...)
	}

	// Build OR conditions
	if len(orFilters) > 0 {
		orConditions, orArgs, err := b.buildFilterConditions(orFilters)
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
		notConditions, notArgs, err := b.buildFilterConditions(notFilters)
		if err != nil {
			return nil, nil, err
		}
		for _, notCondition := range notConditions {
			conditions = append(conditions, fmt.Sprintf("NOT (%s)", notCondition))
		}
		args = append(args, notArgs...)
	}

	return conditions, args, nil
}

// buildFilterConditions creates ClickHouse conditions for specific filter types
func (b *ContentSearchQueryBuilder) buildFilterConditions(filters []search.ContentSearchFilter) ([]string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	for _, filter := range filters {
		condition, filterArgs, err := b.buildSingleFilterCondition(filter)
		if err != nil {
			return nil, nil, err
		}
		conditions = append(conditions, condition)
		args = append(args, filterArgs...)
	}

	return conditions, args, nil
}

// buildSingleFilterCondition creates ClickHouse condition for a single filter with tokenbf_v1 optimization
func (b *ContentSearchQueryBuilder) buildSingleFilterCondition(filter search.ContentSearchFilter) (string, []interface{}, error) {
	var condition string
	var args []interface{}

	switch filter.Type {
	case search.ContentSearchExact:
		if filter.CaseInsensitive {
			condition = "positionCaseInsensitive(Content, ?) > 0"
		} else {
			// Use position for exact matches - benefits from tokenbf_v1 index
			condition = "position(Content, ?) > 0"
		}
		args = append(args, filter.Pattern)

	case search.ContentSearchCaseInsensitive:
		condition = "positionCaseInsensitive(Content, ?) > 0"
		args = append(args, filter.Pattern)

	case search.ContentSearchRegex:
		// Use match function for regex - can benefit from tokenbf_v1 for literal parts
		condition = "match(Content, ?)"
		args = append(args, filter.Pattern)

	case search.ContentSearchWildcard:
		// Convert wildcard pattern to SQL LIKE pattern
		likePattern := strings.ReplaceAll(strings.ReplaceAll(filter.Pattern, "*", "%"), "?", "_")
		if filter.CaseInsensitive {
			condition = "Body ILIKE ?"
		} else {
			condition = "Body LIKE ?"
		}
		args = append(args, likePattern)

	case search.ContentSearchPhrase:
		// Use exact phrase matching with word boundaries
		phrasePattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(filter.Pattern))
		condition = "match(Content, ?)"
		args = append(args, phrasePattern)

	case search.ContentSearchProximity:
		// Build proximity search using ClickHouse functions
		terms := strings.Fields(filter.Pattern)
		if len(terms) < 2 {
			return "", nil, fmt.Errorf("proximity search requires at least 2 terms")
		}

		// Create a complex condition for proximity search
		// This is a simplified version - a full implementation would use more sophisticated algorithms
		var proximityConditions []string
		for _, term := range terms {
			proximityConditions = append(proximityConditions, fmt.Sprintf("position(Content, '%s') > 0", strings.ReplaceAll(term, "'", "''")))
		}

		// For now, just ensure all terms are present (simple implementation)
		condition = strings.Join(proximityConditions, " AND ")

	case search.ContentSearchBoolean:
		return "", nil, fmt.Errorf("boolean search should be decomposed before reaching this level")

	default:
		return "", nil, fmt.Errorf("unsupported content search type: %s", filter.Type)
	}

	return condition, args, nil
}

// buildHighlightingFields creates highlighting expressions for matched content
func (b *ContentSearchQueryBuilder) buildHighlightingFields(contentSearch *search.ContentSearchExpression) []string {
	var highlightFields []string

	for i, filter := range contentSearch.Filters {
		switch filter.Type {
		case search.ContentSearchExact, search.ContentSearchCaseInsensitive:
			// Use replaceRegexpAll for highlighting
			highlightExpr := fmt.Sprintf(
				"replaceRegexpAll(Body, '(?i)(%s)', '<mark class=\"highlight-%d\">$1</mark>') AS highlighted_content_%d",
				regexp.QuoteMeta(filter.Pattern), i, i)
			highlightFields = append(highlightFields, highlightExpr)

		case search.ContentSearchRegex:
			highlightExpr := fmt.Sprintf(
				"replaceRegexpAll(Body, '(%s)', '<mark class=\"highlight-%d\">$1</mark>') AS highlighted_content_%d",
				filter.Pattern, i, i)
			highlightFields = append(highlightFields, highlightExpr)

		case search.ContentSearchPhrase:
			phrasePattern := regexp.QuoteMeta(filter.Pattern)
			highlightExpr := fmt.Sprintf(
				"replaceRegexpAll(Body, '(?i)(%s)', '<mark class=\"highlight-%d\">$1</mark>') AS highlighted_content_%d",
				phrasePattern, i, i)
			highlightFields = append(highlightFields, highlightExpr)
		}
	}

	return highlightFields
}

// buildRelevanceScoring creates relevance scoring expression
func (b *ContentSearchQueryBuilder) buildRelevanceScoring(contentSearch *search.ContentSearchExpression) string {
	if !contentSearch.RelevanceScoring || len(contentSearch.Filters) == 0 {
		return ""
	}

	var scoreComponents []string
	for _, filter := range contentSearch.Filters {
		var scoreExpr string
		switch filter.Type {
		case search.ContentSearchExact, search.ContentSearchCaseInsensitive:
			scoreExpr = fmt.Sprintf("(position(Content, '%s') > 0 ? %f : 0)",
				strings.ReplaceAll(filter.Pattern, "'", "''"), filter.Weight)
		case search.ContentSearchPhrase:
			phrasePattern := strings.ReplaceAll(filter.Pattern, "'", "''")
			scoreExpr = fmt.Sprintf("(match(Content, '\\b%s\\b') ? %f : 0)",
				phrasePattern, filter.Weight)
		case search.ContentSearchProximity:
			// Higher score for proximity matches
			scoreExpr = fmt.Sprintf("(position(Content, '%s') > 0 ? %f : 0)",
				strings.ReplaceAll(strings.Fields(filter.Pattern)[0], "'", "''"), filter.Weight)
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

// buildOrderByClause adds relevance-based ordering
func (b *ContentSearchQueryBuilder) buildOrderByClause(contentSearch *search.ContentSearchExpression) string {
	if contentSearch != nil && contentSearch.RelevanceScoring && len(contentSearch.Filters) > 0 {
		return "ORDER BY search_relevance_score DESC, timestamp DESC"
	}
	return "ORDER BY timestamp DESC"
}

// buildBasicQuery creates a basic query without content search (OTEL format)
func (b *ContentSearchQueryBuilder) buildBasicQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	// Dataset condition
	if req.Dataset != "" {
		conditions = append(conditions, "dataset = ?")
		args = append(args, req.Dataset)
	}

	// Time range conditions
	if req.StartTime != nil {
		conditions = append(conditions, "Timestamp >= ?")
		args = append(args, *req.StartTime)
	}
	if req.EndTime != nil {
		conditions = append(conditions, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	// Basic filter
	if req.Filter != "" {
		conditions = append(conditions, "positionCaseInsensitive(Content, ?) > 0")
		args = append(args, req.Filter)
	}

	query := fmt.Sprintf(`
		SELECT Timestamp,
		       SeverityText, SeverityNumber, ServiceName,
		       Content AS Body,
		       pod_name,
		       namespace_name,
		       container_name,
		       container_id
		FROM ` + "`logs_mv`" + `
		WHERE %s
		ORDER BY Timestamp DESC
		LIMIT %d OFFSET %d
	`, strings.Join(conditions, " AND "), req.PageSize, req.Page*req.PageSize)

	return strings.TrimSpace(query), args, nil
}

// buildBasicCountQuery creates a basic count query without content search (OTEL format)
func (b *ContentSearchQueryBuilder) buildBasicCountQuery(req *request.LogQueryRequest) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	// Dataset condition
	if req.Dataset != "" {
		conditions = append(conditions, "dataset = ?")
		args = append(args, req.Dataset)
	}

	// Time range conditions
	if req.StartTime != nil {
		conditions = append(conditions, "Timestamp >= ?")
		args = append(args, *req.StartTime)
	}
	if req.EndTime != nil {
		conditions = append(conditions, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	// Basic filter
	if req.Filter != "" {
		conditions = append(conditions, "positionCaseInsensitive(Content, ?) > 0")
		args = append(args, req.Filter)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM ` + "`logs_mv`" + `
		WHERE %s
	`, strings.Join(conditions, " AND "))

	return strings.TrimSpace(query), args, nil
}