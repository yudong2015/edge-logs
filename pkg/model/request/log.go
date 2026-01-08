package request

import "time"

// LogQueryRequest represents a log query request
type LogQueryRequest struct {
	Query     string            `json:"query,omitempty"`
	StartTime *time.Time        `json:"start_time,omitempty"`
	EndTime   *time.Time        `json:"end_time,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
	Pod       string            `json:"pod,omitempty"`
	Container string            `json:"container,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Page      int               `json:"page,omitempty"`
	PageSize  int               `json:"page_size,omitempty"`
}

// Validate validates the log query request
func (r *LogQueryRequest) Validate() error {
	// TODO: Implement validation logic
	return nil
}