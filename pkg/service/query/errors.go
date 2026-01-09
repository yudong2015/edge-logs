package query

import (
	"fmt"
)

// ServiceError represents service-level errors with context
type ServiceError struct {
	Op      string                 // Operation name
	Message string                 // Error message
	Err     error                  // Original error
	Context map[string]interface{} // Additional context
}

// Error implements the error interface
func (se *ServiceError) Error() string {
	if se.Err != nil {
		return fmt.Sprintf("service operation '%s' failed: %s: %v", se.Op, se.Message, se.Err)
	}
	return fmt.Sprintf("service operation '%s' failed: %s", se.Op, se.Message)
}

// Unwrap returns the underlying error
func (se *ServiceError) Unwrap() error {
	return se.Err
}

// NewValidationError creates a validation error
func NewValidationError(op, message string) *ServiceError {
	return &ServiceError{
		Op:      op,
		Message: message,
		Context: map[string]interface{}{
			"error_type": "validation_error",
			"retryable":  false,
		},
	}
}

// NewBusinessLogicError creates a business logic error
func NewBusinessLogicError(op, message string) *ServiceError {
	return &ServiceError{
		Op:      op,
		Message: message,
		Context: map[string]interface{}{
			"error_type": "business_logic_error",
			"retryable":  false,
		},
	}
}

// NewRepositoryError wraps repository errors for service layer
func NewRepositoryError(op string, err error) *ServiceError {
	return &ServiceError{
		Op:      op,
		Message: "repository operation failed",
		Err:     err,
		Context: map[string]interface{}{
			"error_type": "repository_error",
			"retryable":  isRetryableRepositoryError(err),
		},
	}
}

// NewTransformationError creates a data transformation error
func NewTransformationError(op, message string) *ServiceError {
	return &ServiceError{
		Op:      op,
		Message: message,
		Context: map[string]interface{}{
			"error_type": "transformation_error",
			"retryable":  false,
		},
	}
}

// isRetryableRepositoryError determines if a repository error is retryable
func isRetryableRepositoryError(err error) bool {
	// Check if it's a repository error with retryable context
	if repoErr, ok := err.(interface{ Unwrap() error }); ok {
		unwrapped := repoErr.Unwrap()
		// Check for common retryable patterns
		errMsg := unwrapped.Error()
		return containsRetryableErrorPattern(errMsg)
	}
	return false
}

// containsRetryableErrorPattern checks for retryable error patterns
func containsRetryableErrorPattern(errMsg string) bool {
	retryablePatterns := []string{
		"timeout", "connection", "network",
		"temporary", "transient",
	}

	for _, pattern := range retryablePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

// contains checks if string contains substring (case sensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
