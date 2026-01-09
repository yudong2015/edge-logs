package query

import (
	"fmt"
	"regexp"
	"time"

	"k8s.io/klog/v2"
)

// TimeRangeValidator provides comprehensive time range validation and parsing with millisecond precision
type TimeRangeValidator struct {
	maxTimeSpan  time.Duration
	timeFormats  []string
	iso8601Regex *regexp.Regexp
}

// NewTimeRangeValidator creates a new time range validator with comprehensive format support
func NewTimeRangeValidator() *TimeRangeValidator {
	return &TimeRangeValidator{
		maxTimeSpan: 24 * time.Hour, // Prevent expensive queries
		timeFormats: []string{
			time.RFC3339,                      // 2006-01-02T15:04:05Z07:00
			"2006-01-02T15:04:05.000Z07:00",   // With milliseconds
			"2006-01-02T15:04:05.000000Z07:00", // With microseconds
			"2006-01-02T15:04:05.000000000Z07:00", // With nanoseconds
			"2006-01-02T15:04:05.000Z",        // With milliseconds UTC
			"2006-01-02T15:04:05.000000Z",     // With microseconds UTC
			"2006-01-02T15:04:05.000000000Z",  // With nanoseconds UTC
			"2006-01-02T15:04:05",             // Local time (converted to UTC)
			"2006-01-02 15:04:05",             // SQL format
		},
		iso8601Regex: regexp.MustCompile(
			`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d{1,9})?(Z|[+-]\d{2}:\d{2})?$`,
		),
	}
}

// ValidateAndParseTimeRange validates and normalizes time range inputs with millisecond precision
func (v *TimeRangeValidator) ValidateAndParseTimeRange(startStr, endStr string) (*time.Time, *time.Time, error) {
	var startTime, endTime *time.Time
	var err error

	// Parse start time if provided
	if startStr != "" {
		startTime, err = v.parseTimeString(startStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid start_time format '%s': %w", startStr, err)
		}
	}

	// Parse end time if provided
	if endStr != "" {
		endTime, err = v.parseTimeString(endStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid end_time format '%s': %w", endStr, err)
		}
	}

	// Validate time range logic
	if err := v.validateTimeRange(startTime, endTime); err != nil {
		return nil, nil, err
	}

	return startTime, endTime, nil
}

// parseTimeString attempts to parse time string in multiple formats with millisecond precision
func (v *TimeRangeValidator) parseTimeString(timeStr string) (*time.Time, error) {
	// Validate format using regex first
	if !v.iso8601Regex.MatchString(timeStr) {
		return nil, fmt.Errorf("time format must be ISO 8601 compliant (RFC3339 or similar)")
	}

	// Try parsing with multiple formats
	for _, format := range v.timeFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			// Normalize to UTC for consistent storage/querying
			utcTime := t.UTC()
			return &utcTime, nil
		}
	}

	// If standard parsing fails, try parsing with custom millisecond handling
	if t, err := v.parseWithMillisecondHandling(timeStr); err == nil {
		utcTime := t.UTC()
		return &utcTime, nil
	}

	return nil, fmt.Errorf("unsupported time format, supported formats: RFC3339, ISO 8601 with optional milliseconds")
}

// parseWithMillisecondHandling handles edge cases in millisecond parsing
func (v *TimeRangeValidator) parseWithMillisecondHandling(timeStr string) (time.Time, error) {
	// Handle different millisecond precision formats
	formats := []string{
		"2006-01-02T15:04:05.999999999Z", // Nanoseconds
		"2006-01-02T15:04:05.999999Z",    // Microseconds
		"2006-01-02T15:04:05.999Z",       // Milliseconds
		"2006-01-02T15:04:05.99Z",        // Centiseconds
		"2006-01-02T15:04:05.9Z",         // Deciseconds
		// Timezone variants
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999-07:00",
		"2006-01-02T15:04:05.999-07:00",
		"2006-01-02T15:04:05.99-07:00",
		"2006-01-02T15:04:05.9-07:00",
		// Space separator variants
		"2006-01-02 15:04:05.999999999Z",
		"2006-01-02 15:04:05.999999Z",
		"2006-01-02 15:04:05.999Z",
		"2006-01-02 15:04:05.99Z",
		"2006-01-02 15:04:05.9Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time with millisecond precision")
}

// validateTimeRange ensures time range is logical and within limits
func (v *TimeRangeValidator) validateTimeRange(startTime, endTime *time.Time) error {
	now := time.Now().UTC()

	// Check for future times
	if startTime != nil && startTime.After(now) {
		return &TimeValidationError{
			Field:  "start_time",
			Value:  startTime.Format(time.RFC3339Nano),
			Reason: "cannot be in the future",
		}
	}
	if endTime != nil && endTime.After(now) {
		return &TimeValidationError{
			Field:  "end_time",
			Value:  endTime.Format(time.RFC3339Nano),
			Reason: "cannot be in the future",
		}
	}

	// Check time order
	if startTime != nil && endTime != nil {
		if startTime.After(*endTime) {
			return &TimeRangeError{
				StartTime: startTime,
				EndTime:   endTime,
				Issue:     "start_time must be before or equal to end_time",
			}
		}

		// Check maximum time span to prevent expensive queries
		timeSpan := endTime.Sub(*startTime)
		if timeSpan > v.maxTimeSpan {
			return &TimeRangeError{
				StartTime: startTime,
				EndTime:   endTime,
				Issue:     fmt.Sprintf("time range span (%v) exceeds maximum allowed span (%v)", timeSpan, v.maxTimeSpan),
			}
		}

		// Warn for very small time ranges (may indicate precision issues)
		if timeSpan < time.Millisecond {
			// Allow but log warning for sub-millisecond queries
			klog.V(2).InfoS("Sub-millisecond time range query",
				"start_time", startTime.Format(time.RFC3339Nano),
				"end_time", endTime.Format(time.RFC3339Nano),
				"span_ns", timeSpan.Nanoseconds())
		}
	}

	return nil
}

// ValidateTimeFormat validates a single time string format without parsing
func (v *TimeRangeValidator) ValidateTimeFormat(timeStr string) error {
	if timeStr == "" {
		return nil
	}

	if !v.iso8601Regex.MatchString(timeStr) {
		return &TimeValidationError{
			Field:  "time_format",
			Value:  timeStr,
			Reason: "must be ISO 8601 compliant format",
		}
	}

	return nil
}

// GetSupportedFormats returns a list of supported time formats for documentation
func (v *TimeRangeValidator) GetSupportedFormats() []string {
	return []string{
		"RFC3339: 2006-01-02T15:04:05Z",
		"With milliseconds: 2006-01-02T15:04:05.123Z",
		"With microseconds: 2006-01-02T15:04:05.123456Z",
		"With nanoseconds: 2006-01-02T15:04:05.123456789Z",
		"With timezone: 2006-01-02T15:04:05.123-07:00",
		"SQL format: 2006-01-02 15:04:05.123",
	}
}

// GetMaxTimeSpan returns the maximum allowed time span for queries
func (v *TimeRangeValidator) GetMaxTimeSpan() time.Duration {
	return v.maxTimeSpan
}

// SetMaxTimeSpan sets the maximum allowed time span for queries
func (v *TimeRangeValidator) SetMaxTimeSpan(duration time.Duration) {
	v.maxTimeSpan = duration
}