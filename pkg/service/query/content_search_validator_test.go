package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentSearchValidator(t *testing.T) {
	validator := NewContentSearchValidator()
	require.NotNil(t, validator)

	tests := []struct {
		name        string
		query       string
		options     map[string]string
		expectError bool
		expectType  ContentSearchType
		description string
	}{
		{
			name:        "empty query",
			query:       "",
			options:     nil,
			expectError: false,
			description: "Empty query should return empty expression",
		},
		{
			name:        "simple exact search",
			query:       "error",
			options:     nil,
			expectError: false,
			expectType:  ContentSearchExact,
			description: "Simple word should be treated as exact search",
		},
		{
			name:        "case insensitive search",
			query:       "icase:ERROR",
			options:     nil,
			expectError: false,
			expectType:  ContentSearchCaseInsensitive,
			description: "icase: prefix should enable case insensitive search",
		},
		{
			name:        "wildcard search",
			query:       "error*",
			options:     nil,
			expectError: false,
			expectType:  ContentSearchWildcard,
			description: "Asterisk should enable wildcard search",
		},
		{
			name:        "regex search",
			query:       "regex:error\\s+(failed|timeout)",
			options:     nil,
			expectError: false,
			expectType:  ContentSearchRegex,
			description: "regex: prefix should enable regex search",
		},
		{
			name:        "phrase search",
			query:       "\"connection failed\"",
			options:     nil,
			expectError: false,
			expectType:  ContentSearchPhrase,
			description: "Quoted string should enable phrase search",
		},
		{
			name:        "boolean search",
			query:       "boolean:error AND failed",
			options:     nil,
			expectError: false,
			expectType:  ContentSearchExact,
			description: "Boolean search should parse into exact search filters",
		},
		{
			name:        "proximity search",
			query:       "proximity:5:database connection",
			options:     nil,
			expectError: false,
			expectType:  ContentSearchProximity,
			description: "Proximity search should set proximity distance",
		},
		{
			name:        "unsafe regex pattern",
			query:       "regex:.*.*",
			options:     nil,
			expectError: true,
			description: "Unsafe regex patterns should be rejected",
		},
		{
			name:        "query too long",
			query:       string(make([]byte, 1000)),
			options:     nil,
			expectError: true,
			description: "Overly long queries should be rejected",
		},
		{
			name:        "invalid proximity format",
			query:       "proximity:invalid:terms",
			options:     nil,
			expectError: true,
			description: "Invalid proximity format should be rejected",
		},
		{
			name:        "proximity distance too high",
			query:       "proximity:100:error timeout",
			options:     nil,
			expectError: true,
			description: "Proximity distance over limit should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expression, err := validator.ParseContentSearch(tt.query, tt.options)

			if tt.expectError {
				assert.Error(t, err, "Expected error for: %s", tt.description)
				return
			}

			assert.NoError(t, err, "Unexpected error for: %s", tt.description)
			assert.NotNil(t, expression, "Expression should not be nil for: %s", tt.description)

			if tt.expectType != "" && len(expression.Filters) > 0 {
				assert.Equal(t, tt.expectType, expression.Filters[0].Type, "Wrong filter type for: %s", tt.description)
			}
		})
	}
}

func TestContentSearchValidation(t *testing.T) {
	validator := NewContentSearchValidator()

	t.Run("validate single filter", func(t *testing.T) {
		filter := ContentSearchFilter{
			Type:    ContentSearchExact,
			Pattern: "error",
			Weight:  1.0,
		}

		err := validator.validateSingleFilter(filter)
		assert.NoError(t, err)
	})

	t.Run("validate empty pattern", func(t *testing.T) {
		filter := ContentSearchFilter{
			Type:    ContentSearchExact,
			Pattern: "",
			Weight:  1.0,
		}

		err := validator.validateSingleFilter(filter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pattern cannot be empty")
	})

	t.Run("validate invalid regex", func(t *testing.T) {
		filter := ContentSearchFilter{
			Type:    ContentSearchRegex,
			Pattern: "[invalid",
			Weight:  1.0,
		}

		err := validator.validateSingleFilter(filter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid regex pattern")
	})

	t.Run("validate invalid proximity distance", func(t *testing.T) {
		filter := ContentSearchFilter{
			Type:              ContentSearchProximity,
			Pattern:           "error timeout",
			ProximityDistance: 100,
			Weight:            1.0,
		}

		err := validator.validateSingleFilter(filter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "proximity distance must be between 1 and 50")
	})

	t.Run("validate invalid boolean operator", func(t *testing.T) {
		filter := ContentSearchFilter{
			Type:            ContentSearchExact,
			Pattern:         "error",
			BooleanOperator: "INVALID",
			Weight:          1.0,
		}

		err := validator.validateSingleFilter(filter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid boolean operator")
	})
}

func TestContentSearchComplexity(t *testing.T) {
	validator := NewContentSearchValidator()

	tests := []struct {
		name               string
		expression         *ContentSearchExpression
		expectedComplexity float64
		description        string
	}{
		{
			name: "simple exact search",
			expression: &ContentSearchExpression{
				Filters: []ContentSearchFilter{
					{Type: ContentSearchExact, Pattern: "error", Weight: 1.0},
				},
			},
			expectedComplexity: 1.0,
			description:        "Single exact search should have complexity 1.0",
		},
		{
			name: "case insensitive search",
			expression: &ContentSearchExpression{
				Filters: []ContentSearchFilter{
					{Type: ContentSearchCaseInsensitive, Pattern: "ERROR", Weight: 1.0},
				},
			},
			expectedComplexity: 1.5,
			description:        "Case insensitive search should have higher complexity",
		},
		{
			name: "wildcard search",
			expression: &ContentSearchExpression{
				Filters: []ContentSearchFilter{
					{Type: ContentSearchWildcard, Pattern: "error*", Weight: 1.0},
				},
			},
			expectedComplexity: 3.0,
			description:        "Wildcard search should have complexity 3.0",
		},
		{
			name: "regex search",
			expression: &ContentSearchExpression{
				Filters: []ContentSearchFilter{
					{Type: ContentSearchRegex, Pattern: "error\\s+failed", Weight: 1.0},
				},
			},
			expectedComplexity: 6.4, // 5.0 + 14/10 (pattern length bonus)
			description:        "Regex search should have high complexity including pattern length",
		},
		{
			name: "multiple filters",
			expression: &ContentSearchExpression{
				Filters: []ContentSearchFilter{
					{Type: ContentSearchExact, Pattern: "error", Weight: 1.0},
					{Type: ContentSearchExact, Pattern: "timeout", Weight: 1.0},
				},
			},
			expectedComplexity: 3.0, // (1.0 + 1.0) * 1.5 (multiple filter penalty)
			description:        "Multiple filters should have complexity multiplier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := validator.calculateComplexity(tt.expression)
			assert.InDelta(t, tt.expectedComplexity, complexity, 0.1, "Wrong complexity for: %s", tt.description)
		})
	}
}

func TestContentSearchPatternParsing(t *testing.T) {
	validator := NewContentSearchValidator()

	t.Run("extract quoted phrases", func(t *testing.T) {
		query := `"connection failed" OR "timeout error"`
		phrases := validator.extractQuotedPhrases(query)

		assert.Len(t, phrases, 2)
		assert.Contains(t, phrases, "connection failed")
		assert.Contains(t, phrases, "timeout error")
	})

	t.Run("tokenize boolean query", func(t *testing.T) {
		query := "error AND failed OR timeout"
		tokens, err := validator.tokenizeBooleanQuery(query)

		assert.NoError(t, err)
		assert.Contains(t, tokens, "error")
		assert.Contains(t, tokens, "AND")
		assert.Contains(t, tokens, "failed")
		assert.Contains(t, tokens, "OR")
		assert.Contains(t, tokens, "timeout")
	})

	t.Run("invalid boolean query starting with operator", func(t *testing.T) {
		query := "AND error failed"
		_, err := validator.tokenizeBooleanQuery(query)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot start with operator")
	})

	t.Run("invalid boolean query ending with operator", func(t *testing.T) {
		query := "error failed AND"
		_, err := validator.tokenizeBooleanQuery(query)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot end with operator")
	})
}

func TestContentSearchOptions(t *testing.T) {
	validator := NewContentSearchValidator()

	t.Run("default options", func(t *testing.T) {
		expression, err := validator.ParseContentSearch("error", nil)

		assert.NoError(t, err)
		assert.Equal(t, "AND", expression.GlobalOperator)
		assert.True(t, expression.HighlightEnabled)
		assert.True(t, expression.RelevanceScoring)
		assert.Equal(t, 200, expression.MaxSnippetLength)
	})

	t.Run("custom options", func(t *testing.T) {
		options := map[string]string{
			"operator":  "OR",
			"highlight": "false",
			"relevance": "false",
		}

		expression, err := validator.ParseContentSearch("error", options)

		assert.NoError(t, err)
		assert.Equal(t, "OR", expression.GlobalOperator)
		assert.False(t, expression.HighlightEnabled)
		assert.False(t, expression.RelevanceScoring)
	})
}

func TestContentSearchUTF8Validation(t *testing.T) {
	validator := NewContentSearchValidator()

	t.Run("valid UTF-8", func(t *testing.T) {
		query := "错误信息"
		expression, err := validator.ParseContentSearch(query, nil)

		assert.NoError(t, err)
		assert.Len(t, expression.Filters, 1)
		assert.Equal(t, "错误信息", expression.Filters[0].Pattern)
	})

	t.Run("invalid UTF-8", func(t *testing.T) {
		// Create invalid UTF-8 sequence
		invalidUTF8 := string([]byte{0xff, 0xfe, 0xfd})
		_, err := validator.ParseContentSearch(invalidUTF8, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UTF-8")
	})
}

// Benchmark tests for performance validation

func BenchmarkContentSearchValidator_SimpleSearch(b *testing.B) {
	validator := NewContentSearchValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ParseContentSearch("error", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContentSearchValidator_ComplexSearch(b *testing.B) {
	validator := NewContentSearchValidator()
	complexQuery := "boolean:error AND (failed OR timeout) NOT debug"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ParseContentSearch(complexQuery, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContentSearchValidator_RegexSearch(b *testing.B) {
	validator := NewContentSearchValidator()
	regexQuery := "regex:error\\s+(failed|timeout|connection)"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ParseContentSearch(regexQuery, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}