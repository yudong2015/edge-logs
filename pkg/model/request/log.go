package request

import (
	"fmt"
	"time"
)

// LogQueryRequest represents a log query request
type LogQueryRequest struct {
	// Data isolation
	Dataset string `json:"dataset"`

	// Time range filtering
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`

	// Content filtering
	Filter   string `json:"filter,omitempty"`   // Full-text search
	Severity string `json:"severity,omitempty"` // Log severity level

	// K8s metadata filtering
	Namespace string `json:"namespace,omitempty"`
	PodName   string `json:"pod_name,omitempty"`
	NodeName  string `json:"node_name,omitempty"`

	// Host filtering
	HostIP   string `json:"host_ip,omitempty"`
	HostName string `json:"host_name,omitempty"`

	// Container filtering
	ContainerName string `json:"container_name,omitempty"`

	// Tag filtering
	Tags map[string]string `json:"tags,omitempty"`

	// Pagination
	Page     int `json:"page,omitempty"`
	PageSize int `json:"page_size,omitempty"`

	// Result ordering
	OrderBy   string `json:"order_by,omitempty"` // timestamp, severity
	Direction string `json:"direction,omitempty"` // asc, desc
}

// Validate validates the log query request
func (r *LogQueryRequest) Validate() error {
	// Dataset is required for data isolation
	if r.Dataset == "" {
		return fmt.Errorf("dataset is required")
	}

	// Validate pagination
	if r.Page < 0 {
		return fmt.Errorf("page must be non-negative")
	}
	if r.PageSize < 0 {
		return fmt.Errorf("page_size must be non-negative")
	}
	if r.PageSize > 10000 {
		return fmt.Errorf("page_size cannot exceed 10000")
	}

	// Set default page size
	if r.PageSize == 0 {
		r.PageSize = 100
	}

	// Validate time range
	if r.StartTime != nil && r.EndTime != nil && r.StartTime.After(*r.EndTime) {
		return fmt.Errorf("start_time must be before end_time")
	}

	// Validate ordering
	if r.OrderBy != "" && r.OrderBy != "timestamp" && r.OrderBy != "severity" {
		return fmt.Errorf("order_by must be 'timestamp' or 'severity'")
	}
	if r.Direction != "" && r.Direction != "asc" && r.Direction != "desc" {
		return fmt.Errorf("direction must be 'asc' or 'desc'")
	}

	// Set defaults for ordering
	if r.OrderBy == "" {
		r.OrderBy = "timestamp"
	}
	if r.Direction == "" {
		r.Direction = "desc"
	}

	return nil
}