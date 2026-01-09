package v1alpha1

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
)

// TimeParameterError represents API-level time parameter errors
type TimeParameterError struct {
	Parameter string
	Value     string
	Issue     string
	Code      int
}

func (e *TimeParameterError) Error() string {
	return fmt.Sprintf("time parameter error: %s='%s' - %s", e.Parameter, e.Value, e.Issue)
}

// TimeFormatError represents time format validation errors at API level
type TimeFormatError struct {
	Parameter       string
	Value           string
	SupportedFormats []string
	Example         string
}

func (e *TimeFormatError) Error() string {
	return fmt.Sprintf("invalid time format for %s: '%s'", e.Parameter, e.Value)
}

// TimeRangeAPIError represents time range errors at API level
type TimeRangeAPIError struct {
	StartTime *time.Time
	EndTime   *time.Time
	Issue     string
	Suggestion string
}

func (e *TimeRangeAPIError) Error() string {
	return fmt.Sprintf("time range error: %s", e.Issue)
}

// HandleTimeError handles time-specific errors with comprehensive API responses
func (h *LogHandler) HandleTimeError(resp *restful.Response, err error, dataset string) {
	switch e := err.(type) {
	case *TimeParameterError:
		h.writeTimeParameterErrorResponse(resp, e, dataset)
	case *TimeFormatError:
		h.writeTimeFormatErrorResponse(resp, e, dataset)
	case *TimeRangeAPIError:
		h.writeTimeRangeErrorResponse(resp, e, dataset)
	default:
		// Handle generic time errors from service layer
		h.handleGenericTimeError(resp, err, dataset)
	}
}

// writeTimeParameterErrorResponse writes detailed time parameter error responses
func (h *LogHandler) writeTimeParameterErrorResponse(resp *restful.Response, err *TimeParameterError, dataset string) {
	errorResp := map[string]interface{}{
		"error":     "Time parameter validation failed",
		"parameter": err.Parameter,
		"value":     err.Value,
		"issue":     err.Issue,
		"dataset":   dataset,
		"guidelines": map[string]interface{}{
			"supported_formats": []string{
				"RFC3339: 2006-01-02T15:04:05Z",
				"With milliseconds: 2006-01-02T15:04:05.123Z",
				"With microseconds: 2006-01-02T15:04:05.123456Z",
				"With nanoseconds: 2006-01-02T15:04:05.123456789Z",
				"With timezone: 2006-01-02T15:04:05.123-07:00",
			},
			"examples": map[string]string{
				"basic":       "start_time=2024-01-01T10:30:45Z",
				"millisec":    "start_time=2024-01-01T10:30:45.123Z",
				"microsec":    "start_time=2024-01-01T10:30:45.123456Z",
				"nanosec":     "start_time=2024-01-01T10:30:45.123456789Z",
				"timezone":    "start_time=2024-01-01T10:30:45.123-07:00",
			},
			"precision": "Supports up to nanosecond precision",
			"timezone": "All times normalized to UTC",
		},
	}

	h.writeErrorResponseWithDetails(resp, err.Code, errorResp)
	h.timeMetrics.RecordTimeError(dataset, "parameter_validation_failed", err.Parameter)
}

// writeTimeFormatErrorResponse writes detailed time format error responses
func (h *LogHandler) writeTimeFormatErrorResponse(resp *restful.Response, err *TimeFormatError, dataset string) {
	errorResp := map[string]interface{}{
		"error":     "Invalid time format",
		"parameter": err.Parameter,
		"value":     err.Value,
		"dataset":   dataset,
		"supported_formats": err.SupportedFormats,
		"example":   err.Example,
		"help": map[string]interface{}{
			"iso8601_compliance": "Time format must be ISO 8601 compliant",
			"precision_support":  "Millisecond, microsecond, and nanosecond precision supported",
			"timezone_handling":  "Timezone information preserved and normalized to UTC",
			"validation_rules": []string{
				"Date must be in YYYY-MM-DD format",
				"Time must be in HH:MM:SS format",
				"Optional fractional seconds: .SSS, .SSSSSS, or .SSSSSSSSS",
				"Timezone: Z (UTC) or ±HH:MM offset",
			},
		},
	}

	h.writeErrorResponseWithDetails(resp, http.StatusBadRequest, errorResp)
	h.timeMetrics.RecordTimeError(dataset, "format_validation_failed", err.Parameter)
}

// writeTimeRangeErrorResponse writes detailed time range error responses
func (h *LogHandler) writeTimeRangeErrorResponse(resp *restful.Response, err *TimeRangeAPIError, dataset string) {
	errorResp := map[string]interface{}{
		"error":      "Time range validation failed",
		"issue":      err.Issue,
		"start_time": formatOptionalTimeForAPI(err.StartTime),
		"end_time":   formatOptionalTimeForAPI(err.EndTime),
		"dataset":    dataset,
		"suggestion": err.Suggestion,
		"constraints": map[string]interface{}{
			"max_time_span":    "24 hours",
			"boundary_type":    "inclusive (start <= timestamp <= end)",
			"timezone_policy":  "all times converted to UTC",
			"future_times":     "not allowed",
			"precision_support": "up to nanosecond precision",
		},
		"examples": map[string]string{
			"hour_range":     "start_time=2024-01-01T10:00:00Z&end_time=2024-01-01T11:00:00Z",
			"minute_range":   "start_time=2024-01-01T10:30:00Z&end_time=2024-01-01T10:31:00Z",
			"second_range":   "start_time=2024-01-01T10:30:45Z&end_time=2024-01-01T10:30:46Z",
			"millisec_range": "start_time=2024-01-01T10:30:45.123Z&end_time=2024-01-01T10:30:45.456Z",
			"microsec_range": "start_time=2024-01-01T10:30:45.123456Z&end_time=2024-01-01T10:30:45.123789Z",
		},
	}

	h.writeErrorResponseWithDetails(resp, http.StatusBadRequest, errorResp)
	h.timeMetrics.RecordTimeError(dataset, "range_validation_failed", "time_range")
}

// handleGenericTimeError handles service layer time errors
func (h *LogHandler) handleGenericTimeError(resp *restful.Response, err error, dataset string) {
	errMsg := err.Error()
	var statusCode int
	var errorType string

	// Map service layer errors to appropriate HTTP status codes
	switch {
	case strings.Contains(errMsg, "time validation failed"),
		strings.Contains(errMsg, "time format"),
		strings.Contains(errMsg, "time range error"):
		statusCode = http.StatusBadRequest
		errorType = "validation_failed"

	case strings.Contains(errMsg, "time range too large"),
		strings.Contains(errMsg, "maximum allowed span"):
		statusCode = http.StatusBadRequest
		errorType = "range_too_large"

	case strings.Contains(errMsg, "future"),
		strings.Contains(errMsg, "cannot be in the future"):
		statusCode = http.StatusBadRequest
		errorType = "future_time_not_allowed"

	default:
		statusCode = http.StatusBadRequest
		errorType = "unknown_time_error"
	}

	errorResp := map[string]interface{}{
		"error":   "Time processing failed",
		"message": errMsg,
		"dataset": dataset,
		"type":    errorType,
		"help": map[string]string{
			"format_guide": "Use ISO 8601 format (RFC3339) with optional millisecond precision",
			"range_limit":  "Maximum time range: 24 hours",
			"timezone":     "All times converted to UTC",
			"precision":    "Supports up to nanosecond precision",
		},
	}

	h.writeErrorResponseWithDetails(resp, statusCode, errorResp)
	h.timeMetrics.RecordTimeError(dataset, errorType, "time_generic")
}

// NewTimeParameterError creates a new time parameter error
func NewTimeParameterError(parameter, value, issue string, code int) *TimeParameterError {
	return &TimeParameterError{
		Parameter: parameter,
		Value:     value,
		Issue:     issue,
		Code:      code,
	}
}

// NewTimeFormatError creates a new time format error
func NewTimeFormatError(parameter, value string, supportedFormats []string, example string) *TimeFormatError {
	return &TimeFormatError{
		Parameter:        parameter,
		Value:            value,
		SupportedFormats: supportedFormats,
		Example:          example,
	}
}

// NewTimeRangeAPIError creates a new time range API error
func NewTimeRangeAPIError(startTime, endTime *time.Time, issue, suggestion string) *TimeRangeAPIError {
	return &TimeRangeAPIError{
		StartTime:  startTime,
		EndTime:    endTime,
		Issue:      issue,
		Suggestion: suggestion,
	}
}

// Helper functions for API error handling

// formatOptionalTimeForAPI formats time pointers for API responses
func formatOptionalTimeForAPI(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

// writeErrorResponseWithDetails writes enhanced error responses
func (h *LogHandler) writeErrorResponseWithDetails(resp *restful.Response, statusCode int, details interface{}) {
	resp.WriteHeaderAndEntity(statusCode, details)
}

// GetTimeErrorHelp returns comprehensive time parameter help
func GetTimeErrorHelp() map[string]interface{} {
	return map[string]interface{}{
		"supported_formats": []string{
			"RFC3339: 2006-01-02T15:04:05Z",
			"With milliseconds: 2006-01-02T15:04:05.123Z",
			"With microseconds: 2006-01-02T15:04:05.123456Z",
			"With nanoseconds: 2006-01-02T15:04:05.123456789Z",
			"With timezone: 2006-01-02T15:04:05.123-07:00",
		},
		"examples": map[string]string{
			"basic_query":      "/logs?start_time=2024-01-01T10:00:00Z&end_time=2024-01-01T11:00:00Z",
			"millisec_query":   "/logs?start_time=2024-01-01T10:30:45.123Z&end_time=2024-01-01T10:30:45.456Z",
			"microsec_query":   "/logs?start_time=2024-01-01T10:30:45.123456Z&end_time=2024-01-01T10:30:45.123789Z",
			"nanosec_query":    "/logs?start_time=2024-01-01T10:30:45.123456789Z&end_time=2024-01-01T10:30:45.123456999Z",
		},
		"constraints": map[string]string{
			"max_span":     "24 hours",
			"boundaries":   "inclusive (start <= timestamp <= end)",
			"timezone":     "normalized to UTC",
			"future_times": "not allowed",
			"precision":    "up to nanosecond",
		},
	}
}