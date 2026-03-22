package clickhouse

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

// RepositoryError represents repository-specific errors with context
type RepositoryError struct {
	Op      string                 // Operation (QueryLogs, InsertLog)
	Table   string                 // Table name
	Dataset string                 // Dataset context
	Err     error                  // Original error
	Context map[string]interface{} // Additional context
}

// Error implements the error interface
func (re *RepositoryError) Error() string {
	if re.Table != "" {
		return fmt.Sprintf("repository operation '%s' on table '%s' failed: %v", re.Op, re.Table, re.Err)
	}
	return fmt.Sprintf("repository operation '%s' failed: %v", re.Op, re.Err)
}

// Unwrap returns the underlying error
func (re *RepositoryError) Unwrap() error {
	return re.Err
}

// ErrorType represents the category of error
type ErrorType string

const (
	// Connection related errors
	ErrorTypeConnection     ErrorType = "connection_error"
	ErrorTypeTimeout        ErrorType = "timeout_error"
	ErrorTypeAuthentication ErrorType = "auth_error"

	// Query related errors
	ErrorTypeQuery      ErrorType = "query_error"
	ErrorTypeSyntax     ErrorType = "syntax_error"
	ErrorTypeValidation ErrorType = "validation_error"

	// Data related errors
	ErrorTypeDataFormat ErrorType = "data_format_error"
	ErrorTypeConstraint ErrorType = "constraint_error"
	ErrorTypeNotFound   ErrorType = "not_found_error"

	// System related errors
	ErrorTypeSystem   ErrorType = "system_error"
	ErrorTypeResource ErrorType = "resource_error"
	ErrorTypeUnknown  ErrorType = "unknown_error"
)

// MapClickHouseError maps ClickHouse errors to repository errors with proper categorization
func MapClickHouseError(err error, op string) *RepositoryError {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())
	repoErr := &RepositoryError{
		Op:      op,
		Table:   "logs", // Default table (unified logs table)
		Err:     err,
		Context: make(map[string]interface{}),
	}

	// Categorize errors by examining error messages
	switch {
	// Connection errors
	case strings.Contains(errMsg, "connection refused"):
		repoErr.Context["type"] = ErrorTypeConnection
		repoErr.Context["retryable"] = true
		repoErr.Context["description"] = "ClickHouse server is not accessible"

	case strings.Contains(errMsg, "connection reset"):
		repoErr.Context["type"] = ErrorTypeConnection
		repoErr.Context["retryable"] = true
		repoErr.Context["description"] = "Connection was reset by ClickHouse server"

	case strings.Contains(errMsg, "no such host"):
		repoErr.Context["type"] = ErrorTypeConnection
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "ClickHouse host cannot be resolved"

	// Timeout errors
	case strings.Contains(errMsg, "timeout"):
		repoErr.Context["type"] = ErrorTypeTimeout
		repoErr.Context["retryable"] = true
		repoErr.Context["description"] = "Operation timed out"

	case strings.Contains(errMsg, "deadline exceeded"):
		repoErr.Context["type"] = ErrorTypeTimeout
		repoErr.Context["retryable"] = true
		repoErr.Context["description"] = "Context deadline exceeded"

	// Authentication errors
	case strings.Contains(errMsg, "authentication failed"):
		repoErr.Context["type"] = ErrorTypeAuthentication
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "ClickHouse authentication failed"

	case strings.Contains(errMsg, "access denied"):
		repoErr.Context["type"] = ErrorTypeAuthentication
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "Access denied to ClickHouse resource"

	// Query syntax errors
	case strings.Contains(errMsg, "syntax error"):
		repoErr.Context["type"] = ErrorTypeSyntax
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "SQL syntax error"

	case strings.Contains(errMsg, "unknown identifier"):
		repoErr.Context["type"] = ErrorTypeSyntax
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "Unknown column or identifier"

	case strings.Contains(errMsg, "no such table"):
		repoErr.Context["type"] = ErrorTypeValidation
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "Table does not exist"

	// Data format errors
	case strings.Contains(errMsg, "type mismatch"):
		repoErr.Context["type"] = ErrorTypeDataFormat
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "Data type mismatch"

	case strings.Contains(errMsg, "cannot parse"):
		repoErr.Context["type"] = ErrorTypeDataFormat
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "Cannot parse data format"

	// Resource errors
	case strings.Contains(errMsg, "memory limit"):
		repoErr.Context["type"] = ErrorTypeResource
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "Memory limit exceeded"

	case strings.Contains(errMsg, "disk space"):
		repoErr.Context["type"] = ErrorTypeResource
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "Insufficient disk space"

	case strings.Contains(errMsg, "too many connections"):
		repoErr.Context["type"] = ErrorTypeResource
		repoErr.Context["retryable"] = true
		repoErr.Context["description"] = "Too many connections to ClickHouse"

	// System errors
	case strings.Contains(errMsg, "server error"):
		repoErr.Context["type"] = ErrorTypeSystem
		repoErr.Context["retryable"] = true
		repoErr.Context["description"] = "ClickHouse server error"

	default:
		repoErr.Context["type"] = ErrorTypeUnknown
		repoErr.Context["retryable"] = false
		repoErr.Context["description"] = "Unknown error"
	}

	// Log the error with structured information
	klog.ErrorS(err, "ClickHouse 操作错误",
		"operation", op,
		"error_type", repoErr.Context["type"],
		"retryable", repoErr.Context["retryable"],
		"description", repoErr.Context["description"])

	return repoErr
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if repoErr, ok := err.(*RepositoryError); ok {
		if retryable, exists := repoErr.Context["retryable"]; exists {
			return retryable.(bool)
		}
	}
	return false
}

// GetErrorType extracts the error type from a repository error
func GetErrorType(err error) ErrorType {
	if repoErr, ok := err.(*RepositoryError); ok {
		if errorType, exists := repoErr.Context["type"]; exists {
			return errorType.(ErrorType)
		}
	}
	return ErrorTypeUnknown
}

// NewValidationError creates a validation error
func NewValidationError(op, message string) *RepositoryError {
	return &RepositoryError{
		Op:    op,
		Table: "logs",
		Err:   fmt.Errorf("validation failed: %s", message),
		Context: map[string]interface{}{
			"type":        ErrorTypeValidation,
			"retryable":   false,
			"description": message,
		},
	}
}

// NewQueryError creates a query execution error
func NewQueryError(op, query string, err error) *RepositoryError {
	return &RepositoryError{
		Op:    op,
		Table: "logs",
		Err:   fmt.Errorf("query execution failed: %w", err),
		Context: map[string]interface{}{
			"type":        ErrorTypeQuery,
			"retryable":   IsRetryableError(err),
			"description": "Query execution failed",
			"query":       query,
		},
	}
}

// NewDataFormatError creates a data format error
func NewDataFormatError(op, field string, err error) *RepositoryError {
	return &RepositoryError{
		Op:    op,
		Table: "logs",
		Err:   fmt.Errorf("data format error in field '%s': %w", field, err),
		Context: map[string]interface{}{
			"type":        ErrorTypeDataFormat,
			"retryable":   false,
			"description": fmt.Sprintf("Invalid data format in field: %s", field),
			"field":       field,
		},
	}
}
