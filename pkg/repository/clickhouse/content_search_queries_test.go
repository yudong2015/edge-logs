package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

func TestContentSearchQueryBuilder(t *testing.T) {
	builder := NewContentSearchQueryBuilder()
	require.NotNil(t, builder)

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := baseTime.Add(1 * time.Hour)

	baseReq := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		StartTime: &baseTime,
		EndTime:   &endTime,
		Page:      0,
		PageSize:  100,
	}

	t.Run("build basic query without content search", func(t *testing.T) {
		query, args, err := builder.BuildContentSearchQuery(baseReq, nil)

		assert.NoError(t, err)
		assert.NotEmpty(t, query)
		assert.Contains(t, query, "dataset = ?")
		assert.Contains(t, query, "timestamp >= ?")
		assert.Contains(t, query, "timestamp <= ?")
		assert.Len(t, args, 3) // dataset, start_time, end_time
	})

	t.Run("build query with exact content search", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    ContentSearchExact,
					Pattern: "error",
					Weight:  1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: false,
		}

		query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.NotEmpty(t, query)
		assert.Contains(t, query, "position(content, ?) > 0")
		assert.Contains(t, args, "error")
	})

	t.Run("build query with case insensitive search", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    ContentSearchCaseInsensitive,
					Pattern: "ERROR",
					Weight:  1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: false,
		}

		query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "positionCaseInsensitive(content, ?) > 0")
		assert.Contains(t, args, "ERROR")
	})

	t.Run("build query with regex search", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    ContentSearchRegex,
					Pattern: "error\\s+failed",
					Weight:  1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: false,
		}

		query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "match(content, ?)")
		assert.Contains(t, args, "error\\s+failed")
	})

	t.Run("build query with wildcard search", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    ContentSearchWildcard,
					Pattern: "error*",
					Weight:  1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: false,
		}

		query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "content LIKE ?")
		assert.Contains(t, args, "error%") // Converted from wildcard to SQL LIKE
	})

	t.Run("build query with phrase search", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    ContentSearchPhrase,
					Pattern: "connection failed",
					Weight:  1.2,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: false,
		}

		query, _, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "match(content, ?)")
	})

	t.Run("build query with highlighting enabled", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    ContentSearchExact,
					Pattern: "error",
					Weight:  1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: true,
			RelevanceScoring: false,
		}

		query, _, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "replaceRegexpAll")
		assert.Contains(t, query, "highlighted_content_0")
	})

	t.Run("build query with relevance scoring", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    ContentSearchExact,
					Pattern: "error",
					Weight:  2.5,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: true,
		}

		query, _, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "search_relevance_score")
		assert.Contains(t, query, "ORDER BY search_relevance_score DESC")
	})
}

func TestContentSearchQueryBuilder_BooleanOperators(t *testing.T) {
	builder := NewContentSearchQueryBuilder()
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	baseReq := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		StartTime: &baseTime,
		Page:      0,
		PageSize:  100,
	}

	t.Run("build query with AND operators", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:            ContentSearchExact,
					Pattern:         "error",
					BooleanOperator: "AND",
					Weight:          1.0,
				},
				{
					Type:            ContentSearchExact,
					Pattern:         "failed",
					BooleanOperator: "AND",
					Weight:          1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: false,
		}

		query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "position(content, ?) > 0")
		assert.Contains(t, args, "error")
		assert.Contains(t, args, "failed")
	})

	t.Run("build query with OR operators", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:            ContentSearchExact,
					Pattern:         "error",
					BooleanOperator: "OR",
					Weight:          1.0,
				},
				{
					Type:            ContentSearchExact,
					Pattern:         "warning",
					BooleanOperator: "OR",
					Weight:          1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: false,
		}

		query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, " OR ")
		assert.Contains(t, args, "error")
		assert.Contains(t, args, "warning")
	})

	t.Run("build query with NOT operators", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:            ContentSearchExact,
					Pattern:         "debug",
					BooleanOperator: "NOT",
					Weight:          1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: false,
			RelevanceScoring: false,
		}

		query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "NOT (")
		assert.Contains(t, args, "debug")
	})
}

func TestContentSearchQueryBuilder_K8sIntegration(t *testing.T) {
	builder := NewContentSearchQueryBuilder()
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	baseReq := &request.LogQueryRequest{
		Dataset:       "test-dataset",
		StartTime:     &baseTime,
		Namespace:     "production",
		PodName:       "web-server-123",
		ContainerName: "nginx",
		Page:          0,
		PageSize:      100,
	}

	contentSearch := &ContentSearchExpression{
		Filters: []ContentSearchFilter{
			{
				Type:    ContentSearchExact,
				Pattern: "error",
				Weight:  1.0,
			},
		},
		GlobalOperator:   "AND",
		HighlightEnabled: false,
		RelevanceScoring: false,
	}

	query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

	assert.NoError(t, err)
	assert.Contains(t, query, "k8s_namespace_name = ?")
	assert.Contains(t, query, "k8s_pod_name = ?")
	assert.Contains(t, query, "container_name = ?")
	assert.Contains(t, query, "position(content, ?) > 0")

	assert.Contains(t, args, "test-dataset")
	assert.Contains(t, args, "production")
	assert.Contains(t, args, "web-server-123")
	assert.Contains(t, args, "nginx")
	assert.Contains(t, args, "error")
}

func TestContentSearchQueryBuilder_CountQuery(t *testing.T) {
	builder := NewContentSearchQueryBuilder()
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	baseReq := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		StartTime: &baseTime,
		Page:      0,
		PageSize:  100,
	}

	contentSearch := &ContentSearchExpression{
		Filters: []ContentSearchFilter{
			{
				Type:    ContentSearchExact,
				Pattern: "error",
				Weight:  1.0,
			},
		},
		GlobalOperator:   "AND",
		HighlightEnabled: true,
		RelevanceScoring: true,
	}

	t.Run("build count query", func(t *testing.T) {
		query, args, err := builder.BuildContentSearchCountQuery(baseReq, contentSearch)

		assert.NoError(t, err)
		assert.Contains(t, query, "SELECT COUNT(*)")
		assert.Contains(t, query, "position(content, ?) > 0")
		assert.NotContains(t, query, "highlighted_content") // No highlighting in count query
		assert.NotContains(t, query, "search_relevance_score") // No relevance in count query

		assert.Contains(t, args, "test-dataset")
		assert.Contains(t, args, "error")
	})
}

func TestContentSearchQueryBuilder_ComplexQueries(t *testing.T) {
	builder := NewContentSearchQueryBuilder()
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := baseTime.Add(1 * time.Hour)

	baseReq := &request.LogQueryRequest{
		Dataset:       "production-cluster",
		StartTime:     &baseTime,
		EndTime:       &endTime,
		Namespace:     "backend",
		Severity:      "ERROR",
		HostIP:        "192.168.1.100",
		Page:          1,
		PageSize:      50,
	}

	t.Run("build complex query with multiple filters", func(t *testing.T) {
		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:            ContentSearchExact,
					Pattern:         "database",
					BooleanOperator: "AND",
					Weight:          1.5,
				},
				{
					Type:            ContentSearchWildcard,
					Pattern:         "connection*",
					BooleanOperator: "AND",
					Weight:          1.2,
				},
				{
					Type:            ContentSearchExact,
					Pattern:         "debug",
					BooleanOperator: "NOT",
					Weight:          1.0,
				},
			},
			GlobalOperator:   "AND",
			HighlightEnabled: true,
			RelevanceScoring: true,
		}

		query, args, err := builder.BuildContentSearchQuery(baseReq, contentSearch)

		assert.NoError(t, err)

		// Check basic structure
		assert.Contains(t, query, "SELECT")
		assert.Contains(t, query, "FROM logs")
		assert.Contains(t, query, "WHERE")
		assert.Contains(t, query, "ORDER BY")
		assert.Contains(t, query, "LIMIT 50 OFFSET 50") // Page 1 with PageSize 50

		// Check dataset isolation
		assert.Contains(t, query, "dataset = ?")
		assert.Contains(t, args, "production-cluster")

		// Check time range
		assert.Contains(t, query, "timestamp >= ?")
		assert.Contains(t, query, "timestamp <= ?")
		assert.Contains(t, args, baseTime)
		assert.Contains(t, args, endTime)

		// Check K8s filters
		assert.Contains(t, query, "k8s_namespace_name = ?")
		assert.Contains(t, args, "backend")

		// Check severity filter
		assert.Contains(t, query, "severity = ?")
		assert.Contains(t, args, "ERROR")

		// Check host filter
		assert.Contains(t, query, "host_ip = ?")
		assert.Contains(t, args, "192.168.1.100")

		// Check content search conditions
		assert.Contains(t, query, "position(content, ?) > 0")
		assert.Contains(t, query, "content LIKE ?")
		assert.Contains(t, query, "NOT (")

		assert.Contains(t, args, "database")
		assert.Contains(t, args, "connection%") // Wildcard converted to LIKE
		assert.Contains(t, args, "debug")

		// Check highlighting
		assert.Contains(t, query, "highlighted_content_")

		// Check relevance scoring
		assert.Contains(t, query, "search_relevance_score")
		assert.Contains(t, query, "ORDER BY search_relevance_score DESC")
	})
}

func TestContentSearchQueryBuilder_EdgeCases(t *testing.T) {
	builder := NewContentSearchQueryBuilder()

	t.Run("invalid proximity search", func(t *testing.T) {
		baseReq := &request.LogQueryRequest{
			Dataset:  "test",
			Page:     0,
			PageSize: 100,
		}

		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:              ContentSearchProximity,
					Pattern:           "single",
					ProximityDistance: 5,
					Weight:            1.0,
				},
			},
		}

		_, _, err := builder.BuildContentSearchQuery(baseReq, contentSearch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "proximity search requires at least 2 terms")
	})

	t.Run("unsupported search type", func(t *testing.T) {
		baseReq := &request.LogQueryRequest{
			Dataset:  "test",
			Page:     0,
			PageSize: 100,
		}

		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    "unsupported",
					Pattern: "test",
					Weight:  1.0,
				},
			},
		}

		_, _, err := builder.BuildContentSearchQuery(baseReq, contentSearch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported content search type")
	})

	t.Run("boolean search type should error", func(t *testing.T) {
		baseReq := &request.LogQueryRequest{
			Dataset:  "test",
			Page:     0,
			PageSize: 100,
		}

		contentSearch := &ContentSearchExpression{
			Filters: []ContentSearchFilter{
				{
					Type:    ContentSearchBoolean,
					Pattern: "error AND failed",
					Weight:  1.0,
				},
			},
		}

		_, _, err := builder.BuildContentSearchQuery(baseReq, contentSearch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "boolean search should be decomposed")
	})
}

// Benchmark tests for performance validation

func BenchmarkContentSearchQueryBuilder_SimpleQuery(b *testing.B) {
	builder := NewContentSearchQueryBuilder()
	baseTime := time.Now()

	baseReq := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		StartTime: &baseTime,
		Page:      0,
		PageSize:  100,
	}

	contentSearch := &ContentSearchExpression{
		Filters: []ContentSearchFilter{
			{
				Type:    ContentSearchExact,
				Pattern: "error",
				Weight:  1.0,
			},
		},
		GlobalOperator:   "AND",
		HighlightEnabled: false,
		RelevanceScoring: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := builder.BuildContentSearchQuery(baseReq, contentSearch)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContentSearchQueryBuilder_ComplexQuery(b *testing.B) {
	builder := NewContentSearchQueryBuilder()
	baseTime := time.Now()

	baseReq := &request.LogQueryRequest{
		Dataset:       "test-dataset",
		StartTime:     &baseTime,
		Namespace:     "production",
		Severity:      "ERROR",
		Page:          0,
		PageSize:      100,
	}

	contentSearch := &ContentSearchExpression{
		Filters: []ContentSearchFilter{
			{Type: ContentSearchExact, Pattern: "error", BooleanOperator: "AND", Weight: 1.0},
			{Type: ContentSearchWildcard, Pattern: "timeout*", BooleanOperator: "OR", Weight: 1.2},
			{Type: ContentSearchRegex, Pattern: "failed\\s+connection", BooleanOperator: "AND", Weight: 1.5},
			{Type: ContentSearchExact, Pattern: "debug", BooleanOperator: "NOT", Weight: 1.0},
		},
		GlobalOperator:   "AND",
		HighlightEnabled: true,
		RelevanceScoring: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := builder.BuildContentSearchQuery(baseReq, contentSearch)
		if err != nil {
			b.Fatal(err)
		}
	}
}