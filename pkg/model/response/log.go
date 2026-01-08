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
}

// LogQueryResponse represents a log query response
type LogQueryResponse struct {
	Logs       []LogEntry `json:"logs"`
	TotalCount int        `json:"total_count"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	HasMore    bool       `json:"has_more"`
}