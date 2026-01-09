package search

// ContentSearchType defines different search modes
type ContentSearchType string

const (
	ContentSearchExact        ContentSearchType = "exact"
	ContentSearchCaseInsensitive ContentSearchType = "icase"
	ContentSearchRegex        ContentSearchType = "regex"
	ContentSearchWildcard     ContentSearchType = "wildcard"
	ContentSearchPhrase       ContentSearchType = "phrase"
	ContentSearchProximity    ContentSearchType = "proximity"
	ContentSearchBoolean      ContentSearchType = "boolean"
)

// ContentSearchFilter represents a single content search condition
type ContentSearchFilter struct {
	Type              ContentSearchType `json:"type"`
	Pattern           string            `json:"pattern"`
	CaseInsensitive   bool              `json:"case_insensitive,omitempty"`
	BooleanOperator   string            `json:"boolean_operator,omitempty"` // AND, OR, NOT
	ProximityDistance int               `json:"proximity_distance,omitempty"` // For proximity searches
	FieldTarget       string            `json:"field_target,omitempty"`     // content, severity, etc.
	Weight            float64           `json:"weight,omitempty"`           // For relevance scoring
}

// ContentSearchExpression represents a complete search expression
type ContentSearchExpression struct {
	Filters          []ContentSearchFilter `json:"filters"`
	GlobalOperator   string                `json:"global_operator,omitempty"`   // Default operator for combining filters
	HighlightEnabled bool                  `json:"highlight_enabled,omitempty"`
	MaxSnippetLength int                   `json:"max_snippet_length,omitempty"`
	RelevanceScoring bool                  `json:"relevance_scoring,omitempty"`
}