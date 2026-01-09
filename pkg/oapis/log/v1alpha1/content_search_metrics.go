package v1alpha1

import (
	"time"

	"k8s.io/klog/v2"
)

// ContentSearchMetrics tracks performance of content search queries
type ContentSearchMetrics struct {
	// Metrics are tracked internally for logging purposes
	// Prometheus metrics can be added later when dependency is available
}

// NewContentSearchMetrics creates new content search metrics
func NewContentSearchMetrics() *ContentSearchMetrics {
	klog.InfoS("Content search metrics initialized")
	return &ContentSearchMetrics{}
}

// RecordContentSearchQuery records metrics for content search queries
func (m *ContentSearchMetrics) RecordContentSearchQuery(dataset string, duration time.Duration,
	patterns []string, resultCount, totalScanned int) {

	if len(patterns) == 0 {
		return
	}

	complexityLevel := m.categorizeComplexity(patterns)
	searchType := m.categorizeSearchType(patterns)

	klog.V(4).InfoS("Content search query recorded",
		"dataset", dataset,
		"duration_ms", duration.Milliseconds(),
		"complexity_level", complexityLevel,
		"search_type", searchType,
		"result_count", resultCount,
		"total_scanned", totalScanned)

	// Track slow queries
	if duration > 2*time.Second {
		klog.InfoS("Slow content search query detected",
			"dataset", dataset,
			"duration_ms", duration.Milliseconds(),
			"pattern_count", len(patterns))
	}
}

// RecordSearchError records search error metrics
func (m *ContentSearchMetrics) RecordSearchError(dataset, errorType string) {
	klog.V(4).InfoS("Content search error recorded",
		"dataset", dataset,
		"error_type", errorType)
}

// categorizeComplexity determines complexity level of content search
func (m *ContentSearchMetrics) categorizeComplexity(patterns []string) string {
	complexity := m.calculateComplexity(patterns)
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
func (m *ContentSearchMetrics) calculateComplexity(patterns []string) float64 {
	complexity := 0.0
	for _, pattern := range patterns {
		switch {
		case len(pattern) <= 5:
			complexity += 1.0
		case len(pattern) <= 20:
			complexity += 2.0
		case len(pattern) <= 50:
			complexity += 4.0
		default:
			complexity += 8.0
		}

		// Add complexity for special patterns
		patternType := m.detectPatternType(pattern)
		switch patternType {
		case "regex":
			complexity += 5.0
		case "wildcard":
			complexity += 3.0
		case "phrase":
			complexity += 2.0
		case "boolean":
			complexity += 6.0
		case "proximity":
			complexity += 4.0
		}
	}

	// Boolean complexity multiplier
	if len(patterns) > 1 {
		complexity *= 1.5
	}

	return complexity
}

// categorizeSearchType determines the primary search type
func (m *ContentSearchMetrics) categorizeSearchType(patterns []string) string {
	if len(patterns) == 0 {
		return "empty"
	}

	// Check for complex patterns first
	for _, pattern := range patterns {
		patternType := m.detectPatternType(pattern)
		if patternType != "exact" {
			return patternType
		}
	}

	if len(patterns) > 1 {
		return "multi_term"
	}

	return "exact"
}

// detectPatternType detects the type of a search pattern
func (m *ContentSearchMetrics) detectPatternType(pattern string) string {
	switch {
	case len(pattern) > 8 && pattern[:8] == "boolean:":
		return "boolean"
	case len(pattern) > 6 && pattern[:6] == "regex:":
		return "regex"
	case len(pattern) > 10 && pattern[:10] == "proximity:":
		return "proximity"
	case len(pattern) > 6 && pattern[:6] == "icase:":
		return "case_insensitive"
	case len(pattern) >= 2 && pattern[0] == '"' && pattern[len(pattern)-1] == '"':
		return "phrase"
	case containsWildcards(pattern):
		return "wildcard"
	default:
		return "exact"
	}
}

// containsWildcards checks if pattern contains wildcard characters
func containsWildcards(pattern string) bool {
	for _, char := range pattern {
		if char == '*' || char == '?' {
			return true
		}
	}
	return false
}
