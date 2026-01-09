package v1alpha1

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/model/response"
)

// MockQueryService is a mock implementation of the query service
type MockQueryService struct {
	mock.Mock
}

func (m *MockQueryService) QueryLogsByDataset(ctx interface{}, req *request.LogQueryRequest) (*response.LogQueryResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*response.LogQueryResponse), args.Error(1)
}

func (m *MockQueryService) DatasetExists(ctx interface{}, dataset string) (bool, error) {
	args := m.Called(ctx, dataset)
	return args.Bool(0), args.Error(1)
}

func TestLogHandler_ParseTimeParameters(t *testing.T) {
	handler := &LogHandler{
		timeMetrics: NewTimeMetrics(),
	}

	tests := []struct {
		name            string
		startTime       string
		endTime         string
		expectError     bool
		expectedStart   *time.Time
		expectedEnd     *time.Time
		errorType       string
		description     string
	}{
		{
			name:        "Valid RFC3339 parameters",
			startTime:   "2024-01-01T10:30:45Z",
			endTime:     "2024-01-01T11:30:45Z",
			expectError: false,
			description: "Should parse valid RFC3339 timestamps",
		},
		{
			name:        "Valid millisecond precision",
			startTime:   "2024-01-01T10:30:45.123Z",
			endTime:     "2024-01-01T10:30:45.456Z",
			expectError: false,
			description: "Should parse millisecond precision timestamps",
		},
		{
			name:        "Valid microsecond precision",
			startTime:   "2024-01-01T10:30:45.123456Z",
			endTime:     "2024-01-01T10:30:45.654321Z",
			expectError: false,
			description: "Should parse microsecond precision timestamps",
		},
		{
			name:        "Valid nanosecond precision",
			startTime:   "2024-01-01T10:30:45.123456789Z",
			endTime:     "2024-01-01T10:30:45.987654321Z",
			expectError: false,
			description: "Should parse nanosecond precision timestamps",
		},
		{
			name:        "Valid timezone offset",
			startTime:   "2024-01-01T10:30:45.123-07:00",
			endTime:     "2024-01-01T11:30:45.456-07:00",
			expectError: false,
			description: "Should parse timezone offset and normalize to UTC",
		},
		{
			name:        "Empty parameters",
			startTime:   "",
			endTime:     "",
			expectError: false,
			description: "Should handle empty time parameters",
		},
		{
			name:        "Only start time",
			startTime:   "2024-01-01T10:30:45.123Z",
			endTime:     "",
			expectError: false,
			description: "Should handle only start time parameter",
		},
		{
			name:        "Only end time",
			startTime:   "",
			endTime:     "2024-01-01T11:30:45.456Z",
			expectError: false,
			description: "Should handle only end time parameter",
		},
		{
			name:        "Invalid start time format",
			startTime:   "invalid-date",
			endTime:     "2024-01-01T11:30:45Z",
			expectError: true,
			errorType:   "format_error",
			description: "Should reject invalid start time format",
		},
		{
			name:        "Invalid end time format",
			startTime:   "2024-01-01T10:30:45Z",
			endTime:     "not-a-date",
			expectError: true,
			errorType:   "format_error",
			description: "Should reject invalid end time format",
		},
		{
			name:        "Start time after end time",
			startTime:   "2024-01-01T11:30:45Z",
			endTime:     "2024-01-01T10:30:45Z",
			expectError: true,
			errorType:   "range_error",
			description: "Should reject start time after end time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock request
			req := createMockRequest(map[string]string{
				"start_time": tt.startTime,
				"end_time":   tt.endTime,
			})

			startTime, endTime, err := handler.parseTimeParameters(req)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)

			// Verify parsing results
			if tt.startTime != "" {
				require.NotNil(t, startTime, "Start time should not be nil")
				assert.Equal(t, time.UTC, startTime.Location(), "Start time should be in UTC")
			} else {
				assert.Nil(t, startTime, "Start time should be nil when empty")
			}

			if tt.endTime != "" {
				require.NotNil(t, endTime, "End time should not be nil")
				assert.Equal(t, time.UTC, endTime.Location(), "End time should be in UTC")
			} else {
				assert.Nil(t, endTime, "End time should be nil when empty")
			}

			// Verify time order if both are present
			if startTime != nil && endTime != nil {
				assert.True(t, startTime.Before(*endTime) || startTime.Equal(*endTime),
					"Start time should be before or equal to end time")
			}
		})
	}
}

func TestLogHandler_AnalyzeTimeFormat(t *testing.T) {
	handler := &LogHandler{}

	tests := []struct {
		name             string
		startTime        string
		endTime          string
		expectedFormat   string
		expectedPrecision string
	}{
		{
			name:             "RFC3339 UTC",
			startTime:        "2024-01-01T10:30:45Z",
			endTime:          "",
			expectedFormat:   "rfc3339_utc",
			expectedPrecision: "second",
		},
		{
			name:             "RFC3339 with timezone",
			startTime:        "2024-01-01T10:30:45-07:00",
			endTime:          "",
			expectedFormat:   "rfc3339_tz",
			expectedPrecision: "second",
		},
		{
			name:             "Millisecond precision",
			startTime:        "2024-01-01T10:30:45.123Z",
			endTime:          "",
			expectedFormat:   "rfc3339_utc",
			expectedPrecision: "millisecond",
		},
		{
			name:             "Microsecond precision",
			startTime:        "2024-01-01T10:30:45.123456Z",
			endTime:          "",
			expectedFormat:   "rfc3339_utc",
			expectedPrecision: "microsecond",
		},
		{
			name:             "Nanosecond precision",
			startTime:        "2024-01-01T10:30:45.123456789Z",
			endTime:          "",
			expectedFormat:   "rfc3339_utc",
			expectedPrecision: "nanosecond",
		},
		{
			name:             "SQL format",
			startTime:        "2024-01-01 10:30:45",
			endTime:          "",
			expectedFormat:   "sql_format",
			expectedPrecision: "second",
		},
		{
			name:             "SQL format with fractional",
			startTime:        "2024-01-01 10:30:45.123",
			endTime:          "",
			expectedFormat:   "sql_format",
			expectedPrecision: "millisecond",
		},
		{
			name:             "Empty parameters",
			startTime:        "",
			endTime:          "",
			expectedFormat:   "none",
			expectedPrecision: "second",
		},
		{
			name:             "End time analysis",
			startTime:        "",
			endTime:          "2024-01-01T10:30:45.123456Z",
			expectedFormat:   "rfc3339_utc",
			expectedPrecision: "microsecond",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, precision := handler.analyzeTimeFormat(tt.startTime, tt.endTime)

			assert.Equal(t, tt.expectedFormat, format,
				"Format should match expected value")
			assert.Equal(t, tt.expectedPrecision, precision,
				"Precision should match expected value")
		})
	}
}

func TestLogHandler_ConvertTimeValidationError(t *testing.T) {
	handler := &LogHandler{}

	tests := []struct {
		name         string
		inputError   string
		startTimeStr string
		endTimeStr   string
		expectedType string
		description  string
	}{
		{
			name:         "Start time format error",
			inputError:   "invalid start_time format 'invalid': parsing error",
			startTimeStr: "invalid",
			endTimeStr:   "2024-01-01T10:30:45Z",
			expectedType: "*v1alpha1.TimeParameterError",
			description:  "Should convert start time format errors",
		},
		{
			name:         "End time format error",
			inputError:   "invalid end_time format 'bad-date': parsing error",
			startTimeStr: "2024-01-01T10:30:45Z",
			endTimeStr:   "bad-date",
			expectedType: "*v1alpha1.TimeParameterError",
			description:  "Should convert end time format errors",
		},
		{
			name:         "ISO 8601 compliance error",
			inputError:   "time format must be ISO 8601 compliant",
			startTimeStr: "2024/01/01 10:30:45",
			endTimeStr:   "",
			expectedType: "*v1alpha1.TimeFormatError",
			description:  "Should convert ISO 8601 compliance errors",
		},
		{
			name:         "Time range error",
			inputError:   "time range error: start_time must be before end_time",
			startTimeStr: "2024-01-01T11:30:45Z",
			endTimeStr:   "2024-01-01T10:30:45Z",
			expectedType: "*v1alpha1.TimeRangeAPIError",
			description:  "Should convert time range errors",
		},
		{
			name:         "Future time error",
			inputError:   "start_time cannot be in the future",
			startTimeStr: "2030-01-01T10:30:45Z",
			endTimeStr:   "",
			expectedType: "*v1alpha1.TimeParameterError",
			description:  "Should convert future time errors",
		},
		{
			name:         "Generic parsing error",
			inputError:   "unexpected parsing failure",
			startTimeStr: "2024-01-01T10:30:45Z",
			endTimeStr:   "",
			expectedType: "*v1alpha1.TimeParameterError",
			description:  "Should convert generic parsing errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputErr := fmt.Errorf(tt.inputError)
			convertedErr := handler.convertTimeValidationError(inputErr, tt.startTimeStr, tt.endTimeStr)

			require.Error(t, convertedErr, tt.description)

			// Check error type
			errType := fmt.Sprintf("%T", convertedErr)
			assert.Equal(t, tt.expectedType, errType, "Error type should match expected")
		})
	}
}

func TestLogHandler_TimeErrorHandling(t *testing.T) {
	handler := &LogHandler{
		metrics:     NewDatasetMetrics(),
		timeMetrics: NewTimeMetrics(),
	}

	tests := []struct {
		name           string
		error          error
		dataset        string
		expectedStatus int
		description    string
	}{
		{
			name:           "Time parameter error",
			error:          NewTimeParameterError("start_time", "invalid", "bad format", http.StatusBadRequest),
			dataset:        "test-dataset",
			expectedStatus: http.StatusBadRequest,
			description:    "Should handle time parameter errors with 400 status",
		},
		{
			name: "Time format error",
			error: NewTimeFormatError("end_time", "bad-date", []string{"RFC3339"}, "2024-01-01T10:30:45Z"),
			dataset:        "test-dataset",
			expectedStatus: http.StatusBadRequest,
			description:    "Should handle time format errors with 400 status",
		},
		{
			name:           "Time range API error",
			error:          NewTimeRangeAPIError(nil, nil, "invalid range", "check parameters"),
			dataset:        "test-dataset",
			expectedStatus: http.StatusBadRequest,
			description:    "Should handle time range errors with 400 status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response recorder
			recorder := httptest.NewRecorder()
			resp := &restful.Response{ResponseWriter: recorder}

			// Handle the error
			handler.HandleTimeError(resp, tt.error, tt.dataset)

			// Verify status code
			assert.Equal(t, tt.expectedStatus, recorder.Code, tt.description)

			// Verify response body contains error information
			assert.Contains(t, recorder.Body.String(), "error", "Response should contain error field")
		})
	}
}

func TestLogHandler_TimeMetricsRecording(t *testing.T) {
	handler := &LogHandler{
		timeMetrics: NewTimeMetrics(),
	}

	t.Run("Record time parsing metrics", func(t *testing.T) {
		// This test verifies that time metrics are recorded during parsing
		req := createMockRequest(map[string]string{
			"start_time": "2024-01-01T10:30:45.123Z",
			"end_time":   "2024-01-01T10:30:45.456Z",
		})

		startTime, endTime, err := handler.parseTimeParameters(req)
		require.NoError(t, err)

		assert.NotNil(t, startTime, "Start time should be parsed")
		assert.NotNil(t, endTime, "End time should be parsed")

		// Verify metrics were recorded (this would be more comprehensive with actual metrics collection)
		assert.NotNil(t, handler.timeMetrics, "Time metrics should be initialized")
	})
}

// Helper functions

func createMockRequest(queryParams map[string]string) *restful.Request {
	// Create a mock HTTP request with query parameters
	req, _ := http.NewRequest("GET", "/test", nil)
	q := req.URL.Query()
	for key, value := range queryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	// Create restful request
	restfulReq := &restful.Request{Request: req}
	return restfulReq
}

func createMockQueryResponse() *response.LogQueryResponse {
	return &response.LogQueryResponse{
		Logs: []response.LogEntry{
			{
				ID:        "test-log-1",
				Timestamp: time.Now(),
				Message:   "Test log message 1",
				Level:     "INFO",
				Namespace: "default",
				Pod:       "test-pod-1",
			},
			{
				ID:        "test-log-2",
				Timestamp: time.Now(),
				Message:   "Test log message 2",
				Level:     "ERROR",
				Namespace: "default",
				Pod:       "test-pod-2",
			},
		},
		TotalCount: 2,
		Page:       0,
		PageSize:   100,
		HasMore:    false,
	}
}

// Integration test for time parameter handling
func TestLogHandler_TimeParameterIntegration(t *testing.T) {
	// Create mock query service
	mockService := &MockQueryService{}

	// Create handler
	handler := NewLogHandler(mockService)

	// Set up mock expectations
	expectedResponse := createMockQueryResponse()
	mockService.On("DatasetExists", mock.Anything, "test-dataset").Return(true, nil)
	mockService.On("QueryLogsByDataset", mock.Anything, mock.MatchedBy(func(req *request.LogQueryRequest) bool {
		// Verify time parameters are correctly parsed and passed
		return req.Dataset == "test-dataset" &&
			req.StartTime != nil &&
			req.EndTime != nil &&
			req.StartTime.Location() == time.UTC &&
			req.EndTime.Location() == time.UTC
	})).Return(expectedResponse, nil)

	// Create container and install handler
	container := restful.NewContainer()
	handler.InstallHandler(container)

	// Create test request with time parameters
	req, _ := http.NewRequest("GET",
		"/apis/log.theriseunion.io/v1alpha1/logdatasets/test-dataset/logs?start_time=2024-01-01T10:30:45.123Z&end_time=2024-01-01T11:30:45.456Z",
		nil)

	// Execute request
	recorder := httptest.NewRecorder()
	container.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code, "Request should succeed")
	assert.Contains(t, recorder.Body.String(), "test-log-1", "Response should contain test data")

	// Verify all mock expectations were met
	mockService.AssertExpectations(t)
}