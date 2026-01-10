package query

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/outpostos/edge-logs/pkg/model/clickhouse"
	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/model/response"
	clickhouseRepo "github.com/outpostos/edge-logs/pkg/repository/clickhouse"
)

// MockRepository is a mock implementation of clickhouse.Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) QueryLogs(ctx context.Context, req *request.LogQueryRequest) ([]clickhouse.LogEntry, int, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int), args.Error(2)
	}
	return args.Get(0).([]clickhouse.LogEntry), args.Get(1).(int), args.Error(2)
}

func (m *MockRepository) InsertLog(ctx context.Context, log *clickhouse.LogEntry) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockRepository) InsertLogsBatch(ctx context.Context, logs []clickhouse.LogEntry) error {
	args := m.Called(ctx, logs)
	return args.Error(0)
}

func (m *MockRepository) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Dataset-specific methods for enhanced data isolation
func (m *MockRepository) DatasetExists(ctx context.Context, dataset string) (bool, error) {
	args := m.Called(ctx, dataset)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetDatasetStats(ctx context.Context, dataset string) (*clickhouseRepo.DatasetMetadata, error) {
	args := m.Called(ctx, dataset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clickhouseRepo.DatasetMetadata), args.Error(1)
}

func (m *MockRepository) ListAvailableDatasets(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRepository) GetDatasetHealth(ctx context.Context, dataset string) (*clickhouseRepo.DatasetHealth, error) {
	args := m.Called(ctx, dataset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clickhouseRepo.DatasetHealth), args.Error(1)
}

func (m *MockRepository) QueryAggregation(ctx context.Context, req *request.AggregationRequest) (*response.AggregationResponse, error) {
	// Type assertion to ensure response.AggregationResponse is used
	var respType *response.AggregationResponse
	_ = respType
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*response.AggregationResponse), args.Error(1)
}

// Test NewService
func TestNewService(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
}

// Test QueryLogs - basic successful query
func TestQueryLogs_BasicQuery(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	now := time.Now()
	startTime := now.Add(-24 * time.Hour)

	req := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		StartTime: &startTime,
		EndTime:   &now,
		Page:      0,
		PageSize:  10,
	}

	expectedLogs := []clickhouse.LogEntry{
		{
			Timestamp:     now,
			Dataset:       "test-dataset",
			Content:       "Test log message",
			Severity:      "INFO",
			K8sNamespace:  "default",
			K8sPodName:    "test-pod",
			ContainerName: "test-container",
			HostIP:        "192.168.1.1",
			Tags:          map[string]string{"env": "test"},
		},
	}

	mockRepo.On("QueryLogs", mock.Anything, req).Return(expectedLogs, 1, nil)

	response, err := service.QueryLogs(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Logs, 1)
	assert.Equal(t, 1, response.TotalCount)
	assert.Equal(t, 0, response.Page)
	assert.Equal(t, 10, response.PageSize)
	assert.False(t, response.HasMore)

	mockRepo.AssertExpectations(t)
}

// Test QueryLogs - validation errors
func TestQueryLogs_ValidationError(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	tests := []struct {
		name    string
		req     *request.LogQueryRequest
		wantErr string
	}{
		{
			name: "missing dataset",
			req: &request.LogQueryRequest{
				Dataset: "",
			},
			wantErr: "dataset is required",
		},
		{
			name: "invalid time range",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: &[]time.Time{time.Now()}[0],
				EndTime:   &[]time.Time{time.Now().Add(-24 * time.Hour)}[0],
				PageSize:  10,
			},
			wantErr: "start_time must be before end_time",
		},
		{
			name: "negative page size",
			req: &request.LogQueryRequest{
				Dataset:  "test-dataset",
				PageSize: -1,
			},
			wantErr: "page_size must be non-negative",
		},
		{
			name: "filter too short",
			req: &request.LogQueryRequest{
				Dataset:  "test-dataset",
				Filter:   "a",
				PageSize: 10,
			},
			wantErr: "filter too short",
		},
		{
			name: "filter too long",
			req: &request.LogQueryRequest{
				Dataset:  "test-dataset",
				Filter:   string(make([]byte, 1001)),
				PageSize: 10,
			},
			wantErr: "filter too long",
		},
		{
			name: "SQL injection attempt",
			req: &request.LogQueryRequest{
				Dataset:  "test-dataset",
				Filter:   "'; DROP TABLE logs; --",
				PageSize: 10,
			},
			wantErr: "filter contains potentially harmful content",
		},
		{
			name: "time range too large",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: &[]time.Time{time.Now().Add(-8 * 24 * time.Hour)}[0],
				EndTime:   &[]time.Time{time.Now()}[0],
				PageSize:  10,
			},
			wantErr: "time range too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.QueryLogs(context.Background(), tt.req)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// Test QueryLogs - repository errors
func TestQueryLogs_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	req := &request.LogQueryRequest{
		Dataset:  "test-dataset",
		PageSize: 10,
	}

	mockRepo.On("QueryLogs", mock.Anything, req).Return(
		([]clickhouse.LogEntry)(nil), 0, errors.New("connection failed"))

	_, err := service.QueryLogs(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository operation failed")
	assert.Contains(t, err.Error(), "connection failed")

	mockRepo.AssertExpectations(t)
}

// Test QueryLogs - default time range
func TestQueryLogs_DefaultTimeRange(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	req := &request.LogQueryRequest{
		Dataset:  "test-dataset",
		PageSize: 10,
	}

	var capturedReq *request.LogQueryRequest
	mockRepo.On("QueryLogs", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		capturedReq = args.Get(1).(*request.LogQueryRequest)
	}).Return([]clickhouse.LogEntry{}, 0, nil)

	_, err := service.QueryLogs(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, capturedReq.StartTime)
	assert.NotNil(t, capturedReq.EndTime)

	duration := capturedReq.EndTime.Sub(*capturedReq.StartTime)
	assert.True(t, duration >= 23*time.Hour && duration <= 25*time.Hour,
		"Default time range should be approximately 24 hours")

	mockRepo.AssertExpectations(t)
}

// Test QueryLogs - pagination and hasMore calculation
func TestQueryLogs_PaginationAndHasMore(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	now := time.Now()

	tests := []struct {
		name            string
		req             *request.LogQueryRequest
		totalLogs       int
		returnedLogs    int
		expectedHasMore bool
	}{
		{
			name: "full page with more results",
			req: &request.LogQueryRequest{
				Dataset:  "test-dataset",
				Page:     0,
				PageSize: 10,
			},
			totalLogs:       25,
			returnedLogs:    10,
			expectedHasMore: true,
		},
		{
			name: "last page",
			req: &request.LogQueryRequest{
				Dataset:  "test-dataset",
				Page:     2,
				PageSize: 10,
			},
			totalLogs:       25,
			returnedLogs:    5,
			expectedHasMore: false,
		},
		{
			name: "exact page size",
			req: &request.LogQueryRequest{
				Dataset:  "test-dataset",
				Page:     0,
				PageSize: 10,
			},
			totalLogs:       10,
			returnedLogs:    10,
			expectedHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := make([]clickhouse.LogEntry, tt.returnedLogs)
			for i := 0; i < tt.returnedLogs; i++ {
				logs[i] = clickhouse.LogEntry{
					Timestamp:     now,
					Dataset:       "test",
					Content:       "Test log",
					Severity:      "INFO",
					K8sNamespace:  "default",
					K8sPodName:    "pod",
					ContainerName: "container",
					HostIP:        "192.168.1.1",
				}
			}

			mockRepo.On("QueryLogs", mock.Anything, tt.req).Return(logs, tt.totalLogs, nil)

			response, err := service.QueryLogs(context.Background(), tt.req)

			assert.NoError(t, err)
			assert.Equal(t, tt.returnedLogs, len(response.Logs))
			assert.Equal(t, tt.totalLogs, response.TotalCount)
			assert.Equal(t, tt.expectedHasMore, response.HasMore)

			mockRepo.AssertExpectations(t)
			mockRepo.ExpectedCalls = nil // Reset for next iteration
		})
	}
}

// Test Helper functions
func TestNormalizeSeverityLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"error", "ERROR"},
		{"ERROR", "ERROR"},
		{"err", "ERROR"},
		{"warn", "WARN"},
		{"WARNING", "WARN"},
		{"info", "INFO"},
		{"debug", "DEBUG"},
		{"DEBUG", "DEBUG"},
		{"custom", "CUSTOM"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeSeverityLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsSQLInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"normal text", "normal search query", false},
		{"SQL injection DROP", "'; DROP TABLE logs; --", true},
		{"SQL injection DELETE", "'); DELETE FROM logs WHERE '1'='1", true},
		{"SQL injection UNION", "' UNION SELECT * FROM users --", true},
		{"SQL injection comment", "test /* comment */", true},
		{"case insensitive", "'; drop table logs; --", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSQLInjection(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnrichLabels(t *testing.T) {
	log := clickhouse.LogEntry{
		Timestamp:   time.Now(),
		Dataset:     "test",
		Content:     "test",
		Severity:    "INFO",
		HostName:    "test-host",
		K8sNodeName: "test-node",
		K8sPodUID:   "test-pod-uid",
		Tags: map[string]string{
			"original": "tag",
		},
	}

	result := enrichLabels(log.Tags, log)

	assert.Equal(t, "tag", result["original"])
	assert.Equal(t, "test-host", result["host_name"])
	assert.Equal(t, "test-node", result["node_name"])
	assert.Equal(t, "test-pod-uid", result["pod_uid"])
}
