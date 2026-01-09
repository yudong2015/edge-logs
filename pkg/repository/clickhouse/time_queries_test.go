package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

func TestTimeQueryBuilder_BuildOptimizedTimeRangeQuery(t *testing.T) {
	tqb := NewTimeQueryBuilder()

	tests := []struct {
		name        string
		req         *request.LogQueryRequest
		expectError bool
		description string
	}{
		{
			name: "Basic time range query",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45Z"),
				EndTime:   timePtr("2024-01-01T11:30:45Z"),
				PageSize:  100,
				Page:      0,
				Direction: "desc",
			},
			expectError: false,
			description: "Should build basic time range query successfully",
		},
		{
			name: "Millisecond precision query",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45.123Z"),
				EndTime:   timePtr("2024-01-01T10:30:45.456Z"),
				PageSize:  50,
				Page:      0,
				Direction: "asc",
			},
			expectError: false,
			description: "Should handle millisecond precision correctly",
		},
		{
			name: "Nanosecond precision query",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45.123456789Z"),
				EndTime:   timePtr("2024-01-01T10:30:45.123456999Z"),
				PageSize:  10,
				Page:      0,
				Direction: "desc",
			},
			expectError: false,
			description: "Should handle nanosecond precision correctly",
		},
		{
			name: "Query with additional filters",
			req: &request.LogQueryRequest{
				Dataset:       "test-dataset",
				StartTime:     timePtr("2024-01-01T10:30:45Z"),
				EndTime:       timePtr("2024-01-01T11:30:45Z"),
				Namespace:     "default",
				PodName:       "test-pod",
				Filter:        "error",
				Severity:      "ERROR",
				ContainerName: "app",
				PageSize:      100,
				Page:          1,
			},
			expectError: false,
			description: "Should handle complex queries with multiple filters",
		},
		{
			name: "Query without time range",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				Filter:    "warning",
				Namespace: "system",
				PageSize:  100,
				Page:      0,
			},
			expectError: false,
			description: "Should handle queries without time constraints",
		},
		{
			name: "Only start time",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45.123Z"),
				PageSize:  100,
				Page:      0,
			},
			expectError: false,
			description: "Should handle queries with only start time",
		},
		{
			name: "Only end time",
			req: &request.LogQueryRequest{
				Dataset:  "test-dataset",
				EndTime:  timePtr("2024-01-01T11:30:45.456Z"),
				PageSize: 100,
				Page:     0,
			},
			expectError: false,
			description: "Should handle queries with only end time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := tqb.BuildOptimizedTimeRangeQuery(tt.req)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)
			assert.NotEmpty(t, query, "Query should not be empty")
			assert.NotNil(t, args, "Args should not be nil")

			// Verify query structure
			assert.Contains(t, query, "SELECT", "Query should contain SELECT")
			assert.Contains(t, query, "FROM logs", "Query should select from logs table")
			assert.Contains(t, query, "dataset = ?", "Query should filter by dataset")

			// Verify time conditions are present when expected
			if tt.req.StartTime != nil {
				assert.Contains(t, query, "timestamp >= toDateTime64(?, 9)",
					"Query should contain start time condition")
			}
			if tt.req.EndTime != nil {
				assert.Contains(t, query, "timestamp <= toDateTime64(?, 9)",
					"Query should contain end time condition")
			}

			// Verify ordering
			assert.Contains(t, query, "ORDER BY timestamp", "Query should order by timestamp")

			// Verify pagination
			if tt.req.PageSize > 0 {
				assert.Contains(t, query, "LIMIT", "Query should contain LIMIT")
				if tt.req.Page > 0 {
					assert.Contains(t, query, "OFFSET", "Query should contain OFFSET")
				}
			}

			// Verify additional filters
			if tt.req.Namespace != "" {
				assert.Contains(t, query, "k8s_namespace_name = ?",
					"Query should contain namespace filter")
			}
			if tt.req.Filter != "" {
				assert.Contains(t, query, "positionCaseInsensitive(content, ?) > 0",
					"Query should contain content filter")
			}

			// Reset query builder for next iteration
			tqb.Reset()
		})
	}
}

func TestTimeQueryBuilder_AddTimeCondition(t *testing.T) {
	tqb := NewTimeQueryBuilder()

	tests := []struct {
		name     string
		operator string
		time     time.Time
		expected string
	}{
		{
			name:     "Greater than or equal",
			operator: ">=",
			time:     parseTime(t, "2024-01-01T10:30:45.123Z"),
			expected: "timestamp >= toDateTime64(?, 9)",
		},
		{
			name:     "Less than or equal",
			operator: "<=",
			time:     parseTime(t, "2024-01-01T11:30:45.456Z"),
			expected: "timestamp <= toDateTime64(?, 9)",
		},
		{
			name:     "Equal",
			operator: "=",
			time:     parseTime(t, "2024-01-01T10:30:45.123456789Z"),
			expected: "timestamp = toDateTime64(?, 9)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tqb.Reset()
			tqb.AddTimeCondition(tt.operator, tt.time)

			assert.Contains(t, tqb.conditions, tt.expected,
				"Should add correct time condition")

			// Verify the time argument is correctly converted
			require.Len(t, tqb.args, 1, "Should have one argument")

			arg, ok := tqb.args[0].(float64)
			require.True(t, ok, "Argument should be float64")

			// Verify precision is preserved
			expectedSeconds := float64(tt.time.UnixNano()) / 1e9
			assert.InDelta(t, expectedSeconds, arg, 0.000000001,
				"Time conversion should preserve nanosecond precision")
		})
	}
}

func TestTimeQueryBuilder_ValidateTimeQuery(t *testing.T) {
	tqb := NewTimeQueryBuilder()

	tests := []struct {
		name        string
		req         *request.LogQueryRequest
		expectError bool
		description string
	}{
		{
			name: "Valid time range",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45Z"),
				EndTime:   timePtr("2024-01-01T11:30:45Z"),
			},
			expectError: false,
			description: "Valid time range should pass validation",
		},
		{
			name: "Time range too large",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45Z"),
				EndTime:   timePtr("2024-01-02T11:30:45Z"), // > 24 hours
			},
			expectError: true,
			description: "Time range exceeding 7 days should fail validation",
		},
		{
			name: "Start time after end time",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T11:30:45Z"),
				EndTime:   timePtr("2024-01-01T10:30:45Z"),
			},
			expectError: true,
			description: "Start time after end time should fail validation",
		},
		{
			name: "No time range",
			req: &request.LogQueryRequest{
				Dataset: "test-dataset",
			},
			expectError: false,
			description: "Query without time range should pass validation with warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tqb.ValidateTimeQuery(tt.req)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestTimeQueryBuilder_GetTimeQueryMetrics(t *testing.T) {
	tqb := NewTimeQueryBuilder()

	tests := []struct {
		name        string
		req         *request.LogQueryRequest
		expectedMetrics map[string]interface{}
	}{
		{
			name: "Sub-hour query",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45Z"),
				EndTime:   timePtr("2024-01-01T10:45:45Z"), // 15 minutes
				Filter:    "error",
			},
			expectedMetrics: map[string]interface{}{
				"complexity":         "low",
				"has_content_filter": true,
			},
		},
		{
			name: "Multi-hour query",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45Z"),
				EndTime:   timePtr("2024-01-01T16:30:45Z"), // 6 hours
			},
			expectedMetrics: map[string]interface{}{
				"complexity":         "medium",
				"has_content_filter": false,
			},
		},
		{
			name: "Daily query",
			req: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				StartTime: timePtr("2024-01-01T10:30:45Z"),
				EndTime:   timePtr("2024-01-02T10:30:45Z"), // 24 hours
			},
			expectedMetrics: map[string]interface{}{
				"complexity":         "high",
				"has_content_filter": false,
			},
		},
		{
			name: "Unbounded query",
			req: &request.LogQueryRequest{
				Dataset: "test-dataset",
			},
			expectedMetrics: map[string]interface{}{
				"complexity":         "unbounded",
				"has_content_filter": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add some conditions to test filter count
			if tt.req.Filter != "" {
				tqb.AddCondition("positionCaseInsensitive(content, ?) > 0", tt.req.Filter)
			}

			metrics := tqb.GetTimeQueryMetrics(tt.req)

			for key, expectedValue := range tt.expectedMetrics {
				assert.Equal(t, expectedValue, metrics[key],
					"Metric %s should match expected value", key)
			}

			// Verify additional metrics are present
			assert.Contains(t, metrics, "filter_count", "Should contain filter count")

			if tt.req.StartTime != nil && tt.req.EndTime != nil {
				assert.Contains(t, metrics, "time_span_seconds", "Should contain time span in seconds")
				assert.Contains(t, metrics, "time_span_hours", "Should contain time span in hours")
				assert.Contains(t, metrics, "estimated_partitions", "Should contain partition estimate")
			}

			// Reset for next iteration
			tqb.Reset()
		})
	}
}

func TestTimeQueryBuilder_Performance(t *testing.T) {
	tqb := NewTimeQueryBuilder()

	// Test query building performance
	req := &request.LogQueryRequest{
		Dataset:       "performance-test-dataset",
		StartTime:     timePtr("2024-01-01T10:30:45.123456789Z"),
		EndTime:       timePtr("2024-01-01T11:30:45.987654321Z"),
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		ContainerName: "test-container",
		Filter:        "performance test",
		Severity:      "INFO",
		PageSize:      1000,
		Page:          5,
		Tags: map[string]string{
			"environment": "test",
			"version":     "1.0.0",
		},
	}

	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		query, args, err := tqb.BuildOptimizedTimeRangeQuery(req)
		require.NoError(t, err)
		assert.NotEmpty(t, query)
		assert.NotNil(t, args)

		tqb.Reset()
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	t.Logf("Query building performance: %d iterations in %v (avg: %v)",
		iterations, duration, avgDuration)

	// Performance should be reasonable (< 1ms per query build on average)
	assert.Less(t, avgDuration.Milliseconds(), int64(1),
		"Query building should be performant")
}

// Helper functions

func timePtr(timeStr string) *time.Time {
	t, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		panic(err)
	}
	return &t
}

func parseTime(t *testing.T, timeStr string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, timeStr)
	require.NoError(t, err)
	return parsed
}

// Benchmark tests
func BenchmarkTimeQueryBuilder_BuildOptimizedTimeRangeQuery(b *testing.B) {
	tqb := NewTimeQueryBuilder()

	benchmarks := []struct {
		name string
		req  *request.LogQueryRequest
	}{
		{
			name: "Simple",
			req: &request.LogQueryRequest{
				Dataset:   "bench-dataset",
				StartTime: timePtr("2024-01-01T10:30:45Z"),
				EndTime:   timePtr("2024-01-01T11:30:45Z"),
				PageSize:  100,
			},
		},
		{
			name: "Complex",
			req: &request.LogQueryRequest{
				Dataset:       "bench-dataset",
				StartTime:     timePtr("2024-01-01T10:30:45.123456789Z"),
				EndTime:       timePtr("2024-01-01T11:30:45.987654321Z"),
				Namespace:     "benchmark",
				PodName:       "bench-pod",
				ContainerName: "bench-container",
				Filter:        "benchmark query",
				Severity:      "INFO",
				PageSize:      500,
				Page:          2,
				Tags: map[string]string{
					"test": "benchmark",
					"type": "performance",
				},
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, err := tqb.BuildOptimizedTimeRangeQuery(bm.req)
				if err != nil {
					b.Fatal(err)
				}
				tqb.Reset()
			}
		})
	}
}