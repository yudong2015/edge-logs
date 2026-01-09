package query

import (
	"fmt"
	"time"
)

// TimeValidationError represents a time format validation error
type TimeValidationError struct {
	Field  string
	Value  string
	Reason string
}

func (e *TimeValidationError) Error() string {
	return fmt.Sprintf("time validation failed for %s='%s': %s", e.Field, e.Value, e.Reason)
}

// TimeRangeError represents a time range logic error
type TimeRangeError struct {
	StartTime *time.Time
	EndTime   *time.Time
	Issue     string
}

func (e *TimeRangeError) Error() string {
	return fmt.Sprintf("time range error: %s (start: %v, end: %v)",
		e.Issue,
		formatOptionalTime(e.StartTime),
		formatOptionalTime(e.EndTime))
}

// TimePrecisionError represents a time precision handling error
type TimePrecisionError struct {
	TimeValue string
	Precision string
	Issue     string
}

func (e *TimePrecisionError) Error() string {
	return fmt.Sprintf("time precision error: %s (value: %s, precision: %s)", e.Issue, e.TimeValue, e.Precision)
}

// TimeParsingError represents a time string parsing error
type TimeParsingError struct {
	TimeValue    string
	AttemptedFormat string
	UnderlyingError error
}

func (e *TimeParsingError) Error() string {
	if e.AttemptedFormat != "" {
		return fmt.Sprintf("time parsing error: failed to parse '%s' with format '%s': %v",
			e.TimeValue, e.AttemptedFormat, e.UnderlyingError)
	}
	return fmt.Sprintf("time parsing error: failed to parse '%s': %v", e.TimeValue, e.UnderlyingError)
}

func (e *TimeParsingError) Unwrap() error {
	return e.UnderlyingError
}

// Helper function to format optional time pointers
func formatOptionalTime(t *time.Time) string {
	if t == nil {
		return "<nil>"
	}
	return t.Format(time.RFC3339Nano)
}

// IsTimeValidationError checks if an error is a time validation error
func IsTimeValidationError(err error) bool {
	_, ok := err.(*TimeValidationError)
	return ok
}

// IsTimeRangeError checks if an error is a time range error
func IsTimeRangeError(err error) bool {
	_, ok := err.(*TimeRangeError)
	return ok
}

// IsTimePrecisionError checks if an error is a time precision error
func IsTimePrecisionError(err error) bool {
	_, ok := err.(*TimePrecisionError)
	return ok
}

// IsTimeParsingError checks if an error is a time parsing error
func IsTimeParsingError(err error) bool {
	_, ok := err.(*TimeParsingError)
	return ok
}

// NewTimeValidationError creates a new time validation error
func NewTimeValidationError(field, value, reason string) *TimeValidationError {
	return &TimeValidationError{
		Field:  field,
		Value:  value,
		Reason: reason,
	}
}

// NewTimeRangeError creates a new time range error
func NewTimeRangeError(startTime, endTime *time.Time, issue string) *TimeRangeError {
	return &TimeRangeError{
		StartTime: startTime,
		EndTime:   endTime,
		Issue:     issue,
	}
}

// NewTimePrecisionError creates a new time precision error
func NewTimePrecisionError(timeValue, precision, issue string) *TimePrecisionError {
	return &TimePrecisionError{
		TimeValue: timeValue,
		Precision: precision,
		Issue:     issue,
	}
}

// NewTimeParsingError creates a new time parsing error
func NewTimeParsingError(timeValue, attemptedFormat string, underlyingError error) *TimeParsingError {
	return &TimeParsingError{
		TimeValue:       timeValue,
		AttemptedFormat: attemptedFormat,
		UnderlyingError: underlyingError,
	}
}