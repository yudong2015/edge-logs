package query

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/search"
)

// Type aliases for backward compatibility
type ContentSearchType = search.ContentSearchType
type ContentSearchFilter = search.ContentSearchFilter
type ContentSearchExpression = search.ContentSearchExpression

// Constants from search package
const (
	ContentSearchExact        = search.ContentSearchExact
	ContentSearchCaseInsensitive = search.ContentSearchCaseInsensitive
	ContentSearchRegex        = search.ContentSearchRegex
	ContentSearchWildcard     = search.ContentSearchWildcard
	ContentSearchPhrase       = search.ContentSearchPhrase
	ContentSearchProximity    = search.ContentSearchProximity
	ContentSearchBoolean      = search.ContentSearchBoolean
)

// ContentSearchValidator validates and parses content search expressions
type ContentSearchValidator struct {
	maxSearchTerms     int
	maxPatternLength   int
	maxQueryComplexity int
	safetyPatterns     []*regexp.Regexp
}

// NewContentSearchValidator creates a new content search validator
func NewContentSearchValidator() *ContentSearchValidator {
	return &ContentSearchValidator{
		maxSearchTerms:     20,  // Prevent overly complex queries
		maxPatternLength:   500, // Prevent extremely long search patterns
		maxQueryComplexity: 100, // Complex scoring threshold
		safetyPatterns: []*regexp.Regexp{
			// Prevent expensive regex patterns
			regexp.MustCompile(`\.\*\.\*`),       // Multiple greedy quantifiers
			regexp.MustCompile(`\.\+\.\+`),       // Multiple possessive quantifiers
			regexp.MustCompile(`\(\?\<\!\.\*`),   // Negative lookbehind with .*
			regexp.MustCompile(`\.\*\.\+\.\*`),   // Mixed greedy patterns
			regexp.MustCompile(`\(\?\!\.\*\)`),   // Negative lookahead with .*
		},
	}
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

	klog.V(4).InfoS("Content search parsed successfully",
		"query", searchQuery,
		"filters", len(expression.Filters),
		"complexity", v.calculateComplexity(expression))

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
			return nil, fmt.Errorf("unsafe regex pattern detected: potential performance impact")
		}
	}

	// Test regex compilation
	_, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	expression.Filters = []ContentSearchFilter{
		{
			Type:    ContentSearchRegex,
			Pattern: pattern,
			Weight:  1.0,
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

// tokenizeBooleanQuery tokenizes boolean query expressions
func (v *ContentSearchValidator) tokenizeBooleanQuery(query string) ([]string, error) {
	// Simple tokenization for boolean expressions
	// This is a simplified implementation - a full implementation would use a proper parser
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty boolean query")
	}

	// Basic validation of boolean operators
	for i, token := range tokens {
		switch strings.ToUpper(token) {
		case "AND", "OR", "NOT":
			if i == 0 && strings.ToUpper(token) != "NOT" {
				return nil, fmt.Errorf("boolean query cannot start with operator: %s", token)
			}
			if i == len(tokens)-1 {
				return nil, fmt.Errorf("boolean query cannot end with operator: %s", token)
			}
		}
	}

	return tokens, nil
}

// convertTokensToFilters converts tokenized boolean expression to filters
func (v *ContentSearchValidator) convertTokensToFilters(tokens []string) ([]ContentSearchFilter, error) {
	var filters []ContentSearchFilter
	currentOperator := "AND"

	for _, token := range tokens {
		switch strings.ToUpper(token) {
		case "AND", "OR", "NOT":
			currentOperator = strings.ToUpper(token)
		case "(", ")":
			// Handle parentheses - simplified for now
			continue
		default:
			// This is a search term
			filters = append(filters, ContentSearchFilter{
				Type:            ContentSearchExact,
				Pattern:         token,
				BooleanOperator: currentOperator,
				Weight:          1.0,
			})
		}
	}

	return filters, nil
}

// extractQuotedPhrases extracts quoted phrases from query
func (v *ContentSearchValidator) extractQuotedPhrases(query string) []string {
	var phrases []string
	inQuotes := false
	currentPhrase := ""

	for _, char := range query {
		if char == '"' {
			if inQuotes {
				// End of phrase
				if currentPhrase != "" {
					phrases = append(phrases, currentPhrase)
					currentPhrase = ""
				}
				inQuotes = false
			} else {
				// Start of phrase
				inQuotes = true
			}
		} else if inQuotes {
			currentPhrase += string(char)
		}
	}

	return phrases
}

// extractOperators extracts boolean operators between phrases
func (v *ContentSearchValidator) extractOperators(query string) []string {
	// Simplified operator extraction between phrases
	operators := []string{}
	if strings.Contains(query, " OR ") {
		operators = append(operators, "OR")
	}
	if strings.Contains(query, " AND ") {
		operators = append(operators, "AND")
	}
	if strings.Contains(query, " NOT ") {
		operators = append(operators, "NOT")
	}
	return operators
}

// validateComplexity ensures search complexity is within acceptable bounds
func (v *ContentSearchValidator) validateComplexity(expression *ContentSearchExpression) error {
	complexity := v.calculateComplexity(expression)
	if complexity > float64(v.maxQueryComplexity) {
		return fmt.Errorf("search complexity (%.1f) exceeds maximum (%d)", complexity, v.maxQueryComplexity)
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

	// Boolean complexity multiplier
	if len(expression.Filters) > 1 {
		complexity *= 1.5
	}

	return complexity
}

// ValidateContentSearchExpression validates a complete content search expression
func (v *ContentSearchValidator) ValidateContentSearchExpression(expression *ContentSearchExpression) error {
	if expression == nil {
		return nil
	}

	if len(expression.Filters) == 0 {
		return nil
	}

	// Validate each filter
	for i, filter := range expression.Filters {
		if err := v.validateSingleFilter(filter); err != nil {
			return fmt.Errorf("filter %d validation failed: %w", i, err)
		}
	}

	// Validate overall complexity
	if err := v.validateComplexity(expression); err != nil {
		return err
	}

	return nil
}

// validateSingleFilter validates a single content search filter
func (v *ContentSearchValidator) validateSingleFilter(filter ContentSearchFilter) error {
	if filter.Pattern == "" {
		return fmt.Errorf("search pattern cannot be empty")
	}

	if len(filter.Pattern) > v.maxPatternLength {
		return fmt.Errorf("search pattern too long (%d chars), max: %d", len(filter.Pattern), v.maxPatternLength)
	}

	// Validate regex patterns
	if filter.Type == ContentSearchRegex {
		for _, safetyPattern := range v.safetyPatterns {
			if safetyPattern.MatchString(filter.Pattern) {
				return fmt.Errorf("unsafe regex pattern detected")
			}
		}

		if _, err := regexp.Compile(filter.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Validate proximity distance
	if filter.Type == ContentSearchProximity {
		if filter.ProximityDistance <= 0 || filter.ProximityDistance > 50 {
			return fmt.Errorf("proximity distance must be between 1 and 50")
		}
	}

	// Validate boolean operators
	if filter.BooleanOperator != "" {
		switch filter.BooleanOperator {
		case "AND", "OR", "NOT":
			// Valid
		default:
			return fmt.Errorf("invalid boolean operator: %s", filter.BooleanOperator)
		}
	}

	return nil
}