package clickhouse

import (
	"fmt"
	"strings"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// K8sFilterBuilder creates optimized ClickHouse conditions for K8s filters
type K8sFilterBuilder struct {
	filters []request.K8sFilter
}

// NewK8sFilterBuilder creates a new filter builder
func NewK8sFilterBuilder() *K8sFilterBuilder {
	return &K8sFilterBuilder{}
}

// SetFilters sets the K8s filters to process
func (b *K8sFilterBuilder) SetFilters(filters []request.K8sFilter) {
	b.filters = filters
}

// BuildK8sFilterConditions creates optimized ClickHouse conditions for K8s filters
func (b *K8sFilterBuilder) BuildK8sFilterConditions() ([]string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	if len(b.filters) == 0 {
		return conditions, args, nil
	}

	// Group filters by field and type for optimization
	namespaceFilters := make(map[request.K8sFilterType][]request.K8sFilter)
	podFilters := make(map[request.K8sFilterType][]request.K8sFilter)

	for _, filter := range b.filters {
		if filter.Field == "namespace" {
			namespaceFilters[filter.Type] = append(namespaceFilters[filter.Type], filter)
		} else if filter.Field == "pod" {
			podFilters[filter.Type] = append(podFilters[filter.Type], filter)
		}
	}

	// Build optimized namespace conditions
	if nsConditions, nsArgs, err := b.buildNamespaceConditions(namespaceFilters); err != nil {
		return nil, nil, err
	} else if len(nsConditions) > 0 {
		conditions = append(conditions, nsConditions...)
		args = append(args, nsArgs...)
	}

	// Build optimized pod conditions
	if podConditions, podArgs, err := b.buildPodConditions(podFilters); err != nil {
		return nil, nil, err
	} else if len(podConditions) > 0 {
		conditions = append(conditions, podConditions...)
		args = append(args, podArgs...)
	}

	return conditions, args, nil
}

// buildNamespaceConditions creates efficient ClickHouse conditions for namespace filtering
func (b *K8sFilterBuilder) buildNamespaceConditions(filters map[request.K8sFilterType][]request.K8sFilter) ([]string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	// Handle exact matches with IN clause for efficiency
	if exactFilters := filters[request.K8sFilterExact]; len(exactFilters) > 0 {
		var exactPatterns []string
		for _, filter := range exactFilters {
			exactPatterns = append(exactPatterns, filter.Pattern)
		}

		if len(exactPatterns) == 1 {
			conditions = append(conditions, "namespace_name = ?")
			args = append(args, exactPatterns[0])
		} else {
			placeholders := make([]string, len(exactPatterns))
			for i, pattern := range exactPatterns {
				placeholders[i] = "?"
				args = append(args, pattern)
			}
			conditions = append(conditions, fmt.Sprintf("namespace_name IN (%s)",
				strings.Join(placeholders, ",")))
		}
	}

	// Handle prefix matches
	if prefixFilters := filters[request.K8sFilterPrefix]; len(prefixFilters) > 0 {
		var prefixConditions []string
		for _, filter := range prefixFilters {
			prefixConditions = append(prefixConditions, "startsWith(namespace_name, ?)")
			args = append(args, filter.Pattern)
		}
		if len(prefixConditions) == 1 {
			conditions = append(conditions, prefixConditions[0])
		} else {
			conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(prefixConditions, " OR ")))
		}
	}

	// Handle regex matches
	if regexFilters := filters[request.K8sFilterRegex]; len(regexFilters) > 0 {
		var regexConditions []string
		for _, filter := range regexFilters {
			regexConditions = append(regexConditions, "match(namespace_name, ?)")
			args = append(args, filter.Pattern)
		}
		if len(regexConditions) == 1 {
			conditions = append(conditions, regexConditions[0])
		} else {
			conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(regexConditions, " OR ")))
		}
	}

	// Handle wildcard matches
	if wildcardFilters := filters[request.K8sFilterWildcard]; len(wildcardFilters) > 0 {
		var wildcardConditions []string
		for _, filter := range wildcardFilters {
			// Convert wildcard pattern to SQL LIKE pattern
			likePattern := strings.ReplaceAll(strings.ReplaceAll(filter.Pattern, "*", "%"), "?", "_")
			wildcardConditions = append(wildcardConditions, "namespace_name LIKE ?")
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
func (b *K8sFilterBuilder) buildPodConditions(filters map[request.K8sFilterType][]request.K8sFilter) ([]string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	// Handle exact matches
	if exactFilters := filters[request.K8sFilterExact]; len(exactFilters) > 0 {
		var exactPatterns []string
		for _, filter := range exactFilters {
			exactPatterns = append(exactPatterns, filter.Pattern)
		}

		if len(exactPatterns) == 1 {
			conditions = append(conditions, "pod_name = ?")
			args = append(args, exactPatterns[0])
		} else {
			placeholders := make([]string, len(exactPatterns))
			for i, pattern := range exactPatterns {
				placeholders[i] = "?"
				args = append(args, pattern)
			}
			conditions = append(conditions, fmt.Sprintf("pod_name IN (%s)",
				strings.Join(placeholders, ",")))
		}
	}

	// Handle prefix matches
	if prefixFilters := filters[request.K8sFilterPrefix]; len(prefixFilters) > 0 {
		var prefixConditions []string
		for _, filter := range prefixFilters {
			if filter.CaseInsensitive {
				prefixConditions = append(prefixConditions, "startsWithUTF8(lowerUTF8(pod_name), lowerUTF8(?))")
			} else {
				prefixConditions = append(prefixConditions, "startsWith(pod_name, ?)")
			}
			args = append(args, filter.Pattern)
		}
		conditions = append(conditions, b.combineConditions(prefixConditions))
	}

	// Handle suffix matches
	if suffixFilters := filters[request.K8sFilterSuffix]; len(suffixFilters) > 0 {
		var suffixConditions []string
		for _, filter := range suffixFilters {
			if filter.CaseInsensitive {
				suffixConditions = append(suffixConditions, "endsWithUTF8(lowerUTF8(pod_name), lowerUTF8(?))")
			} else {
				suffixConditions = append(suffixConditions, "endsWith(pod_name, ?)")
			}
			args = append(args, filter.Pattern)
		}
		conditions = append(conditions, b.combineConditions(suffixConditions))
	}

	// Handle contains matches
	if containsFilters := filters[request.K8sFilterContains]; len(containsFilters) > 0 {
		var containsConditions []string
		for _, filter := range containsFilters {
			if filter.CaseInsensitive {
				containsConditions = append(containsConditions, "positionCaseInsensitiveUTF8(pod_name, ?) > 0")
			} else {
				containsConditions = append(containsConditions, "position(pod_name, ?) > 0")
			}
			args = append(args, filter.Pattern)
		}
		conditions = append(conditions, b.combineConditions(containsConditions))
	}

	// Handle regex matches
	if regexFilters := filters[request.K8sFilterRegex]; len(regexFilters) > 0 {
		var regexConditions []string
		for _, filter := range regexFilters {
			if filter.CaseInsensitive {
				// For case-insensitive regex, add (?i) flag
				pattern := filter.Pattern
				if !strings.HasPrefix(pattern, "(?i)") {
					pattern = "(?i)" + pattern
				}
				regexConditions = append(regexConditions, "match(pod_name, ?)")
				args = append(args, pattern)
			} else {
				regexConditions = append(regexConditions, "match(pod_name, ?)")
				args = append(args, filter.Pattern)
			}
		}
		conditions = append(conditions, b.combineConditions(regexConditions))
	}

	// Handle wildcard matches
	if wildcardFilters := filters[request.K8sFilterWildcard]; len(wildcardFilters) > 0 {
		var wildcardConditions []string
		for _, filter := range wildcardFilters {
			likePattern := strings.ReplaceAll(strings.ReplaceAll(filter.Pattern, "*", "%"), "?", "_")
			if filter.CaseInsensitive {
				wildcardConditions = append(wildcardConditions, "lowerUTF8(pod_name) LIKE lowerUTF8(?)")
			} else {
				wildcardConditions = append(wildcardConditions, "pod_name LIKE ?")
			}
			args = append(args, likePattern)
		}
		conditions = append(conditions, b.combineConditions(wildcardConditions))
	}

	return conditions, args, nil
}

// combineConditions efficiently combines multiple conditions with OR logic
func (b *K8sFilterBuilder) combineConditions(conditions []string) string {
	if len(conditions) == 1 {
		return conditions[0]
	}
	return fmt.Sprintf("(%s)", strings.Join(conditions, " OR "))
}

// EstimateFilterComplexity calculates complexity score for performance optimization
func (b *K8sFilterBuilder) EstimateFilterComplexity() float64 {
	score := 0.0
	for _, filter := range b.filters {
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
		}

		// Case-insensitive adds complexity
		if filter.CaseInsensitive {
			score += 1.0
		}
	}
	return score
}

// EstimateFilterSelectivity provides query optimization hints
func (b *K8sFilterBuilder) EstimateFilterSelectivity() float64 {
	selectivity := 1.0

	for _, filter := range b.filters {
		switch filter.Type {
		case request.K8sFilterExact:
			selectivity *= 0.01 // Exact matches are highly selective
		case request.K8sFilterPrefix:
			selectivity *= 0.1 // Prefix matches are moderately selective
		case request.K8sFilterSuffix:
			selectivity *= 0.15 // Suffix matches are less selective than prefix
		case request.K8sFilterContains:
			selectivity *= 0.2 // Contains can be less selective
		case request.K8sFilterRegex:
			selectivity *= 0.3 // Regex can vary widely
		case request.K8sFilterWildcard:
			selectivity *= 0.2 // Wildcards are moderately selective
		default:
			selectivity *= 0.5 // Conservative estimate
		}
	}

	return selectivity
}