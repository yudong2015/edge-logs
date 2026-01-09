package query

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeRangeValidator_ValidateAndParseTimeRange(t *testing.T) {
	validator := NewTimeRangeValidator()

	tests := []struct {
		name        string
		startTime   string
		endTime     string
		wantErr     bool
		expectStart *time.Time
		expectEnd   *time.Time
		description string
	}{
		{
			name:        "Valid RFC3339 range",
			startTime:   "2024-01-01T10:30:45Z",
			endTime:     "2024-01-01T10:31:45Z",
			wantErr:     false,
			description: "Basic RFC3339 format should parse successfully",
		},
		{
			name:        "Valid millisecond precision",
			startTime:   "2024-01-01T10:30:45.123Z",
			endTime:     "2024-01-01T10:30:45.456Z",
			wantErr:     false,
			description: "Millisecond precision should be supported",
		},
		{
			name:        "Valid microsecond precision",
			startTime:   "2024-01-01T10:30:45.123456Z",
			endTime:     "2024-01-01T10:30:45.123789Z",
			wantErr:     false,
			description: "Microsecond precision should be supported",
		},
		{
			name:        "Valid nanosecond precision",
			startTime:   "2024-01-01T10:30:45.123456789Z",
			endTime:     "2024-01-01T10:30:45.123456999Z",
			wantErr:     false,
			description: "Nanosecond precision should be supported",
		},
		{
			name:        "Valid timezone offset",
			startTime:   "2024-01-01T10:30:45.123-07:00",
			endTime:     "2024-01-01T10:31:45.456-07:00",
			wantErr:     false,
			description: "Timezone offset should be normalized to UTC",
		},
		{
			name:        "Valid SQL format",
			startTime:   "2024-01-01 10:30:45",
			endTime:     "2024-01-01 10:31:45",
			wantErr:     false,
			description: "SQL format should be supported",
		},
		{
			name:        "Invalid format",
			startTime:   "not-a-date",
			endTime:     "2024-01-01T10:31:45Z",
			wantErr:     true,
			description: "Invalid date format should return error",
		},
		{
			name:        "Start after end",
			startTime:   "2024-01-01T10:31:45Z",
			endTime:     "2024-01-01T10:30:45Z",
			wantErr:     true,
			description: "Start time after end time should return error",
		},
		{
			name:        "Too large time span",
			startTime:   "2024-01-01T10:30:45Z",
			endTime:     "2024-01-02T10:30:46Z", // > 24 hours
			wantErr:     true,
			description: "Time span exceeding 24 hours should return error",
		},
		{
			name:        "Only start time",
			startTime:   "2024-01-01T10:30:45.123Z",
			endTime:     "",
			wantErr:     false,
			description: "Only start time should be valid",
		},
		{
			name:        "Only end time",
			startTime:   "",
			endTime:     "2024-01-01T10:30:45.123Z",
			wantErr:     false,
			description: "Only end time should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime, endTime, err := validator.ValidateAndParseTimeRange(tt.startTime, tt.endTime)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)

			// Verify time parsing results
			if tt.startTime != "" {
				require.NotNil(t, startTime, "Start time should not be nil")
				assert.Equal(t, time.UTC, startTime.Location(), "Start time should be normalized to UTC")
			} else {
				assert.Nil(t, startTime, "Start time should be nil when empty")
			}

			if tt.endTime != "" {
				require.NotNil(t, endTime, "End time should not be nil")
				assert.Equal(t, time.UTC, endTime.Location(), "End time should be normalized to UTC")
			} else {
				assert.Nil(t, endTime, "End time should be nil when empty")
			}

			// Verify time order
			if startTime != nil && endTime != nil {
				assert.True(t, startTime.Before(*endTime) || startTime.Equal(*endTime),
					"Start time should be before or equal to end time")
			}
		})
	}
}

func TestTimeRangeValidator_ParseTimeString(t *testing.T) {
	validator := NewTimeRangeValidator()

	tests := []struct {
		name             string
		timeStr          string
		wantErr          bool
		expectedPrecision string
		description      string
	}{
		{
			name:             "RFC3339 basic",
			timeStr:          "2024-01-01T10:30:45Z",
			wantErr:          false,
			expectedPrecision: "second",
			description:      "Basic RFC3339 should parse to second precision",
		},
		{
			name:             "Millisecond precision",
			timeStr:          "2024-01-01T10:30:45.123Z",
			wantErr:          false,
			expectedPrecision: "millisecond",
			description:      "Should preserve millisecond precision",
		},
		{
			name:             "Microsecond precision",
			timeStr:          "2024-01-01T10:30:45.123456Z",
			wantErr:          false,
			expectedPrecision: "microsecond",
			description:      "Should preserve microsecond precision",
		},
		{
			name:             "Nanosecond precision",
			timeStr:          "2024-01-01T10:30:45.123456789Z",
			wantErr:          false,
			expectedPrecision: "nanosecond",
			description:      "Should preserve nanosecond precision",
		},
		{
			name:             "Variable precision",
			timeStr:          "2024-01-01T10:30:45.12Z",
			wantErr:          false,
			expectedPrecision: "centisecond",
			description:      "Should handle variable precision",
		},
		{
			name:        "Invalid format",
			timeStr:     "2024/01/01 10:30:45",
			wantErr:     true,
			description: "Non-ISO format should fail",
		},
		{
			name:        "Empty string",
			timeStr:     "",
			wantErr:     true,
			description: "Empty string should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedTime, err := validator.parseTimeString(tt.timeStr)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)
			require.NotNil(t, parsedTime, "Parsed time should not be nil")

			// Verify UTC normalization
			assert.Equal(t, time.UTC, parsedTime.Location(), "Time should be normalized to UTC")

			// Verify precision is preserved (nanoseconds should contain fractional second information)
			if tt.expectedPrecision != "second" {
				nanosec := parsedTime.Nanosecond()
				switch tt.expectedPrecision {
				case "millisecond":
					// For millisecond precision, nanoseconds should show millisecond info
					millisPart := (nanosec / 1000000) * 1000000
					assert.NotZero(t, millisPart, "Millisecond precision should be preserved")
				case "microsecond":
					// For microsecond precision, nanoseconds should show microsecond info
					microsPart := (nanosec / 1000) * 1000
					assert.NotZero(t, microsPart, "Microsecond precision should be preserved")
				case "nanosecond":
					// Any nanosecond value is valid for nanosecond precision
				}
			}
		})
	}
}

func TestTimeRangeValidator_ValidateTimeRange(t *testing.T) {
	validator := NewTimeRangeValidator()
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		startTime *time.Time
		endTime   *time.Time
		wantErr   bool
		errorType string
	}{
		{
			name:      "Valid past range",
			startTime: &past,
			endTime:   &now,
			wantErr:   false,
		},
		{
			name:      "Start time in future",
			startTime: &future,
			endTime:   &now,
			wantErr:   true,
			errorType: "future_start",
		},
		{
			name:      "End time in future",
			startTime: &past,
			endTime:   &future,
			wantErr:   true,
			errorType: "future_end",
		},
		{
			name:      "Both nil",
			startTime: nil,
			endTime:   nil,
			wantErr:   false,
		},
		{
			name:      "Only start time",
			startTime: &past,
			endTime:   nil,
			wantErr:   false,
		},
		{
			name:      "Only end time",
			startTime: nil,
			endTime:   &now,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateTimeRange(tt.startTime, tt.endTime)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTimeRangeValidator_TimezoneNormalization(t *testing.T) {
	validator := NewTimeRangeValidator()

	tests := []struct {
		name        string
		timeStr     string
		expectedUTC string
	}{
		{
			name:        "UTC timezone",
			timeStr:     "2024-01-01T10:30:45Z",
			expectedUTC: "2024-01-01T10:30:45Z",
		},
		{
			name:        "PST timezone (-08:00)",
			timeStr:     "2024-01-01T10:30:45-08:00",
			expectedUTC: "2024-01-01T18:30:45Z",
		},
		{
			name:        "CEST timezone (+02:00)",
			timeStr:     "2024-01-01T10:30:45+02:00",
			expectedUTC: "2024-01-01T08:30:45Z",
		},
		{
			name:        "Millisecond with timezone",
			timeStr:     "2024-01-01T10:30:45.123-05:00",
			expectedUTC: "2024-01-01T15:30:45.123Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedTime, err := validator.parseTimeString(tt.timeStr)
			require.NoError(t, err)

			expectedTime, err := time.Parse(time.RFC3339Nano, tt.expectedUTC)
			require.NoError(t, err)

			assert.Equal(t, expectedTime.UTC(), parsedTime.UTC(),
				"Timezone should be normalized to UTC correctly")
		})
	}
}

func TestTimeRangeValidator_EdgeCases(t *testing.T) {
	validator := NewTimeRangeValidator()

	t.Run("Sub-millisecond range", func(t *testing.T) {
		start := "2024-01-01T10:30:45.123456789Z"
		end := "2024-01-01T10:30:45.123456999Z"

		startTime, endTime, err := validator.ValidateAndParseTimeRange(start, end)
		require.NoError(t, err)

		timeSpan := endTime.Sub(*startTime)
		assert.True(t, timeSpan > 0, "Should handle sub-millisecond ranges")
		assert.True(t, timeSpan < time.Millisecond, "Should be sub-millisecond range")
	})

	t.Run("Identical timestamps", func(t *testing.T) {
		timestamp := "2024-01-01T10:30:45.123456789Z"

		startTime, endTime, err := validator.ValidateAndParseTimeRange(timestamp, timestamp)
		require.NoError(t, err)

		assert.Equal(t, *startTime, *endTime, "Identical timestamps should be valid")
	})

	t.Run("Maximum time span", func(t *testing.T) {
		validator.SetMaxTimeSpan(1 * time.Hour) // Set shorter limit for testing

		start := "2024-01-01T10:00:00Z"
		end := "2024-01-01T11:30:00Z" // 1.5 hours

		_, _, err := validator.ValidateAndParseTimeRange(start, end)
		assert.Error(t, err, "Should reject time span exceeding maximum")

		// Reset to default
		validator.SetMaxTimeSpan(24 * time.Hour)
	})
}

func TestTimeRangeValidator_PerformancePrecision(t *testing.T) {
	validator := NewTimeRangeValidator()

	// Test parsing performance with different precisions
	timeFormats := []struct {
		name    string
		format  string
		timeStr string
	}{
		{"second", "second", "2024-01-01T10:30:45Z"},
		{"millisecond", "millisecond", "2024-01-01T10:30:45.123Z"},
		{"microsecond", "microsecond", "2024-01-01T10:30:45.123456Z"},
		{"nanosecond", "nanosecond", "2024-01-01T10:30:45.123456789Z"},
	}

	for _, tf := range timeFormats {
		t.Run(fmt.Sprintf("performance_%s", tf.name), func(t *testing.T) {
			iterations := 1000
			start := time.Now()

			for i := 0; i < iterations; i++ {
				_, err := validator.parseTimeString(tf.timeStr)
				require.NoError(t, err)
			}

			duration := time.Since(start)
			avgDuration := duration / time.Duration(iterations)

			t.Logf("%s precision parsing average: %v", tf.name, avgDuration)

			// Performance should be reasonable (< 100μs per parse on average)
			assert.Less(t, avgDuration.Microseconds(), int64(100),
				"Time parsing should be performant")
		})
	}
}

// Benchmark tests for time parsing performance
func BenchmarkTimeRangeValidator_ParseTimeString(b *testing.B) {
	validator := NewTimeRangeValidator()

	benchmarks := []struct {
		name    string
		timeStr string
	}{
		{"RFC3339", "2024-01-01T10:30:45Z"},
		{"Millisecond", "2024-01-01T10:30:45.123Z"},
		{"Microsecond", "2024-01-01T10:30:45.123456Z"},
		{"Nanosecond", "2024-01-01T10:30:45.123456789Z"},
		{"Timezone", "2024-01-01T10:30:45.123-07:00"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := validator.parseTimeString(bm.timeStr)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTimeRangeValidator_ValidateAndParseTimeRange(b *testing.B) {
	validator := NewTimeRangeValidator()

	b.Run("FullRange", func(b *testing.B) {
		startTime := "2024-01-01T10:30:45.123456Z"
		endTime := "2024-01-01T11:30:45.654321Z"

		for i := 0; i < b.N; i++ {
			_, _, err := validator.ValidateAndParseTimeRange(startTime, endTime)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}