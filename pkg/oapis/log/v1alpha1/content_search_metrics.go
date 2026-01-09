package v1alpha1

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// ContentSearchMetrics tracks performance of content search queries
type ContentSearchMetrics struct {
	contentQueryDuration    *prometheus.HistogramVec
	searchComplexity        *prometheus.HistogramVec
	contentMatchRate        *prometheus.HistogramVec
	slowContentQueries      *prometheus.CounterVec
	searchPatternTypes      *prometheus.CounterVec
	indexEfficiency         *prometheus.HistogramVec
	searchErrors            *prometheus.CounterVec
}

// NewContentSearchMetrics creates new content search metrics
func NewContentSearchMetrics() *ContentSearchMetrics {
	metrics := &ContentSearchMetrics{
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
		searchErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edge_logs_content_search_errors_total",
				Help: "Count of content search errors by type",
			},
			[]string{"dataset", "error_type"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		metrics.contentQueryDuration,
		metrics.searchComplexity,
		metrics.contentMatchRate,
		metrics.slowContentQueries,
		metrics.searchPatternTypes,
		metrics.indexEfficiency,
		metrics.searchErrors,
	)

	klog.InfoS("Content search metrics initialized")
	return metrics
}

// RecordContentSearchQuery records metrics for content search queries
func (m *ContentSearchMetrics) RecordContentSearchQuery(dataset string, duration time.Duration,
	patterns []string, resultCount, totalScanned int) {

	if len(patterns) == 0 {
		return
	}

	// Calculate metrics
	complexityLevel := m.categorizeComplexity(patterns)
	patternCount := len(patterns)
	complexity := m.calculateComplexity(patterns)
	matchRate := float64(resultCount) / float64(max(totalScanned, 1))

	// Record duration with context
	m.contentQueryDuration.With(prometheus.Labels{
		"dataset":         dataset,
		"complexity_level": complexityLevel,
		"pattern_count":   string(rune('0' + patternCount)),
	}).Observe(duration.Seconds())

	// Record complexity
	m.searchComplexity.With(prometheus.Labels{
		"dataset": dataset,
	}).Observe(complexity)

	// Record match rate by search type
	searchType := m.categorizeSearchType(patterns)
	m.contentMatchRate.With(prometheus.Labels{
		"dataset":     dataset,
		"search_type": searchType,
	}).Observe(matchRate)

	// Track pattern type usage
	for _, pattern := range patterns {
		patternType := m.detectPatternType(pattern)
		m.searchPatternTypes.With(prometheus.Labels{
			"pattern_type": patternType,
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

	// Estimate index efficiency
	indexEfficiency := m.estimateIndexEfficiency(patterns, resultCount, totalScanned)
	m.indexEfficiency.With(prometheus.Labels{
		"dataset":    dataset,
		"index_type": "tokenbf_v1",
	}).Observe(indexEfficiency)
}

// RecordSearchError records search error metrics
func (m *ContentSearchMetrics) RecordSearchError(dataset, errorType string) {
	m.searchErrors.With(prometheus.Labels{
		"dataset":    dataset,
		"error_type": errorType,
	}).Inc()
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

// determineSlowReason determines why a query was slow
func (m *ContentSearchMetrics) determineSlowReason(duration time.Duration, complexity, matchRate float64) string {
	switch {
	case complexity > 50:
		return "high_complexity"
	case matchRate > 0.5:
		return "high_match_rate"
	case duration > 5*time.Second:
		return "very_slow"
	default:
		return "slow"
	}
}

// estimateIndexEfficiency provides index utilization scoring
func (m *ContentSearchMetrics) estimateIndexEfficiency(patterns []string, resultCount, totalScanned int) float64 {
	if totalScanned == 0 {
		return 1.0
	}

	// Base efficiency from selectivity
	baseEfficiency := float64(resultCount) / float64(totalScanned)

	// Adjust based on search patterns (some patterns benefit more from indexing)
	indexFriendliness := 1.0
	for _, pattern := range patterns {
		patternType := m.detectPatternType(pattern)
		switch patternType {
		case "exact":
			indexFriendliness *= 1.2 // Exact matches benefit most from indexing
		case "case_insensitive":
			indexFriendliness *= 1.1
		case "wildcard":
			indexFriendliness *= 0.8 // Wildcards are less index-friendly
		case "regex":
			indexFriendliness *= 0.6 // Regex can be expensive
		case "phrase":
			indexFriendliness *= 1.0 // Neutral
		case "proximity":
			indexFriendliness *= 0.7 // Complex proximity calculations
		}
	}

	efficiency := baseEfficiency * indexFriendliness
	if efficiency > 1.0 {
		return 1.0
	}
	return efficiency
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}