package response

import "time"

// AggregationResult represents a single aggregation result row
type AggregationResult struct {
	Dimensions map[string]interface{} `json:"dimensions"` // Dimension key-value pairs
	Metrics    map[string]interface{} `json:"metrics"`    // Aggregation function results
}

// AggregationMetadata provides execution metadata for aggregation queries
type AggregationMetadata struct {
	QueryDurationMs      int64     `json:"query_duration_ms"`
	DimensionCount       int       `json:"dimension_count"`
	FunctionCount        int       `json:"function_count"`
	ResultSetSize        int       `json:"result_set_size"`
	GeneratedAt          time.Time `json:"generated_at"`
}

// AggregationResponse represents the complete aggregation query response
type AggregationResponse struct {
	Dataset  string                `json:"dataset"`
	Results  []AggregationResult   `json:"results"`
	Metadata *AggregationMetadata   `json:"metadata,omitempty"`
	Query    *AggregationQueryInfo `json:"query,omitempty"`
}

// AggregationQueryInfo provides information about the aggregation query
type AggregationQueryInfo struct {
	Dimensions []string `json:"dimensions"` // Dimension names
	Functions  []string `json:"functions"`  // Function names
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
}
