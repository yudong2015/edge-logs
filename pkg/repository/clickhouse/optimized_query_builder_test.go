package clickhouse

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// TestOptimizedQueryBuilder tests the optimized query builder with MATERIALIZED columns
func TestOptimizedQueryBuilder(t *testing.T) {
	t.Run("NewOptimizedQueryBuilder", func(t *testing.T) {
		oqb := NewOptimizedQueryBuilder(nil)
		assert.NotNil(t, oqb)
		assert.NotNil(t, oqb.QueryBuilder)
		assert.False(t, oqb.hasMaterializedColumns)
		assert.Nil(t, oqb.db)
	})

	t.Run("GetMaterializedColumnInfo", func(t *testing.T) {
		info := GetMaterializedColumnInfo()
		assert.NotEmpty(t, info)

		// Check for expected columns
		columnNames := make([]string, len(info))
		for i, col := range info {
			columnNames[i] = col.ColumnName
			assert.NotEmpty(t, col.SourceMap)
			assert.NotEmpty(t, col.SourceField)
			assert.NotEmpty(t, col.IndexType)
			assert.NotEmpty(t, col.Benefit)
		}

		assert.Contains(t, columnNames, "dataset")
		assert.Contains(t, columnNames, "k8s_namespace_name")
		assert.Contains(t, columnNames, "k8s_pod_name")
		assert.Contains(t, columnNames, "k8s_container_name")
	})

	t.Run("EstimateQueryPerformance", func(t *testing.T) {
		oqb := NewOptimizedQueryBuilder(nil)

		tests := []struct {
			name                       string
			hasMaterializedColumns     bool
			req                        *request.LogQueryRequest
			expectedQueryType          string
			expectedExpectedPerformance string
		}{
			{
				name:                   "MATERIALIZED columns - namespace filter",
				hasMaterializedColumns: true,
				req: &request.LogQueryRequest{
					Namespace: "default",
					PageSize:  100,
				},
				expectedQueryType:          "MATERIALIZED_COLUMN_QUERY",
				expectedExpectedPerformance: "FAST",
			},
			{
				name:                   "MATERIALIZED columns - pod filter",
				hasMaterializedColumns: true,
				req: &request.LogQueryRequest{
					PodName:  "test-pod",
					PageSize: 100,
				},
				expectedQueryType:          "MATERIALIZED_COLUMN_QUERY",
				expectedExpectedPerformance: "FAST",
			},
			{
				name:                   "MATERIALIZED columns - no filters",
				hasMaterializedColumns: true,
				req: &request.LogQueryRequest{
					Filter:   "error",
					PageSize: 100,
				},
				expectedQueryType:          "MAP_FIELD_QUERY",
				expectedExpectedPerformance: "SLOWER",
			},
			{
				name:                   "Map columns - namespace filter",
				hasMaterializedColumns: false,
				req: &request.LogQueryRequest{
					Namespace: "default",
					PageSize:  100,
				},
				expectedQueryType:          "MAP_FIELD_QUERY",
				expectedExpectedPerformance: "SLOWER",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				oqb.hasMaterializedColumns = tt.hasMaterializedColumns

				estimate := oqb.EstimateQueryPerformance(tt.req)

				assert.Equal(t, tt.expectedQueryType, estimate["query_type"])
				assert.Equal(t, tt.expectedExpectedPerformance, estimate["expected_performance"])
				assert.NotEmpty(t, estimate["performance_factor"])
				assert.NotEmpty(t, estimate["index_usage"])
			})
		}
	})

	t.Run("BuildOptimizedLogQuery with nil DB", func(t *testing.T) {
		oqb := NewOptimizedQueryBuilder(nil)
		oqb.hasMaterializedColumns = false

		now := time.Now()
		req := &request.LogQueryRequest{
			Dataset:   "test-dataset",
			Namespace: "default",
			StartTime: &now,
			EndTime:   &now,
			PageSize:  100,
		}

		query, args, err := oqb.BuildOptimizedLogQuery(context.Background(), req)
		assert.NoError(t, err)
		assert.NotEmpty(t, query)
		assert.NotEmpty(t, args)
		assert.Contains(t, query, "FROM logs")

		// Without DB, should fall back to Map columns
		assert.Contains(t, query, "ResourceAttributes['k8s.namespace.name']")
	})

	t.Run("BuildOptimizedLogQuery with MATERIALIZED columns", func(t *testing.T) {
		oqb := NewOptimizedQueryBuilder(nil)
		oqb.hasMaterializedColumns = true // Simulate MATERIALIZED columns exist

		now := time.Now()
		req := &request.LogQueryRequest{
			Dataset:        "test-dataset",
			Namespace:      "default",
			PodName:        "test-pod",
			ContainerName:  "test-container",
			HostIP:         "192.168.1.100",
			HostName:       "test-host",
			Severity:       "ERROR",
			Filter:         "exception",
			PageSize:       100,
			Page:           1,
			OrderBy:        "Timestamp",
			Direction:      "desc",
		}

		query, args, err := oqb.BuildOptimizedLogQuery(context.Background(), req)
		assert.NoError(t, err)
		assert.NotEmpty(t, args)
		assert.Contains(t, query, "FROM logs")

		// Should use MATERIALIZED columns
		assert.Contains(t, query, "dataset = ?")
		assert.Contains(t, query, "k8s_namespace_name = ?")
		assert.Contains(t, query, "k8s_pod_name LIKE ?")
		assert.Contains(t, query, "k8s_container_name = ?")
		assert.Contains(t, query, "host_ip = ?")
		assert.Contains(t, query, "host_name = ?")

		// Should NOT use Map-based filtering for K8s metadata
		assert.NotContains(t, query, "ResourceAttributes['k8s.namespace.name']")
		assert.NotContains(t, query, "ResourceAttributes['k8s.pod.name']")
		assert.NotContains(t, query, "ResourceAttributes['k8s.container.name']")

		// Should include time filtering
		assert.Contains(t, query, "Timestamp >= ?")
		assert.Contains(t, query, "Timestamp <= ?")

		// Should include severity and full-text search
		assert.Contains(t, query, "SeverityText = ?")
		assert.Contains(t, query, "hasToken(Body, ?)")

		// Should include ORDER BY and pagination
		assert.Contains(t, query, "ORDER BY")
		assert.Contains(t, query, "LIMIT 100")
		assert.Contains(t, query, "OFFSET 100")
	})

	t.Run("BuildOptimizedCountQuery with MATERIALIZED columns", func(t *testing.T) {
		oqb := NewOptimizedQueryBuilder(nil)
		oqb.hasMaterializedColumns = true

		req := &request.LogQueryRequest{
			Dataset:   "test-dataset",
			Namespace: "default",
			PodName:   "test-pod",
		}

		query, args, err := oqb.BuildOptimizedCountQuery(context.Background(), req)
		assert.NoError(t, err)
		assert.NotEmpty(t, args)
		assert.Contains(t, query, "SELECT count(*) FROM logs")

		// Should use MATERIALIZED columns
		assert.Contains(t, query, "dataset = ?")
		assert.Contains(t, query, "k8s_namespace_name = ?")
		assert.Contains(t, query, "k8s_pod_name LIKE ?")

		// Should NOT include ORDER BY or LIMIT in count query
		assert.NotContains(t, query, "ORDER BY")
		assert.NotContains(t, query, "LIMIT")
	})

	t.Run("BuildOptimizedLogQuery without filters", func(t *testing.T) {
		oqb := NewOptimizedQueryBuilder(nil)
		oqb.hasMaterializedColumns = true

		req := &request.LogQueryRequest{
			PageSize:  100,
			OrderBy:   "Timestamp",
			Direction: "asc",
		}

		query, args, err := oqb.BuildOptimizedLogQuery(context.Background(), req)
		assert.NoError(t, err)
		assert.Contains(t, query, "FROM logs")
		assert.Contains(t, query, "ORDER BY")
		assert.Contains(t, query, "LIMIT 100")

		// Should have minimal conditions
		assert.NotContains(t, query, "WHERE")
		assert.Empty(t, args) // No conditions = no args
	})

	t.Run("BuildOptimizedLogQuery with tags", func(t *testing.T) {
		oqb := NewOptimizedQueryBuilder(nil)
		oqb.hasMaterializedColumns = true

		req := &request.LogQueryRequest{
			Namespace: "default",
			Tags: map[string]string{
				"app":     "frontend",
				"version": "v1.2.3",
			},
			PageSize:  50,
			OrderBy:   "Timestamp",
			Direction: "desc",
		}

		query, args, err := oqb.BuildOptimizedLogQuery(context.Background(), req)
		assert.NoError(t, err)
		assert.Contains(t, query, "FROM logs")

		// Should use MATERIALIZED column for namespace
		assert.Contains(t, query, "k8s_namespace_name = ?")

		// Should handle tags
		assert.Contains(t, query, "LogAttributes[?] = ?")

		// Should have tag args (key-value pairs)
		tagArgsCount := 0
		for _, arg := range args {
			if s, ok := arg.(string); ok && (s == "app" || s == "version" || s == "frontend" || s == "v1.2.3") {
				tagArgsCount++
			}
		}
		assert.Equal(t, 4, tagArgsCount) // 2 keys + 2 values
	})
}

// TestGetColumnUsageStats tests column usage statistics
func TestGetColumnUsageStats(t *testing.T) {
	t.Run("with nil DB", func(t *testing.T) {
		oqb := NewOptimizedQueryBuilder(nil)
		stats, err := oqb.GetColumnUsageStats(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, false, stats["has_materialized_columns"])
	})

	// Integration tests would require a real ClickHouse instance
	// These should be in separate test files with proper test setup
}

// TestMaterializedColumnInfoStructure tests the structure of column info
func TestMaterializedColumnInfoStructure(t *testing.T) {
	info := GetMaterializedColumnInfo()

	// Verify all required fields are present
	for _, col := range info {
		assert.NotEmpty(t, col.ColumnName, "ColumnName should not be empty")
		assert.NotEmpty(t, col.SourceMap, "SourceMap should not be empty")
		assert.NotEmpty(t, col.SourceField, "SourceField should not be empty")
		assert.NotEmpty(t, col.IndexType, "IndexType should not be empty")
		assert.NotEmpty(t, col.Benefit, "Benefit should not be empty")

		// Verify SourceMap is either ResourceAttributes or LogAttributes
		assert.Contains(t, []string{"ResourceAttributes", "LogAttributes"}, col.SourceMap)

		// Verify IndexType follows expected pattern
		assert.Contains(t, col.IndexType, "set(")
	}
}

// BenchmarkQueryBuilding compares query building performance
func BenchmarkQueryBuilding(b *testing.B) {
	req := &request.LogQueryRequest{
		Dataset:        "test-dataset",
		Namespace:      "default",
		PodName:        "test-pod",
		ContainerName:  "test-container",
		HostIP:         "192.168.1.100",
		Severity:       "ERROR",
		Filter:         "exception",
		PageSize:       100,
		Page:           1,
		OrderBy:        "Timestamp",
		Direction:      "desc",
	}

	b.Run("OptimizedQueryBuilder with MATERIALIZED columns", func(b *testing.B) {
		oqb := NewOptimizedQueryBuilder(nil)
		oqb.hasMaterializedColumns = true
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = oqb.BuildOptimizedLogQuery(ctx, req)
		}
	})

	b.Run("Standard QueryBuilder", func(b *testing.B) {
		qb := NewQueryBuilder()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = qb.BuildLogQuery(req)
		}
	})
}

// TestBuildWithMaterializedColumns tests the materialized column path
func TestBuildWithMaterializedColumns(t *testing.T) {
	oqb := NewOptimizedQueryBuilder(nil)
	oqb.hasMaterializedColumns = true

	tests := []struct {
		name     string
		req      *request.LogQueryRequest
		contains []string
	}{
		{
			name: "dataset filter",
			req: &request.LogQueryRequest{
				Dataset: "my-dataset",
			},
			contains: []string{"dataset = ?"},
		},
		{
			name: "namespace filter",
			req: &request.LogQueryRequest{
				Namespace: "kube-system",
			},
			contains: []string{"k8s_namespace_name = ?"},
		},
		{
			name: "pod filter",
			req: &request.LogQueryRequest{
				PodName: "frontend-abc123",
			},
			contains: []string{"k8s_pod_name LIKE ?"},
		},
		{
			name: "container filter",
			req: &request.LogQueryRequest{
				ContainerName: "nginx",
			},
			contains: []string{"k8s_container_name = ?"},
		},
		{
			name: "node filter",
			req: &request.LogQueryRequest{
				NodeName: "node-1",
			},
			contains: []string{"k8s_node_name = ?"},
		},
		{
			name: "host IP filter",
			req: &request.LogQueryRequest{
				HostIP: "10.0.0.1",
			},
			contains: []string{"host_ip = ?"},
		},
		{
			name: "host name filter",
			req: &request.LogQueryRequest{
				HostName: "worker-1",
			},
			contains: []string{"host_name = ?"},
		},
		{
			name: "multiple filters",
			req: &request.LogQueryRequest{
				Dataset:       "my-dataset",
				Namespace:     "default",
				PodName:       "app-123",
				ContainerName: "container",
				HostIP:        "192.168.1.1",
			},
			contains: []string{
				"dataset = ?",
				"k8s_namespace_name = ?",
				"k8s_pod_name LIKE ?",
				"k8s_container_name = ?",
				"host_ip = ?",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oqb.Reset()
			oqb.buildWithMaterializedColumns(tt.req)

			query := oqb.baseQuery.String()

			for _, expected := range tt.contains {
				assert.Contains(t, query, expected)
			}
		})
	}
}

// TestBuildWithMapColumns tests the map-based fallback path
func TestBuildWithMapColumns(t *testing.T) {
	oqb := NewOptimizedQueryBuilder(nil)
	oqb.hasMaterializedColumns = false

	tests := []struct {
		name     string
		req      *request.LogQueryRequest
		contains []string
	}{
		{
			name: "dataset filter",
			req: &request.LogQueryRequest{
				Dataset: "my-dataset",
			},
			contains: []string{
				"(ServiceName = ? OR ResourceAttributes['k8s.namespace.name'] = ?)",
			},
		},
		{
			name: "namespace filter",
			req: &request.LogQueryRequest{
				Namespace: "kube-system",
			},
			contains: []string{"ResourceAttributes['k8s.namespace.name'] = ?"},
		},
		{
			name: "pod filter",
			req: &request.LogQueryRequest{
				PodName: "frontend-abc123",
			},
			contains: []string{"ResourceAttributes['k8s.pod.name'] = ?"},
		},
		{
			name: "container filter",
			req: &request.LogQueryRequest{
				ContainerName: "nginx",
			},
			contains: []string{"ResourceAttributes['k8s.container.name'] = ?"},
		},
		{
			name: "node filter",
			req: &request.LogQueryRequest{
				NodeName: "node-1",
			},
			contains: []string{"LogAttributes['k8s.node.name'] = ?"},
		},
		{
			name: "host IP filter",
			req: &request.LogQueryRequest{
				HostIP: "10.0.0.1",
			},
			contains: []string{"ResourceAttributes['host.ip'] = ?"},
		},
		{
			name: "host name filter",
			req: &request.LogQueryRequest{
				HostName: "worker-1",
			},
			contains: []string{"ResourceAttributes['host.name'] = ?"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oqb.Reset()
			oqb.buildWithMapColumns(tt.req)

			query := oqb.baseQuery.String()

			for _, expected := range tt.contains {
				assert.Contains(t, query, expected)
			}
		})
	}
}

// TestQueryMaterializedColumnStats tests the statistics query helper
func TestQueryMaterializedColumnStats(t *testing.T) {
	// This would require a real ClickHouse instance for full testing
	// For now, test error handling
	db, err := sql.Open("clickhouse", "invalid-connection-string")
	require.NoError(t, err)

	stats, err := queryMaterializedColumnStats(context.Background(), db)
	assert.Error(t, err) // Should fail with invalid connection
	assert.Nil(t, stats)
}
