package request

import (
	"fmt"
	"time"
)

// K8sFilterType defines different matching patterns for K8s filtering
type K8sFilterType string

const (
	K8sFilterExact     K8sFilterType = "exact"
	K8sFilterPrefix    K8sFilterType = "prefix"
	K8sFilterSuffix    K8sFilterType = "suffix"
	K8sFilterContains  K8sFilterType = "contains"
	K8sFilterRegex     K8sFilterType = "regex"
	K8sFilterWildcard  K8sFilterType = "wildcard"
)

// K8sFilter represents a single K8s filtering condition
type K8sFilter struct {
	Type            K8sFilterType `json:"type"`
	Pattern         string        `json:"pattern"`
	Field           string        `json:"field"` // "namespace" or "pod"
	CaseInsensitive bool          `json:"case_insensitive,omitempty"`
}

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

	// K8s metadata filtering - enhanced with multiple pattern support
	Namespace  string   `json:"namespace,omitempty"`   // Legacy single namespace for backward compatibility
	Namespaces []string `json:"namespaces,omitempty"` // Multiple namespaces support
	PodName    string   `json:"pod_name,omitempty"`   // Legacy single pod name for backward compatibility
	PodNames   []string `json:"pod_names,omitempty"`  // Multiple pod names with pattern matching
	NodeName   string   `json:"node_name,omitempty"`

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

	// Advanced content search parameters
	ContentSearch     string `json:"content_search,omitempty"`      // Advanced content search query
	ContentHighlight  *bool  `json:"content_highlight,omitempty"`   // Enable search highlighting
	ContentRelevance  *bool  `json:"content_relevance,omitempty"`   // Enable relevance scoring
	ContentOperator   string `json:"content_operator,omitempty"`    // Default boolean operator (AND/OR)

	// K8s metadata enrichment
	EnrichMetadata *bool `json:"enrich_metadata,omitempty"` // Enable K8s metadata enrichment

	// Internal fields for parsed filters (not exposed in JSON)
	K8sFilters            []K8sFilter                    `json:"-"` // Parsed K8s filter conditions
	ParsedContentSearch   *ParsedContentSearchExpression `json:"-"` // Parsed content search expression
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

// ParsedContentSearchExpression represents a parsed content search expression
// This is a lightweight version that can be imported without circular dependencies
type ParsedContentSearchExpression struct {
	Filters          []ContentSearchFilter `json:"filters"`
	GlobalOperator   string                `json:"global_operator,omitempty"`
	HighlightEnabled bool                  `json:"highlight_enabled,omitempty"`
	MaxSnippetLength int                   `json:"max_snippet_length,omitempty"`
	RelevanceScoring bool                  `json:"relevance_scoring,omitempty"`
}

// ContentSearchFilter represents a single content search condition
type ContentSearchFilter struct {
	Type              string  `json:"type"`
	Pattern           string  `json:"pattern"`
	CaseInsensitive   bool    `json:"case_insensitive,omitempty"`
	BooleanOperator   string  `json:"boolean_operator,omitempty"`
	ProximityDistance int     `json:"proximity_distance,omitempty"`
	FieldTarget       string  `json:"field_target,omitempty"`
	Weight            float64 `json:"weight,omitempty"`
}