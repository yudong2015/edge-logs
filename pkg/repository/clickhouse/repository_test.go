package clickhouse

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/config"
	chModel "github.com/outpostos/edge-logs/pkg/model/clickhouse"
	"github.com/outpostos/edge-logs/pkg/model/request"
)

// TestClickHouseRepository provides unit tests for the ClickHouse repository
func TestClickHouseRepository(t *testing.T) {
	t.Run("NewClickHouseRepository", testNewClickHouseRepository)
	t.Run("ValidateLogEntry", testValidateLogEntry)
	t.Run("QueryBuilder", testQueryBuilder)
	t.Run("ErrorMapping", testErrorMapping)
}

// testNewClickHouseRepository tests repository construction
func testNewClickHouseRepository(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.ClickHouseConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: &config.ClickHouseConfig{
				Host:            "localhost",
				Port:            9000,
				Database:        "test_db",
				Username:        "default",
				Password:        "",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 30 * time.Minute,
				QueryTimeout:    30 * time.Second,
				ExecTimeout:     10 * time.Second,
			},
			expectError: true, // Will fail without real ClickHouse
			errorMsg:    "connection",
		},
		{
			name: "invalid configuration - empty host",
			config: &config.ClickHouseConfig{
				Host:     "",
				Port:     9000,
				Database: "test_db",
			},
			expectError: true,
			errorMsg:    "connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewClickHouseRepository(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.NotNil(t, repo.cm)
				assert.NotNil(t, repo.metrics)
				if repo != nil {
					repo.Close()
				}
			}
		})
	}
}

// testValidateLogEntry tests log entry validation
func testValidateLogEntry(t *testing.T) {
	// Create a mock repository for validation testing
	repo := &ClickHouseRepository{}

	tests := []struct {
		name        string
		logEntry    *chModel.LogEntry
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid log entry",
			logEntry: &chModel.LogEntry{
				Timestamp:     time.Now(),
				Dataset:       "test-dataset",
				Content:       "test log message",
				Severity:      "INFO",
				ContainerID:   "container-123",
				ContainerName: "test-container",
				HostIP:        "192.168.1.100",
				HostName:      "test-host",
				K8sNamespace:  "default",
				K8sPodName:    "test-pod",
				Tags:          map[string]string{"cluster": "test"},
			},
			expectError: false,
		},
		{
			name: "missing dataset",
			logEntry: &chModel.LogEntry{
				Timestamp: time.Now(),
				Content:   "test log message",
			},
			expectError: true,
			errorMsg:    "dataset is required",
		},
		{
			name: "missing timestamp",
			logEntry: &chModel.LogEntry{
				Dataset: "test-dataset",
				Content: "test log message",
			},
			expectError: true,
			errorMsg:    "timestamp is required",
		},
		{
			name: "missing content",
			logEntry: &chModel.LogEntry{
				Timestamp: time.Now(),
				Dataset:   "test-dataset",
			},
			expectError: true,
			errorMsg:    "content is required",
		},
		{
			name: "timestamp too old",
			logEntry: &chModel.LogEntry{
				Timestamp: time.Now().Add(-100 * 24 * time.Hour),
				Dataset:   "test-dataset",
				Content:   "old log message",
			},
			expectError: true,
			errorMsg:    "timestamp is too old",
		},
		{
			name: "timestamp in future",
			logEntry: &chModel.LogEntry{
				Timestamp: time.Now().Add(2 * time.Hour),
				Dataset:   "test-dataset",
				Content:   "future log message",
			},
			expectError: true,
			errorMsg:    "timestamp is too far in the future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.validateLogEntry(tt.logEntry)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testQueryBuilder tests query building functionality
func testQueryBuilder(t *testing.T) {
	qb := NewQueryBuilder()
	require.NotNil(t, qb)

	t.Run("BuildLogQuery - basic", func(t *testing.T) {
		req := &request.LogQueryRequest{
			Dataset:   "test-dataset",
			PageSize:  100,
			OrderBy:   "timestamp",
			Direction: "desc",
		}

		query, args, err := qb.BuildLogQuery(req)
		assert.NoError(t, err)
		assert.Contains(t, query, "FROM logs")
		assert.Contains(t, query, "dataset = ?")
		assert.Contains(t, query, "ORDER BY")
		assert.Contains(t, query, "LIMIT 100")
		assert.Equal(t, "test-dataset", args[0])
	})

	t.Run("BuildLogQuery - with filters", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()

		req := &request.LogQueryRequest{
			Dataset:       "test-dataset",
			StartTime:     &startTime,
			EndTime:       &endTime,
			Namespace:     "default",
			Filter:        "error",
			Severity:      "ERROR",
			HostIP:        "192.168.1.100",
			ContainerName: "test-container",
			Tags:          map[string]string{"cluster": "prod"},
			PageSize:      50,
			Page:          1,
			OrderBy:       "timestamp",
			Direction:     "asc",
		}

		// Reset the query builder for new query
		qb.Reset()
		query, args, err := qb.BuildLogQuery(req)
		assert.NoError(t, err)

		// Verify query structure
		assert.Contains(t, query, "FROM logs")
		assert.Contains(t, query, "dataset = ?")
		assert.Contains(t, query, "timestamp >= ?")
		assert.Contains(t, query, "timestamp <= ?")
		assert.Contains(t, query, "k8s_namespace_name = ?")
		assert.Contains(t, query, "hasToken(content, ?)")
		assert.Contains(t, query, "severity = ?")
		assert.Contains(t, query, "host_ip = ?")
		assert.Contains(t, query, "container_name = ?")
		assert.Contains(t, query, "tags[?] = ?")
		assert.Contains(t, query, "ORDER BY")
		assert.Contains(t, query, "LIMIT 50")
		assert.Contains(t, query, "OFFSET 50")

		// Verify arguments
		expectedArgs := []interface{}{
			"test-dataset", startTime, endTime, "default", "ERROR",
			"192.168.1.100", "test-container", "error", "cluster", "prod",
		}
		assert.Len(t, args, len(expectedArgs))
	})

	t.Run("BuildCountQuery", func(t *testing.T) {
		req := &request.LogQueryRequest{
			Dataset:   "test-dataset",
			Namespace: "default",
			Filter:    "error",
		}

		// Reset the query builder for new query
		qb.Reset()
		query, args, err := qb.BuildCountQuery(req)
		assert.NoError(t, err)

		assert.Contains(t, query, "SELECT count(*)")
		assert.Contains(t, query, "FROM logs")
		assert.Contains(t, query, "dataset = ?")
		assert.Contains(t, query, "k8s_namespace_name = ?")
		assert.Contains(t, query, "hasToken(content, ?)")
		assert.NotContains(t, query, "ORDER BY")
		assert.NotContains(t, query, "LIMIT")

		expectedArgs := []interface{}{"test-dataset", "default", "error"}
		assert.Equal(t, expectedArgs, args)
	})

	t.Run("BuildInsertQuery", func(t *testing.T) {
		query, err := qb.BuildInsertQuery()
		assert.NoError(t, err)
		assert.Contains(t, query, "INSERT INTO logs")
		assert.Contains(t, query, "timestamp, dataset, content, severity")
		assert.Contains(t, query, "VALUES")

		// Count the number of placeholders
		placeholderCount := len(regexp.MustCompile(`\?`).FindAllString(query, -1))
		assert.Equal(t, 14, placeholderCount) // Should match the number of fields
	})

	t.Run("ValidateQuery", func(t *testing.T) {
		tests := []struct {
			name        string
			req         *request.LogQueryRequest
			expectError bool
			errorMsg    string
		}{
			{
				name: "valid query",
				req: &request.LogQueryRequest{
					Dataset:   "test-dataset",
					PageSize:  100,
					OrderBy:   "timestamp",
					Direction: "desc",
				},
				expectError: false,
			},
			{
				name: "missing dataset",
				req: &request.LogQueryRequest{
					PageSize:  100,
					OrderBy:   "timestamp",
					Direction: "desc",
				},
				expectError: true,
				errorMsg:    "dataset is required",
			},
			{
				name: "excessive page size",
				req: &request.LogQueryRequest{
					Dataset:  "test-dataset",
					PageSize: 15000,
				},
				expectError: true,
				errorMsg:    "page_size cannot exceed 10000",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				qb := NewQueryBuilder()
				err := qb.ValidateQuery(tt.req)

				if tt.expectError {
					assert.Error(t, err)
					if tt.errorMsg != "" {
						assert.Contains(t, err.Error(), tt.errorMsg)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

// testErrorMapping tests ClickHouse error mapping
func testErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		originalError  error
		operation      string
		expectedType   ErrorType
		expectedRetry  bool
		expectedFields []string
	}{
		{
			name:           "connection refused error",
			originalError:  fmt.Errorf("connection refused"),
			operation:      "test_op",
			expectedType:   ErrorTypeConnection,
			expectedRetry:  true,
			expectedFields: []string{"type", "retryable", "description"},
		},
		{
			name:           "timeout error",
			originalError:  fmt.Errorf("timeout occurred"),
			operation:      "test_op",
			expectedType:   ErrorTypeTimeout,
			expectedRetry:  true,
			expectedFields: []string{"type", "retryable", "description"},
		},
		{
			name:           "authentication failed error",
			originalError:  fmt.Errorf("authentication failed"),
			operation:      "test_op",
			expectedType:   ErrorTypeAuthentication,
			expectedRetry:  false,
			expectedFields: []string{"type", "retryable", "description"},
		},
		{
			name:           "syntax error",
			originalError:  fmt.Errorf("syntax error near 'SELECT'"),
			operation:      "test_op",
			expectedType:   ErrorTypeSyntax,
			expectedRetry:  false,
			expectedFields: []string{"type", "retryable", "description"},
		},
		{
			name:           "unknown error",
			originalError:  fmt.Errorf("some unknown error"),
			operation:      "test_op",
			expectedType:   ErrorTypeUnknown,
			expectedRetry:  false,
			expectedFields: []string{"type", "retryable", "description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoErr := MapClickHouseError(tt.originalError, tt.operation)

			assert.NotNil(t, repoErr)
			assert.Equal(t, tt.operation, repoErr.Op)
			assert.Equal(t, "logs", repoErr.Table)
			assert.Equal(t, tt.originalError, repoErr.Err)

			// Check error type
			assert.Equal(t, tt.expectedType, repoErr.Context["type"])

			// Check retryable flag
			assert.Equal(t, tt.expectedRetry, repoErr.Context["retryable"])

			// Check all expected fields are present
			for _, field := range tt.expectedFields {
				assert.Contains(t, repoErr.Context, field)
			}

			// Test utility functions
			assert.Equal(t, tt.expectedRetry, IsRetryableError(repoErr))
			assert.Equal(t, tt.expectedType, GetErrorType(repoErr))
		})
	}

	t.Run("nil error", func(t *testing.T) {
		repoErr := MapClickHouseError(nil, "test_op")
		assert.Nil(t, repoErr)
	})

	t.Run("NewValidationError", func(t *testing.T) {
		err := NewValidationError("test_op", "validation message")
		assert.NotNil(t, err)
		assert.Equal(t, "test_op", err.Op)
		assert.Equal(t, ErrorTypeValidation, err.Context["type"])
		assert.Equal(t, false, err.Context["retryable"])
		assert.Contains(t, err.Error(), "validation message")
	})

	t.Run("NewQueryError", func(t *testing.T) {
		originalErr := fmt.Errorf("query failed")
		err := NewQueryError("test_op", "SELECT * FROM logs", originalErr)
		assert.NotNil(t, err)
		assert.Equal(t, "test_op", err.Op)
		assert.Equal(t, ErrorTypeQuery, err.Context["type"])
		assert.Contains(t, err.Context, "query")
		assert.Contains(t, err.Error(), "query execution failed")
	})

	t.Run("NewDataFormatError", func(t *testing.T) {
		originalErr := fmt.Errorf("invalid format")
		err := NewDataFormatError("test_op", "timestamp", originalErr)
		assert.NotNil(t, err)
		assert.Equal(t, "test_op", err.Op)
		assert.Equal(t, ErrorTypeDataFormat, err.Context["type"])
		assert.Equal(t, "timestamp", err.Context["field"])
		assert.Equal(t, false, err.Context["retryable"])
		assert.Contains(t, err.Error(), "timestamp")
	})
}

// TestQueryMetricsCollector tests query metrics collection
func TestQueryMetricsCollector(t *testing.T) {
	t.Run("NewQueryMetricsCollector", func(t *testing.T) {
		collector := NewQueryMetricsCollector("test-dataset", "search", "{\"test\":true}")
		assert.NotNil(t, collector)
		assert.Equal(t, "test-dataset", collector.dataset)
		assert.Equal(t, "search", collector.queryType)
		assert.Equal(t, "{\"test\":true}", collector.queryParams)
		assert.True(t, collector.startTime.Before(time.Now().Add(1*time.Second)))
	})

	t.Run("Finish with nil metrics recorder", func(t *testing.T) {
		collector := NewQueryMetricsCollector("test-dataset", "search", "{}")
		// Should not panic with nil metrics recorder
		assert.NotPanics(t, func() {
			collector.Finish(nil, 100)
		})
	})
}

// TestConnectionManager tests connection management (unit tests only, no real connections)
func TestConnectionManager(t *testing.T) {
	t.Run("Stats", func(t *testing.T) {
		// Test with a mock configuration that won't actually connect
		cfg := &config.ClickHouseConfig{
			Host:            "nonexistent-host",
			Port:            9999,
			Database:        "test",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			QueryTimeout:    30 * time.Second,
		}

		// This will fail to connect, but we can test the configuration handling
		_, err := NewConnectionManager(cfg)
		assert.Error(t, err) // Expected since host doesn't exist

		// Test error mapping for connection failures
		mappedErr := MapClickHouseError(err, "connection_test")
		assert.NotNil(t, mappedErr)
		assert.Equal(t, "connection_test", mappedErr.Op)
	})
}
