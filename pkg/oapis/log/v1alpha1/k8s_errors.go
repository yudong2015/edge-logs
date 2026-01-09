package v1alpha1

import (
	"fmt"
	"net/http"
	"strings"
)

// K8sValidationError represents errors in K8s resource name validation
type K8sValidationError struct {
	Field       string
	Value       string
	Reason      string
	FilterType  string
	StatusCode  int
}

func (e *K8sValidationError) Error() string {
	return fmt.Sprintf("K8s %s filter validation failed for '%s' (%s): %s",
		e.Field, e.Value, e.FilterType, e.Reason)
}

// NewK8sValidationError creates a new K8s validation error
func NewK8sValidationError(field, value, filterType, reason string) *K8sValidationError {
	return &K8sValidationError{
		Field:      field,
		Value:      value,
		FilterType: filterType,
		Reason:     reason,
		StatusCode: http.StatusBadRequest,
	}
}

// K8sFilterComplexityError represents errors when K8s filters are too complex
type K8sFilterComplexityError struct {
	FilterCount int
	MaxAllowed  int
	Complexity  float64
	StatusCode  int
}

func (e *K8sFilterComplexityError) Error() string {
	return fmt.Sprintf("K8s filter complexity too high: %d filters (max: %d), complexity: %.1f",
		e.FilterCount, e.MaxAllowed, e.Complexity)
}

// NewK8sFilterComplexityError creates a new complexity error
func NewK8sFilterComplexityError(filterCount, maxAllowed int, complexity float64) *K8sFilterComplexityError {
	return &K8sFilterComplexityError{
		FilterCount: filterCount,
		MaxAllowed:  maxAllowed,
		Complexity:  complexity,
		StatusCode:  http.StatusBadRequest,
	}
}

// K8sPatternError represents errors in K8s pattern syntax
type K8sPatternError struct {
	Pattern    string
	PatternType string
	Reason     string
	Suggestions []string
	StatusCode int
}

func (e *K8sPatternError) Error() string {
	return fmt.Sprintf("K8s %s pattern '%s' is invalid: %s", e.PatternType, e.Pattern, e.Reason)
}

// NewK8sPatternError creates a new pattern error
func NewK8sPatternError(pattern, patternType, reason string, suggestions []string) *K8sPatternError {
	return &K8sPatternError{
		Pattern:     pattern,
		PatternType: patternType,
		Reason:      reason,
		Suggestions: suggestions,
		StatusCode:  http.StatusBadRequest,
	}
}

// K8sResourceFormatError represents errors in K8s resource name format
type K8sResourceFormatError struct {
	ResourceType string
	Value        string
	Violations   []string
	StatusCode   int
}

func (e *K8sResourceFormatError) Error() string {
	return fmt.Sprintf("K8s %s '%s' format is invalid: %s",
		e.ResourceType, e.Value, strings.Join(e.Violations, ", "))
}

// NewK8sResourceFormatError creates a new resource format error
func NewK8sResourceFormatError(resourceType, value string, violations []string) *K8sResourceFormatError {
	return &K8sResourceFormatError{
		ResourceType: resourceType,
		Value:        value,
		Violations:   violations,
		StatusCode:   http.StatusBadRequest,
	}
}

// MapK8sErrorToHTTPStatus maps K8s-specific errors to HTTP status codes
func MapK8sErrorToHTTPStatus(err error) int {
	switch e := err.(type) {
	case *K8sValidationError:
		return e.StatusCode
	case *K8sFilterComplexityError:
		return e.StatusCode
	case *K8sPatternError:
		return e.StatusCode
	case *K8sResourceFormatError:
		return e.StatusCode
	default:
		// Check for string patterns in error messages
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "K8s filter validation failed"),
			strings.Contains(errMsg, "invalid namespace filter"),
			strings.Contains(errMsg, "invalid pod filter"),
			strings.Contains(errMsg, "too many K8s filters"):
			return http.StatusBadRequest

		case strings.Contains(errMsg, "K8s filter complexity too high"),
			strings.Contains(errMsg, "regex pattern may be too expensive"):
			return http.StatusBadRequest

		case strings.Contains(errMsg, "DNS-1123 compliant"),
			strings.Contains(errMsg, "namespace format invalid"),
			strings.Contains(errMsg, "pod name format invalid"):
			return http.StatusBadRequest

		default:
			return http.StatusInternalServerError
		}
	}
}

// GetK8sErrorMessage provides user-friendly error messages for K8s errors
func GetK8sErrorMessage(err error, dataset string) string {
	switch e := err.(type) {
	case *K8sValidationError:
		return fmt.Sprintf("K8s %s filter '%s' is invalid: %s. Please check the format and try again.",
			e.Field, e.Value, e.Reason)

	case *K8sFilterComplexityError:
		return fmt.Sprintf("K8s query too complex with %d filters (max %d allowed). "+
			"Please simplify your K8s filters for dataset '%s'.",
			e.FilterCount, e.MaxAllowed, dataset)

	case *K8sPatternError:
		msg := fmt.Sprintf("K8s %s pattern '%s' is invalid: %s.",
			e.PatternType, e.Pattern, e.Reason)
		if len(e.Suggestions) > 0 {
			msg += fmt.Sprintf(" Suggestions: %s", strings.Join(e.Suggestions, ", "))
		}
		return msg

	case *K8sResourceFormatError:
		return fmt.Sprintf("K8s %s '%s' format is invalid: %s. "+
			"Please follow Kubernetes naming conventions.",
			e.ResourceType, e.Value, strings.Join(e.Violations, ", "))

	default:
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "K8s filter validation failed"):
			return fmt.Sprintf("K8s filter validation failed for dataset '%s': %v. "+
				"Please check your namespace and pod name patterns.", dataset, err)

		case strings.Contains(errMsg, "invalid namespace filter"):
			return fmt.Sprintf("Invalid namespace filter for dataset '%s': %v. "+
				"Namespaces must follow DNS-1123 format (lowercase, alphanumeric, hyphens).", dataset, err)

		case strings.Contains(errMsg, "invalid pod filter"):
			return fmt.Sprintf("Invalid pod name filter for dataset '%s': %v. "+
				"Pod names support patterns: exact, prefix*, *suffix, *contains*, regex:pattern, icase:pattern.", dataset, err)

		case strings.Contains(errMsg, "too many K8s filters"):
			return fmt.Sprintf("Too many K8s filters for dataset '%s': %v. "+
				"Please reduce the number of namespace and pod filters.", dataset, err)

		case strings.Contains(errMsg, "regex pattern may be too expensive"):
			return fmt.Sprintf("K8s regex pattern too complex for dataset '%s': %v. "+
				"Please use simpler regex patterns to avoid performance issues.", dataset, err)

		default:
			return fmt.Sprintf("K8s filtering error for dataset '%s': %v", dataset, err)
		}
	}
}

// HandleK8sError handles K8s-specific errors with appropriate HTTP responses
func (h *LogHandler) HandleK8sError(err error, dataset string) (int, map[string]interface{}) {
	statusCode := MapK8sErrorToHTTPStatus(err)
	message := GetK8sErrorMessage(err, dataset)

	errorResponse := map[string]interface{}{
		"error":   message,
		"dataset": dataset,
		"type":    "k8s_filtering_error",
	}

	// Add specific error details based on error type
	switch e := err.(type) {
	case *K8sValidationError:
		errorResponse["field"] = e.Field
		errorResponse["value"] = e.Value
		errorResponse["filter_type"] = e.FilterType
		errorResponse["reason"] = e.Reason
		errorResponse["supported_patterns"] = map[string][]string{
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
		}

	case *K8sFilterComplexityError:
		errorResponse["filter_count"] = e.FilterCount
		errorResponse["max_allowed"] = e.MaxAllowed
		errorResponse["complexity_score"] = e.Complexity
		errorResponse["optimization_tips"] = []string{
			"Use exact matches when possible",
			"Combine multiple exact matches with comma separation",
			"Avoid complex regex patterns",
			"Use prefix/suffix patterns instead of contains",
			"Reduce the number of wildcard patterns",
		}

	case *K8sPatternError:
		errorResponse["pattern"] = e.Pattern
		errorResponse["pattern_type"] = e.PatternType
		errorResponse["suggestions"] = e.Suggestions

	case *K8sResourceFormatError:
		errorResponse["resource_type"] = e.ResourceType
		errorResponse["value"] = e.Value
		errorResponse["violations"] = e.Violations
		errorResponse["k8s_naming_rules"] = map[string]string{
			"namespace": "DNS-1123 compliant: lowercase alphanumeric and hyphens, max 63 chars",
			"pod":       "Kubernetes naming: alphanumeric, hyphens, dots, underscores, max 253 chars",
		}
	}

	// Add usage examples
	errorResponse["examples"] = []string{
		"namespace=production",
		"namespaces=kube-system,default",
		"pod_names=web-*,api-service-*",
		"pods=regex:^app-.*,*worker*",
		"pods=icase:WEB-*,*database*",
	}

	return statusCode, errorResponse
}