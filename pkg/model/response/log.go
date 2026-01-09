package response

import "time"

// LogEntry represents a single log entry
type LogEntry struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Message   string            `json:"message"`
	Level     string            `json:"level"`
	Namespace string            `json:"namespace"`
	Pod       string            `json:"pod"`
	Container string            `json:"container"`
	Labels    map[string]string `json:"labels"`

	// Content search enhancement fields
	HighlightedContent     []string `json:"highlighted_content,omitempty"`     // Search result highlighting
	SearchRelevanceScore   float64  `json:"search_relevance_score,omitempty"`  // Relevance scoring
	SearchMatchSummary     string   `json:"search_match_summary,omitempty"`    // Search match summary
}

// LogQueryResponse represents a log query response with dataset metadata
type LogQueryResponse struct {
	Logs       []LogEntry       `json:"logs"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	HasMore    bool             `json:"has_more"`
	Dataset    string           `json:"dataset,omitempty"`           // Dataset name from request
	Query      *QuerySummary    `json:"query,omitempty"`             // Query parameters summary
	Metadata   *DatasetMetadata `json:"metadata,omitempty"`          // Dataset metadata
	SearchMeta *SearchMetadata  `json:"search_metadata,omitempty"`   // Content search metadata
}

// QuerySummary provides a sanitized summary of the query parameters
type QuerySummary struct {
	StartTime     *time.Time        `json:"start_time,omitempty"`
	EndTime       *time.Time        `json:"end_time,omitempty"`
	Filter        string            `json:"filter,omitempty"`
	Namespace     string            `json:"namespace,omitempty"`
	Filters       map[string]string `json:"filters,omitempty"`
}

// DatasetMetadata contains metadata about the queried dataset
type DatasetMetadata struct {
	Name            string     `json:"name"`
	TotalLogs       int64      `json:"total_logs"`
	DateRange       *DateRange `json:"date_range,omitempty"`
	LastUpdated     *time.Time `json:"last_updated,omitempty"`
	PartitionCount  int        `json:"partition_count"`
	DataSizeBytes   int64      `json:"data_size_bytes"`
	Health          string     `json:"health,omitempty"`
}

// DateRange represents the earliest and latest timestamps in a dataset
type DateRange struct {
	Earliest time.Time `json:"earliest"`
	Latest   time.Time `json:"latest"`
}

// SearchMetadata contains metadata about content search results
type SearchMetadata struct {
	PatternsMatched    int     `json:"patterns_matched"`
	TotalHighlights    int     `json:"total_highlights"`
	SearchComplexity   float64 `json:"search_complexity"`
	QueryTimeMs        int64   `json:"query_time_ms"`
	HighlightEnabled   bool    `json:"highlight_enabled"`
	RelevanceScoring   bool    `json:"relevance_scoring"`
	SearchType         string  `json:"search_type"`         // exact, wildcard, regex, boolean, etc.
	OptimizationUsed   string  `json:"optimization_used"`   // tokenbf_v1, none, etc.
}