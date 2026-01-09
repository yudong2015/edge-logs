package v1alpha1

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"

	responseWrapper "github.com/outpostos/edge-logs/pkg/response"
)

// ContentSearchValidationError represents content search validation errors
type ContentSearchValidationError struct {
	Pattern     string
	SearchType  string
	Reason      string
	Suggestion  string
}

func (e *ContentSearchValidationError) Error() string {
	return fmt.Sprintf("content search validation failed for pattern '%s' (%s): %s",
		e.Pattern, e.SearchType, e.Reason)
}

// ContentSearchComplexityError represents content search complexity errors
type ContentSearchComplexityError struct {
	Complexity float64
	MaxAllowed float64
	Patterns   []string
}

func (e *ContentSearchComplexityError) Error() string {
	return fmt.Sprintf("content search complexity (%.1f) exceeds maximum (%.1f) for patterns: %v",
		e.Complexity, e.MaxAllowed, e.Patterns)
}

// handleContentSearchError handles content search specific errors
func (h *LogHandler) handleContentSearchError(resp *restful.Response, err error, dataset string) {
	klog.ErrorS(err, "Content search error", "dataset", dataset)

	errMsg := err.Error()

	switch {
	case contains(errMsg, "content search validation failed"):
		h.handleContentSearchValidationError(resp, err, dataset)
	case contains(errMsg, "search complexity"):
		h.handleContentSearchComplexityError(resp, err, dataset)
	case contains(errMsg, "unsafe regex pattern"):
		h.handleUnsafeRegexError(resp, err, dataset)
	case contains(errMsg, "search query too long"):
		h.handleSearchQueryTooLongError(resp, err, dataset)
	case contains(errMsg, "too many search terms"):
		h.handleTooManyTermsError(resp, err, dataset)
	default:
		h.handleGenericContentSearchError(resp, err, dataset)
	}
}

// handleContentSearchValidationError handles validation errors
func (h *LogHandler) handleContentSearchValidationError(resp *restful.Response, err error, dataset string) {
	errorResp := map[string]interface{}{
		"error":   "Invalid content search pattern",
		"message": err.Error(),
		"dataset": dataset,
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

	if h.metrics != nil {
		h.metrics.RecordSearchError(dataset, "validation_error")
	}

	resp.WriteHeaderAndEntity(http.StatusBadRequest, errorResp)
}

// handleContentSearchComplexityError handles complexity errors
func (h *LogHandler) handleContentSearchComplexityError(resp *restful.Response, err error, dataset string) {
	errorResp := map[string]interface{}{
		"error":   "Content search complexity too high",
		"message": err.Error(),
		"dataset": dataset,
		"optimization_tips": []string{
			"Use exact matches instead of wildcards when possible",
			"Avoid complex regex patterns with multiple quantifiers",
			"Limit boolean expressions to essential terms",
			"Consider breaking complex searches into multiple queries",
			"Use phrase search for exact multi-word matches",
		},
		"examples": []string{
			"Simple: error",
			"Better: \"connection failed\"",
			"Avoid: regex:.*error.*timeout.*failed.*",
			"Better: boolean:error AND (timeout OR failed)",
		},
	}

	if h.metrics != nil {
		h.metrics.RecordSearchError(dataset, "complexity_error")
	}

	resp.WriteHeaderAndEntity(http.StatusBadRequest, errorResp)
}

// handleUnsafeRegexError handles unsafe regex pattern errors
func (h *LogHandler) handleUnsafeRegexError(resp *restful.Response, err error, dataset string) {
	errorResp := map[string]interface{}{
		"error":   "Unsafe regex pattern detected",
		"message": "The regex pattern contains potentially expensive constructs that could impact performance",
		"dataset": dataset,
		"unsafe_patterns": []string{
			".*.*  (multiple greedy quantifiers)",
			".+.+  (multiple possessive quantifiers)",
			"(?<!.*  (negative lookbehind with .*)",
			"(?!.*)  (negative lookahead with .*)",
		},
		"safe_alternatives": []string{
			"Use exact matches: error",
			"Use wildcard: error*",
			"Use simple regex: ^error",
			"Use phrase search: \"connection failed\"",
		},
	}

	if h.metrics != nil {
		h.metrics.RecordSearchError(dataset, "unsafe_regex")
	}

	resp.WriteHeaderAndEntity(http.StatusBadRequest, errorResp)
}

// handleSearchQueryTooLongError handles query length errors
func (h *LogHandler) handleSearchQueryTooLongError(resp *restful.Response, err error, dataset string) {
	errorResp := map[string]interface{}{
		"error":   "Search query too long",
		"message": err.Error(),
		"dataset": dataset,
		"suggestions": []string{
			"Shorten your search query to less than 500 characters",
			"Use multiple simpler queries instead of one complex query",
			"Focus on the most important search terms",
		},
	}

	if h.metrics != nil {
		h.metrics.RecordSearchError(dataset, "query_too_long")
	}

	resp.WriteHeaderAndEntity(http.StatusBadRequest, errorResp)
}

// handleTooManyTermsError handles too many search terms errors
func (h *LogHandler) handleTooManyTermsError(resp *restful.Response, err error, dataset string) {
	errorResp := map[string]interface{}{
		"error":   "Too many search terms",
		"message": err.Error(),
		"dataset": dataset,
		"suggestions": []string{
			"Limit your search to the most important terms (max 20)",
			"Use phrase search for multi-word terms: \"connection failed\"",
			"Use boolean operators to combine related terms: error AND timeout",
		},
	}

	if h.metrics != nil {
		h.metrics.RecordSearchError(dataset, "too_many_terms")
	}

	resp.WriteHeaderAndEntity(http.StatusBadRequest, errorResp)
}

// handleGenericContentSearchError handles general content search errors
func (h *LogHandler) handleGenericContentSearchError(resp *restful.Response, err error, dataset string) {
	errorResp := &responseWrapper.ErrorResponse{
		Code:    http.StatusBadRequest,
		Message: "Content search error: " + err.Error(),
	}

	if h.metrics != nil {
		h.metrics.RecordSearchError(dataset, "generic_error")
	}

	resp.WriteHeaderAndEntity(http.StatusBadRequest, errorResp)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		containsSubstring(s, substr)))
}

// containsSubstring performs substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}