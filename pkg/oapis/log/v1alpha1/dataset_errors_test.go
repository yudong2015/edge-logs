package v1alpha1

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatasetNotFoundError(t *testing.T) {
	dataset := "test-dataset"
	err := NewDatasetNotFoundError(dataset)

	assert.Equal(t, dataset, err.Dataset)
	assert.Contains(t, err.Error(), dataset)
	assert.Contains(t, err.Error(), "not found or contains no data")
}

func TestDatasetUnauthorizedError(t *testing.T) {
	dataset := "restricted-dataset"
	err := NewDatasetUnauthorizedError(dataset)

	assert.Equal(t, dataset, err.Dataset)
	assert.Contains(t, err.Error(), dataset)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestDatasetValidationError(t *testing.T) {
	dataset := "invalid@dataset"
	details := "contains invalid characters"
	err := NewDatasetValidationError(dataset, details)

	assert.Equal(t, dataset, err.Dataset)
	assert.Equal(t, details, err.Details)
	assert.Contains(t, err.Error(), dataset)
	assert.Contains(t, err.Error(), details)
}

func TestDatasetSecurityError(t *testing.T) {
	dataset := "malicious'; DROP TABLE"
	details := "SQL injection detected"
	err := NewDatasetSecurityError(dataset, details)

	assert.Equal(t, dataset, err.Dataset)
	assert.Equal(t, details, err.Details)
	assert.Contains(t, err.Error(), dataset)
	assert.Contains(t, err.Error(), details)
}

func TestMapDatasetErrorToHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		error          error
		expectedStatus int
	}{
		{
			name:           "dataset not found",
			error:          NewDatasetNotFoundError("test"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "dataset unauthorized",
			error:          NewDatasetUnauthorizedError("test"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "dataset validation error",
			error:          NewDatasetValidationError("test", "invalid format"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "dataset security error",
			error:          NewDatasetSecurityError("test", "SQL injection"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "generic error",
			error:          assert.AnError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := MapDatasetErrorToHTTPStatus(tt.error)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestGetDatasetErrorMessage(t *testing.T) {
	dataset := "test-dataset"

	tests := []struct {
		name            string
		error           error
		expectedContains string
	}{
		{
			name:            "dataset not found",
			error:           NewDatasetNotFoundError(dataset),
			expectedContains: "not found",
		},
		{
			name:            "dataset unauthorized",
			error:           NewDatasetUnauthorizedError(dataset),
			expectedContains: "not authorized",
		},
		{
			name:            "dataset validation error",
			error:           NewDatasetValidationError(dataset, "invalid format"),
			expectedContains: "Invalid dataset name format",
		},
		{
			name:            "dataset security error",
			error:           NewDatasetSecurityError(dataset, "SQL injection"),
			expectedContains: "invalid or potentially harmful",
		},
		{
			name:            "generic error",
			error:           assert.AnError,
			expectedContains: "Dataset operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := GetDatasetErrorMessage(tt.error, dataset)
			assert.Contains(t, message, tt.expectedContains)
		})
	}
}

func TestDatasetError_Error(t *testing.T) {
	err := &DatasetError{
		Dataset:   "test-dataset",
		ErrorType: "validation",
		Message:   "invalid format",
		Code:      400,
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "test-dataset")
	assert.Contains(t, errorMsg, "invalid format")
}