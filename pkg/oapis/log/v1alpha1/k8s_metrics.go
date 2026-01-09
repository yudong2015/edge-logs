package v1alpha1

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// K8sMetrics tracks performance and usage of K8s metadata queries using simple counters
type K8sMetrics struct {
	mu                    sync.RWMutex
	totalQueries          int64
	totalErrors           int64
	slowQueries           int64
	totalDuration         time.Duration
	filterTypeUsage       map[string]int64
	errorTypeCount        map[string]int64
	namespaceFilterCount  int64
	podFilterCount        int64
	complexitySum         float64
	selectivitySum        float64
}

// NewK8sMetrics creates a new K8s metrics collector
func NewK8sMetrics() *K8sMetrics {
	return &K8sMetrics{
		filterTypeUsage: make(map[string]int64),
		errorTypeCount:  make(map[string]int64),
	}
}

// RecordK8sQuery records metrics for K8s filtered queries
func (m *K8sMetrics) RecordK8sQuery(dataset string, duration time.Duration,
	filters []request.K8sFilter, resultCount int) {

	filterCount := len(filters)
	if filterCount == 0 {
		return // No K8s filters, skip K8s metrics
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update basic counters
	m.totalQueries++
	m.totalDuration += duration

	// Calculate complexity and record metrics
	complexity := m.calculateComplexityScore(filters)
	m.complexitySum += complexity

	// Estimate selectivity
	selectivity := m.estimateQuerySelectivity(filters, resultCount)
	m.selectivitySum += selectivity

	// Track filter type usage
	for _, filter := range filters {
		filterKey := fmt.Sprintf("%s_%s", filter.Field, string(filter.Type))
		m.filterTypeUsage[filterKey]++

		// Track specific filter patterns
		if filter.Field == "namespace" {
			m.namespaceFilterCount++
		} else if filter.Field == "pod" {
			m.podFilterCount++
		}
	}

	// Track slow queries
	if duration > 2*time.Second {
		m.slowQueries++
		klog.InfoS("Slow K8s query detected",
			"dataset", dataset,
			"duration_ms", duration.Milliseconds(),
			"filter_count", filterCount,
			"complexity", complexity)
	}

	// Log detailed metrics every 100 queries for monitoring
	if m.totalQueries%100 == 0 {
		m.logMetricsSummary(dataset)
	}
}

// RecordK8sError records K8s filtering errors
func (m *K8sMetrics) RecordK8sError(dataset, errorType, errorReason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalErrors++
	errorKey := fmt.Sprintf("%s_%s", errorType, errorReason)
	m.errorTypeCount[errorKey]++

	klog.InfoS("K8s filtering error recorded",
		"dataset", dataset,
		"error_type", errorType,
		"error_reason", errorReason,
		"total_errors", m.totalErrors)
}

// categorizeComplexity determines complexity level of K8s filters
func (m *K8sMetrics) categorizeComplexity(filters []request.K8sFilter) string {
	if len(filters) == 0 {
		return "none"
	}
	if len(filters) == 1 && filters[0].Type == request.K8sFilterExact {
		return "simple"
	}
	if len(filters) <= 3 {
		return "moderate"
	}
	if len(filters) <= 10 {
		return "complex"
	}
	return "very_complex"
}

// categorizeFilterTypes creates a summary of filter types used
func (m *K8sMetrics) categorizeFilterTypes(filters []request.K8sFilter) string {
	typeMap := make(map[request.K8sFilterType]bool)
	for _, filter := range filters {
		typeMap[filter.Type] = true
	}

	var types []string
	for filterType := range typeMap {
		types = append(types, string(filterType))
	}

	if len(types) == 1 {
		return types[0]
	}
	return "mixed"
}

// calculateComplexityScore provides numeric complexity scoring
func (m *K8sMetrics) calculateComplexityScore(filters []request.K8sFilter) float64 {
	score := 0.0
	for _, filter := range filters {
		switch filter.Type {
		case request.K8sFilterExact:
			score += 1.0
		case request.K8sFilterPrefix, request.K8sFilterSuffix:
			score += 2.0
		case request.K8sFilterContains:
			score += 3.0
		case request.K8sFilterWildcard:
			score += 4.0
		case request.K8sFilterRegex:
			score += 5.0
		default:
			score += 2.5 // Unknown filter type
		}

		// Case-insensitive adds complexity
		if filter.CaseInsensitive {
			score += 1.0
		}
	}
	return score
}

// estimateQuerySelectivity estimates how selective the query is
func (m *K8sMetrics) estimateQuerySelectivity(filters []request.K8sFilter, resultCount int) float64 {
	if resultCount == 0 {
		return 0.0
	}

	// Base selectivity estimation
	selectivity := 1.0
	for _, filter := range filters {
		switch filter.Type {
		case request.K8sFilterExact:
			selectivity *= 0.01 // Very selective
		case request.K8sFilterPrefix:
			selectivity *= 0.1 // Moderately selective
		case request.K8sFilterSuffix:
			selectivity *= 0.15 // Less selective than prefix
		case request.K8sFilterContains:
			selectivity *= 0.2 // Can be less selective
		case request.K8sFilterRegex:
			selectivity *= 0.3 // Highly variable
		case request.K8sFilterWildcard:
			selectivity *= 0.2 // Moderately selective
		}
	}

	// Adjust based on actual result count
	if resultCount < 10 {
		selectivity *= 0.1 // Very selective query
	} else if resultCount < 100 {
		selectivity *= 0.5 // Moderately selective
	}

	// Cap at 1.0
	if selectivity > 1.0 {
		selectivity = 1.0
	}

	return selectivity
}

// determineSlowestReason determines why a query was slow
func (m *K8sMetrics) determineSlowestReason(duration time.Duration, filterCount int,
	selectivity, complexity float64) string {

	if complexity > 50.0 {
		return "high_complexity"
	}
	if filterCount > 20 {
		return "too_many_filters"
	}
	if selectivity > 0.5 {
		return "low_selectivity"
	}
	if duration > 10*time.Second {
		return "extremely_slow"
	}
	return "general_slowness"
}

// logMetricsSummary logs a summary of current metrics
func (m *K8sMetrics) logMetricsSummary(dataset string) {
	avgDuration := float64(0)
	avgComplexity := float64(0)
	avgSelectivity := float64(0)

	if m.totalQueries > 0 {
		avgDuration = float64(m.totalDuration.Milliseconds()) / float64(m.totalQueries)
		avgComplexity = m.complexitySum / float64(m.totalQueries)
		avgSelectivity = m.selectivitySum / float64(m.totalQueries)
	}

	klog.InfoS("K8s filtering metrics summary",
		"dataset", dataset,
		"total_queries", m.totalQueries,
		"total_errors", m.totalErrors,
		"slow_queries", m.slowQueries,
		"avg_duration_ms", avgDuration,
		"avg_complexity", avgComplexity,
		"avg_selectivity", avgSelectivity,
		"namespace_filters", m.namespaceFilterCount,
		"pod_filters", m.podFilterCount,
		"top_filter_types", m.getTopFilterTypes(3))
}

// getTopFilterTypes returns the most used filter types
func (m *K8sMetrics) getTopFilterTypes(limit int) map[string]int64 {
	if len(m.filterTypeUsage) == 0 {
		return make(map[string]int64)
	}

	// For simplicity, return up to 'limit' filter types
	result := make(map[string]int64)
	count := 0
	for k, v := range m.filterTypeUsage {
		if count >= limit {
			break
		}
		result[k] = v
		count++
	}
	return result
}

// GetK8sMetricsSummary provides a summary of K8s filtering performance
func (m *K8sMetrics) GetK8sMetricsSummary(dataset string) map[string]interface{} {
	summary := make(map[string]interface{})
	summary["dataset"] = dataset
	summary["metrics_available"] = []string{
		"k8s_query_duration",
		"filter_complexity",
		"k8s_selectivity",
		"slow_k8s_queries",
		"k8s_filter_types",
		"k8s_errors",
		"k8s_filter_counts",
		"namespace_filters",
		"pod_filters",
	}
	summary["description"] = "K8s metadata filtering performance metrics"
	return summary
}