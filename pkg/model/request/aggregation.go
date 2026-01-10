package request

import (
	"time"
)

// AggregationDimensionType defines supported aggregation dimensions
type AggregationDimensionType string

const (
	DimensionSeverity      AggregationDimensionType = "severity"
	DimensionNamespace     AggregationDimensionType = "namespace"
	DimensionPodName       AggregationDimensionType = "pod_name"
	DimensionNodeName      AggregationDimensionType = "node_name"
	DimensionHostName      AggregationDimensionType = "host_name"
	DimensionContainerName AggregationDimensionType = "container_name"
	DimensionTimestamp     AggregationDimensionType = "timestamp"
	DimensionDataset       AggregationDimensionType = "dataset"
)

// AggregationFunctionType defines supported aggregation functions
type AggregationFunctionType string

const (
	FunctionCount         AggregationFunctionType = "count"
	FunctionSum           AggregationFunctionType = "sum"
	FunctionAvg           AggregationFunctionType = "avg"
	FunctionMin           AggregationFunctionType = "min"
	FunctionMax           AggregationFunctionType = "max"
	FunctionDistinctCount AggregationFunctionType = "distinct_count"
)

// TimeBucketInterval defines time bucketing intervals
type TimeBucketInterval string

const (
	IntervalMinute     TimeBucketInterval = "1m"
	Interval5Minutes   TimeBucketInterval = "5m"
	Interval15Minutes  TimeBucketInterval = "15m"
	IntervalHour       TimeBucketInterval = "1h"
	Interval6Hours     TimeBucketInterval = "6h"
	Interval12Hours    TimeBucketInterval = "12h"
	IntervalDay        TimeBucketInterval = "1d"
	IntervalWeek       TimeBucketInterval = "1w"
)

// AggregationDimension represents a single aggregation dimension
type AggregationDimension struct {
	Type       AggregationDimensionType `json:"type"`
	Field      string                   `json:"field,omitempty"`       // For custom fields
	TimeBucket TimeBucketInterval       `json:"time_bucket,omitempty"` // For timestamp dimensions
	Alias      string                   `json:"alias,omitempty"`       // Output field alias
	SortOrder  string                   `json:"sort_order,omitempty"`  // ASC, DESC
	Limit      int                      `json:"limit,omitempty"`       // Limit results for this dimension
}

// AggregationFunction represents a single aggregation function
type AggregationFunction struct {
	Type  AggregationFunctionType `json:"type"`
	Field string                 `json:"field,omitempty"` // Field to aggregate
	Alias string                 `json:"alias,omitempty"` // Output field alias
}

// AggregationRequest represents a complete aggregation query
type AggregationRequest struct {
	// Inherited filtering from Epic 2
	Dataset       string   `json:"dataset"`
	StartTime     *time.Time `json:"start_time,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Namespaces    []string `json:"namespaces,omitempty"`
	PodNames      []string `json:"pod_names,omitempty"`
	Severity      string   `json:"severity,omitempty"`
	ContentSearch string   `json:"content_search,omitempty"`

	// Aggregation-specific fields
	Dimensions []AggregationDimension `json:"dimensions"`
	Functions  []AggregationFunction  `json:"functions"`
	OrderBy    []string               `json:"order_by,omitempty"`
	Limit      int                    `json:"limit,omitempty"`
	Offset     int                    `json:"offset,omitempty"`

	// Output formatting
	OutputFormat string `json:"output_format,omitempty"` // json, csv
	TimeZone     string `json:"time_zone,omitempty"`     // Timezone for timestamp formatting
	Precision    int    `json:"precision,omitempty"`     // Decimal precision for numeric results
}
