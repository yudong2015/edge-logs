package v1alpha1

import (
	"fmt"
	"net/http"
)

// DatasetError represents dataset-specific errors
type DatasetError struct {
	Dataset   string
	ErrorType string
	Message   string
	Code      int
}

func (e *DatasetError) Error() string {
	return fmt.Sprintf("dataset '%s' error: %s", e.Dataset, e.Message)
}

// DatasetNotFoundError indicates dataset does not exist or contains no data
type DatasetNotFoundError struct {
	Dataset string
}

func (e *DatasetNotFoundError) Error() string {
	return fmt.Sprintf("dataset '%s' not found or contains no data", e.Dataset)
}

func NewDatasetNotFoundError(dataset string) *DatasetNotFoundError {
	return &DatasetNotFoundError{Dataset: dataset}
}

// DatasetUnauthorizedError indicates access to dataset is not authorized
type DatasetUnauthorizedError struct {
	Dataset string
}

func (e *DatasetUnauthorizedError) Error() string {
	return fmt.Sprintf("access to dataset '%s' is not authorized", e.Dataset)
}

func NewDatasetUnauthorizedError(dataset string) *DatasetUnauthorizedError {
	return &DatasetUnauthorizedError{Dataset: dataset}
}

// DatasetValidationError indicates dataset name format is invalid
type DatasetValidationError struct {
	Dataset string
	Details string
}

func (e *DatasetValidationError) Error() string {
	return fmt.Sprintf("dataset '%s' validation failed: %s", e.Dataset, e.Details)
}

func NewDatasetValidationError(dataset, details string) *DatasetValidationError {
	return &DatasetValidationError{Dataset: dataset, Details: details}
}

// DatasetSecurityError indicates potential security violation
type DatasetSecurityError struct {
	Dataset string
	Details string
}

func (e *DatasetSecurityError) Error() string {
	return fmt.Sprintf("dataset '%s' security violation: %s", e.Dataset, e.Details)
}

func NewDatasetSecurityError(dataset, details string) *DatasetSecurityError {
	return &DatasetSecurityError{Dataset: dataset, Details: details}
}

// MapDatasetErrorToHTTPStatus maps dataset errors to appropriate HTTP status codes
func MapDatasetErrorToHTTPStatus(err error) int {
	switch err.(type) {
	case *DatasetNotFoundError:
		return http.StatusNotFound
	case *DatasetUnauthorizedError:
		return http.StatusForbidden
	case *DatasetValidationError:
		return http.StatusBadRequest
	case *DatasetSecurityError:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// GetDatasetErrorMessage returns user-friendly error message
func GetDatasetErrorMessage(err error, dataset string) string {
	switch e := err.(type) {
	case *DatasetNotFoundError:
		return fmt.Sprintf("Dataset '%s' not found. Available datasets can be listed via the datasets endpoint.", dataset)
	case *DatasetUnauthorizedError:
		return "Access to the requested dataset is not authorized"
	case *DatasetValidationError:
		return fmt.Sprintf("Invalid dataset name format: %s", e.Details)
	case *DatasetSecurityError:
		return "Dataset name contains invalid or potentially harmful content"
	default:
		return fmt.Sprintf("Dataset operation failed: %v", err)
	}
}