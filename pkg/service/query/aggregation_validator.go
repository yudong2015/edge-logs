package query

import (
	"fmt"
	"strings"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// AggregationDimensionValidator validates aggregation dimensions and functions
type AggregationDimensionValidator struct {
	maxDimensions     int
	maxFunctions      int
	maxComplexity     float64
	cardinalityLimits map[request.AggregationDimensionType]int
}

// NewAggregationDimensionValidator creates a new aggregation validator
func NewAggregationDimensionValidator() *AggregationDimensionValidator {
	return &AggregationDimensionValidator{
		maxDimensions: 5,
		maxFunctions:  8,
		maxComplexity: 100.0,
		cardinalityLimits: map[request.AggregationDimensionType]int{
			request.DimensionSeverity:      10,
			request.DimensionNamespace:     1000,
			request.DimensionPodName:       10000,
			request.DimensionNodeName:      500,
			request.DimensionHostName:      1000,
			request.DimensionContainerName: 5000,
			request.DimensionDataset:       100,
		},
	}
}

// ValidateAggregationRequest validates comprehensive aggregation request
func (v *AggregationDimensionValidator) ValidateAggregationRequest(req *request.AggregationRequest) error {
	if req == nil {
		return fmt.Errorf("aggregation request cannot be nil")
	}

	// Validate dataset
	if req.Dataset == "" {
		return fmt.Errorf("dataset is required")
	}

	// Validate dimension count
	if len(req.Dimensions) == 0 {
		return fmt.Errorf("at least one dimension must be specified")
	}

	if len(req.Dimensions) > v.maxDimensions {
		return fmt.Errorf("too many dimensions (%d), maximum allowed: %d",
			len(req.Dimensions), v.maxDimensions)
	}

	// Validate function count
	if len(req.Functions) == 0 {
		return fmt.Errorf("at least one aggregation function must be specified")
	}

	if len(req.Functions) > v.maxFunctions {
		return fmt.Errorf("too many functions (%d), maximum allowed: %d",
			len(req.Functions), v.maxFunctions)
	}

	// Validate dimensions
	for i, dim := range req.Dimensions {
		if err := v.validateDimension(dim); err != nil {
			return fmt.Errorf("dimension %d validation failed: %w", i, err)
		}
	}

	// Validate functions
	for i, fn := range req.Functions {
		if err := v.validateFunction(fn); err != nil {
			return fmt.Errorf("function %d validation failed: %w", i, err)
		}
	}

	// Validate time dimension consistency
	if err := v.validateTimeDimensions(req); err != nil {
		return fmt.Errorf("time dimension validation failed: %w", err)
	}

	// Validate complexity
	complexity := v.calculateRequestComplexity(req)
	if complexity > v.maxComplexity {
		return fmt.Errorf("aggregation complexity (%.1f) exceeds maximum (%.1f)",
			complexity, v.maxComplexity)
	}

	return nil
}

// validateDimension validates a single aggregation dimension
func (v *AggregationDimensionValidator) validateDimension(dim request.AggregationDimension) error {
	// Validate dimension type
	switch dim.Type {
	case request.DimensionSeverity, request.DimensionNamespace, request.DimensionPodName,
		request.DimensionNodeName, request.DimensionHostName, request.DimensionContainerName, request.DimensionDataset:
		// Standard dimensions - no additional validation needed
	case request.DimensionTimestamp:
		// Time dimension requires bucket interval
		if dim.TimeBucket == "" {
			return fmt.Errorf("timestamp dimension requires time_bucket specification")
		}
		if err := v.validateTimeBucket(dim.TimeBucket); err != nil {
			return fmt.Errorf("time bucket validation failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported dimension type: %s", dim.Type)
	}

	// Validate alias
	if dim.Alias != "" {
		if !isValidIdentifier(dim.Alias) {
			return fmt.Errorf("invalid dimension alias: %s", dim.Alias)
		}
	}

	// Validate sort order
	if dim.SortOrder != "" && dim.SortOrder != "ASC" && dim.SortOrder != "DESC" {
		return fmt.Errorf("invalid sort order: %s (must be ASC or DESC)", dim.SortOrder)
	}

	// Validate limit
	if dim.Limit < 0 || dim.Limit > 10000 {
		return fmt.Errorf("dimension limit must be between 0 and 10000, got: %d", dim.Limit)
	}

	return nil
}

// validateFunction validates a single aggregation function
func (v *AggregationDimensionValidator) validateFunction(fn request.AggregationFunction) error {
	// Validate function type
	switch fn.Type {
	case request.FunctionCount:
		// Count function doesn't require field specification
	case request.FunctionSum, request.FunctionAvg, request.FunctionMin, request.FunctionMax:
		// Numeric functions require field specification
		if fn.Field == "" {
			return fmt.Errorf("%s function requires field specification", fn.Type)
		}
		if !isNumericField(fn.Field) {
			return fmt.Errorf("%s function requires numeric field, got: %s", fn.Type, fn.Field)
		}
	case request.FunctionDistinctCount:
		// Distinct count requires field specification
		if fn.Field == "" {
			return fmt.Errorf("distinct_count function requires field specification")
		}
	default:
		return fmt.Errorf("unsupported function type: %s", fn.Type)
	}

	// Validate alias
	if fn.Alias != "" {
		if !isValidIdentifier(fn.Alias) {
			return fmt.Errorf("invalid function alias: %s", fn.Alias)
		}
	}

	return nil
}

// validateTimeBucket validates time bucket configuration
func (v *AggregationDimensionValidator) validateTimeBucket(bucket request.TimeBucketInterval) error {
	switch bucket {
	case request.IntervalMinute, request.Interval5Minutes, request.Interval15Minutes,
		request.IntervalHour, request.Interval6Hours, request.Interval12Hours,
		request.IntervalDay, request.IntervalWeek:
		return nil
	default:
		return fmt.Errorf("unsupported time bucket interval: %s", bucket)
	}
}

// validateTimeDimensions ensures consistent time handling
func (v *AggregationDimensionValidator) validateTimeDimensions(req *request.AggregationRequest) error {
	hasTimeDimension := false
	for _, dim := range req.Dimensions {
		if dim.Type == request.DimensionTimestamp {
			hasTimeDimension = true
			break
		}
	}

	// Validate time range for time-based aggregations
	if hasTimeDimension && (req.StartTime == nil || req.EndTime == nil) {
		return fmt.Errorf("time dimensions require start_time and end_time specification")
	}

	return nil
}

// calculateRequestComplexity provides aggregation complexity scoring
func (v *AggregationDimensionValidator) calculateRequestComplexity(req *request.AggregationRequest) float64 {
	complexity := 0.0

	// Dimension complexity
	for _, dim := range req.Dimensions {
		switch dim.Type {
		case request.DimensionSeverity, request.DimensionDataset:
			complexity += 1.0
		case request.DimensionNamespace, request.DimensionHostName:
			complexity += 2.0
		case request.DimensionPodName, request.DimensionContainerName:
			complexity += 3.0
		case request.DimensionTimestamp:
			complexity += 1.5
		case request.DimensionNodeName:
			complexity += 2.0
		}
	}

	// Function complexity
	for _, fn := range req.Functions {
		switch fn.Type {
		case request.FunctionCount:
			complexity += 0.5
		case request.FunctionSum, request.FunctionAvg, request.FunctionMin, request.FunctionMax:
			complexity += 1.0
		case request.FunctionDistinctCount:
			complexity += 2.0
		}
	}

	// Multi-dimensional complexity multiplier
	if len(req.Dimensions) > 1 {
		complexity *= 1.5 + float64(len(req.Dimensions)-1)*0.3
	}

	return complexity
}

// isValidIdentifier validates SQL identifier naming
func isValidIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// Must start with letter or underscore
	firstChar := name[0]
	if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z') || firstChar == '_') {
		return false
	}

	// Check remaining characters
	for _, r := range name[1:] {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	return len(name) <= 64
}

// isNumericField determines if a field contains numeric data
func isNumericField(field string) bool {
	numericFields := map[string]bool{
		"message_length":  true,
		"processing_time": true,
		"timestamp":       true,
		"error_code":      true,
	}
	return numericFields[field]
}

// BuildAggregationCacheKey generates cache key for aggregation request
func BuildAggregationCacheKey(req *request.AggregationRequest) string {
	var keyParts []string

	keyParts = append(keyParts, "dataset:"+req.Dataset)

	// Time range
	if req.StartTime != nil && req.EndTime != nil {
		keyParts = append(keyParts, fmt.Sprintf("time:%d-%d", req.StartTime.Unix(), req.EndTime.Unix()))
	}

	// Dimensions
	var dimParts []string
	for _, dim := range req.Dimensions {
		dimKey := string(dim.Type)
		if dim.TimeBucket != "" {
			dimKey += ":" + string(dim.TimeBucket)
		}
		dimParts = append(dimParts, dimKey)
	}
	if len(dimParts) > 0 {
		keyParts = append(keyParts, "dims:"+strings.Join(dimParts, ","))
	}

	// Functions
	var funcParts []string
	for _, fn := range req.Functions {
		funcKey := string(fn.Type)
		if fn.Field != "" {
			funcKey += ":" + fn.Field
		}
		funcParts = append(funcParts, funcKey)
	}
	if len(funcParts) > 0 {
		keyParts = append(keyParts, "funcs:"+strings.Join(funcParts, ","))
	}

	return strings.Join(keyParts, "|")
}
