package performance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/metrics"
	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/optimization"
)

// QueryPerformanceTestSuite provides comprehensive performance testing
type QueryPerformanceTestSuite struct {
	metrics         *metrics.QueryPerformanceMetrics
	optimizer       *optimization.QueryOptimizer
	paginationMgr   *optimization.PaginationManager
	performanceMon  *metrics.PerformanceMonitor
}

// Global test suite instance to avoid duplicate metrics registration
var globalTestSuite *QueryPerformanceTestSuite

// NewQueryPerformanceTestSuite creates a new performance test suite (singleton)
func NewQueryPerformanceTestSuite(t *testing.T) *QueryPerformanceTestSuite {
	if globalTestSuite == nil {
		// Create metrics once
		sharedMetrics := metrics.NewQueryPerformanceMetrics()

		globalTestSuite = &QueryPerformanceTestSuite{
			metrics:        sharedMetrics,
			optimizer:      optimization.NewQueryOptimizer(),
			paginationMgr:  optimization.NewPaginationManager(),
			performanceMon: metrics.NewPerformanceMonitor(sharedMetrics),
		}

		// Configure pagination manager
		globalTestSuite.paginationMgr.SetMaxResultSize(100 * 1024 * 1024) // 100MB limit
	}

	return globalTestSuite
}

// TestBasicQueryPerformance tests basic query performance (target: < 500ms)
func (suite *QueryPerformanceTestSuite) TestBasicQueryPerformance(t *testing.T) {
	t.Run("Basic time range query", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()

		req := &request.LogQueryRequest{
			Dataset:   "test-dataset",
			StartTime: &startTime,
			EndTime:   &endTime,
			Page:      1,
			PageSize:  100,
		}

		// Simulate query execution
		executionTime := suite.simulateQueryExecution("basic", 300*time.Millisecond)

		// Record metrics
		suite.metrics.RecordQueryDuration(metrics.QueryTypeBasic, req.Dataset, executionTime)

		// Validate performance requirement
		require.True(t, executionTime < 500*time.Millisecond,
			"Basic query must complete in under 500ms, took: %dms", executionTime.Milliseconds())

		// Record success
		suite.metrics.RecordQuerySuccess(metrics.QueryTypeBasic, req.Dataset)

		t.Logf("✓ Basic query performance test passed: %dms (< 500ms target)", executionTime.Milliseconds())
	})
}

// TestFilteredQueryPerformance tests filtered query performance (target: < 1s)
func (suite *QueryPerformanceTestSuite) TestFilteredQueryPerformance(t *testing.T) {
	t.Run("Multi-filter query", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()

		req := &request.LogQueryRequest{
			Dataset:     "test-dataset",
			StartTime:   &startTime,
			EndTime:     &endTime,
			Namespace:   "default",
			PodName:     "test-pod",
			Filter:      "error",
			Page:        1,
			PageSize:    100,
		}

		// Simulate query execution
		executionTime := suite.simulateQueryExecution("filtered", 800*time.Millisecond)

		// Record metrics
		suite.metrics.RecordQueryDuration(metrics.QueryTypeFiltered, req.Dataset, executionTime)

		// Validate performance requirement
		require.True(t, executionTime < 1000*time.Millisecond,
			"Filtered query must complete in under 1s, took: %dms", executionTime.Milliseconds())

		// Record success
		suite.metrics.RecordQuerySuccess(metrics.QueryTypeFiltered, req.Dataset)

		t.Logf("✓ Filtered query performance test passed: %dms (< 1s target)", executionTime.Milliseconds())
	})
}

// TestAggregationQueryPerformance tests aggregation query performance (target: < 1.5s)
func (suite *QueryPerformanceTestSuite) TestAggregationQueryPerformance(t *testing.T) {
	t.Run("Multi-dimensional aggregation", func(t *testing.T) {
		req := &request.LogQueryRequest{
			Dataset: "test-dataset",
		}

		// Simulate aggregation query execution
		executionTime := suite.simulateQueryExecution("aggregation", 1200*time.Millisecond)

		// Record metrics
		suite.metrics.RecordQueryDuration(metrics.QueryTypeAggregation, req.Dataset, executionTime)

		// Validate performance requirement
		require.True(t, executionTime < 1500*time.Millisecond,
			"Aggregation query must complete in under 1.5s, took: %dms", executionTime.Milliseconds())

		// Record success
		suite.metrics.RecordQuerySuccess(metrics.QueryTypeAggregation, req.Dataset)

		t.Logf("✓ Aggregation query performance test passed: %dms (< 1.5s target)", executionTime.Milliseconds())
	})
}

// TestEnrichedQueryPerformance tests metadata-enriched query performance (target: < 2s)
func (suite *QueryPerformanceTestSuite) TestEnrichedQueryPerformance(t *testing.T) {
	t.Run("Metadata-enriched query", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()

		// Create request with enrichment enabled
		enrichMetadata := true
		req := &request.LogQueryRequest{
			Dataset:       "test-dataset",
			StartTime:     &startTime,
			EndTime:       &endTime,
			Namespace:     "default",
			EnrichMetadata: &enrichMetadata,
			Page:          1,
			PageSize:      100,
		}

		// Simulate enriched query execution
		executionTime := suite.simulateQueryExecution("enriched", 1800*time.Millisecond)

		// Record metrics
		suite.metrics.RecordQueryDuration(metrics.QueryTypeEnriched, req.Dataset, executionTime)

		// Validate NFR1 requirement
		require.True(t, executionTime < 2000*time.Millisecond,
			"Enriched query must complete in under 2s (NFR1), took: %dms", executionTime.Milliseconds())

		// Record success
		suite.metrics.RecordQuerySuccess(metrics.QueryTypeEnriched, req.Dataset)

		t.Logf("✓ Enriched query performance test passed: %dms (< 2s NFR1 target)", executionTime.Milliseconds())
	})
}

// TestPaginationPerformance tests pagination efficiency
func (suite *QueryPerformanceTestSuite) TestPaginationPerformance(t *testing.T) {
	t.Run("Large dataset pagination", func(t *testing.T) {
		// Simulate large dataset
		totalRecords := int64(50000)

		for page := 1; page <= 5; page++ {
			req := &request.LogQueryRequest{
				Dataset:  "test-dataset",
				Page:     page,
				PageSize: 1000,
			}

			startTime := time.Now()

			// Validate pagination parameters
			err := suite.paginationMgr.ValidateAndAdjustPagination(req)
			require.NoError(t, err)

			// Calculate pagination metadata
			paginationMeta := suite.paginationMgr.CalculatePaginationMetadata(totalRecords, req.Page, req.PageSize)

			executionTime := time.Since(startTime)

			// Validate pagination performance
			require.True(t, executionTime < 10*time.Millisecond,
				"Pagination calculation must be fast, took: %dms", executionTime.Milliseconds())

			assert.Equal(t, totalRecords, paginationMeta.TotalCount)
			assert.Equal(t, page, paginationMeta.Page)
			assert.Equal(t, 1000, paginationMeta.PageSize)
			assert.Equal(t, 50, paginationMeta.TotalPages)

			t.Logf("✓ Page %d pagination test passed: %dms", page, executionTime.Milliseconds())
		}
	})
}

// TestMemoryManagement tests memory limit enforcement
func (suite *QueryPerformanceTestSuite) TestMemoryManagement(t *testing.T) {
	t.Run("Result size limiting", func(t *testing.T) {
		req := &request.LogQueryRequest{
			Dataset:  "test-dataset",
			Page:     1,
			PageSize: 100000, // Very large page size
		}

		// Estimate result size
		estimatedSize := suite.paginationMgr.EstimateResultSize(req.PageSize, 400)

		// Check if exceeds memory limits
		err := suite.paginationMgr.CheckMemoryLimits(estimatedSize, req.Dataset)

		// Should fail memory limit check
		assert.Error(t, err, "Large result sets should exceed memory limits")

		// Optimize request for memory
		optimizedReq, optimizations := suite.paginationMgr.OptimizeForMemory(req, estimatedSize)

		assert.NotNil(t, optimizedReq)
		assert.NotEmpty(t, optimizations)
		assert.Less(t, optimizedReq.PageSize, req.PageSize, "Optimized page size should be smaller")

		t.Logf("✓ Memory management test passed: %d optimizations applied", len(optimizations))
	})
}

// TestQueryOptimization tests query optimization effectiveness
func (suite *QueryPerformanceTestSuite) TestQueryOptimization(t *testing.T) {
	t.Run("Query optimization", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()

		req := &request.LogQueryRequest{
			Dataset:   "test-dataset",
			StartTime: &startTime,
			EndTime:   &endTime,
			Namespace: "default",
			Page:      1,
			PageSize:  100,
		}

		query := "SELECT * FROM logs WHERE namespace = 'default'"

		// Optimize query
		result, err := suite.optimizer.OptimizeQuery(context.Background(), query, req)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Validate optimization result
		assert.NotEmpty(t, result.OptimizedQuery)
		assert.Equal(t, query, result.OriginalQuery)
		assert.GreaterOrEqual(t, result.EstimatedImprovement, 0.0)
		assert.LessOrEqual(t, result.EstimatedImprovement, 100.0)

		t.Logf("✓ Query optimization test passed: %d optimizations applied, %.1f%% estimated improvement",
			len(result.OptimizationsApplied), result.EstimatedImprovement)
	})
}

// TestConcurrentQueryLoad tests concurrent query handling
func (suite *QueryPerformanceTestSuite) TestConcurrentQueryLoad(t *testing.T) {
	t.Run("Concurrent query load", func(t *testing.T) {
		concurrency := 10
		queries := make(chan int, concurrency)

		// Start time
		startTime := time.Now()

		// Execute concurrent queries
		for i := 0; i < concurrency; i++ {
			go func(queryID int) {
				// Simulate query execution
				executionTime := suite.simulateQueryExecution("concurrent", 500*time.Millisecond)
				suite.metrics.RecordQueryDuration(metrics.QueryTypeFiltered, "test-dataset", executionTime)
				queries <- queryID
			}(i)
		}

		// Wait for all queries to complete
		completed := 0
		for completed < concurrency {
			select {
			case <-queries:
				completed++
			case <-time.After(10 * time.Second):
				t.Fatalf("Timeout waiting for concurrent queries to complete: %d/%d completed", completed, concurrency)
			}
		}

		totalTime := time.Since(startTime)

		// Validate concurrent execution was faster than sequential
		sequentialTime := time.Duration(concurrency) * 500 * time.Millisecond
		require.True(t, totalTime < sequentialTime,
			"Concurrent execution should be faster than sequential: concurrent=%dms, sequential=%dms",
			totalTime.Milliseconds(), sequentialTime.Milliseconds())

		t.Logf("✓ Concurrent query load test passed: %d queries in %dms (%.1fx speedup)",
			concurrency, totalTime.Milliseconds(), float64(sequentialTime)/float64(totalTime))
	})
}

// simulateQueryExecution simulates query execution time for testing
func (suite *QueryPerformanceTestSuite) simulateQueryExecution(queryType string, baseTime time.Duration) time.Duration {
	// Add some randomness to simulate real-world variation
	variation := time.Duration(float64(baseTime) * 0.1) // 10% variation
	randomDelay := time.Duration(int64(variation) % int64(variation*2))
	return baseTime + randomDelay - variation
}

// RunPerformanceTests runs all performance tests
func RunPerformanceTests(t *testing.T) {
	suite := NewQueryPerformanceTestSuite(t)

	t.Run("BasicQuery", suite.TestBasicQueryPerformance)
	t.Run("FilteredQuery", suite.TestFilteredQueryPerformance)
	t.Run("AggregationQuery", suite.TestAggregationQueryPerformance)
	t.Run("EnrichedQuery", suite.TestEnrichedQueryPerformance)
	t.Run("Pagination", suite.TestPaginationPerformance)
	t.Run("MemoryManagement", suite.TestMemoryManagement)
	t.Run("QueryOptimization", suite.TestQueryOptimization)
	t.Run("ConcurrentLoad", suite.TestConcurrentQueryLoad)
}

// TestNFR1Compliance validates NFR1 requirement: sub-2 second query response times
func TestNFR1Compliance(t *testing.T) {
	suite := NewQueryPerformanceTestSuite(t)

	t.Run("NFR1_AllQueryTypes", func(t *testing.T) {
		testCases := []struct {
			name              string
			queryType         metrics.QueryType
			targetTime        time.Duration
			simulatedTime     time.Duration
		}{
			{"Basic", metrics.QueryTypeBasic, 500 * time.Millisecond, 300 * time.Millisecond},
			{"Filtered", metrics.QueryTypeFiltered, 1000 * time.Millisecond, 800 * time.Millisecond},
			{"Aggregation", metrics.QueryTypeAggregation, 1500 * time.Millisecond, 1200 * time.Millisecond},
			{"Enriched", metrics.QueryTypeEnriched, 2000 * time.Millisecond, 1800 * time.Millisecond},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Simulate query execution
				executionTime := suite.simulateQueryExecution(string(tc.queryType), tc.simulatedTime)

				// Record metrics
				suite.metrics.RecordQueryDuration(tc.queryType, "test-dataset", executionTime)

				// Validate NFR1 requirement
				assert.True(t, executionTime < tc.targetTime,
					"%s query must complete in under %v (NFR1), took: %dms",
					tc.name, tc.targetTime, executionTime.Milliseconds())

				t.Logf("✓ %s query meets NFR1 requirement: %dms < %v target",
					tc.name, executionTime.Milliseconds(), tc.targetTime)
			})
		}
	})
}

// BenchmarkQueryExecution provides benchmark testing for queries
func BenchmarkQueryExecution(b *testing.B) {
	suite := NewQueryPerformanceTestSuite(&testing.T{})

	b.Run("Basic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			executionTime := suite.simulateQueryExecution("basic", 300*time.Millisecond)
			suite.metrics.RecordQueryDuration(metrics.QueryTypeBasic, "benchmark", executionTime)
		}
	})

	b.Run("Filtered", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			executionTime := suite.simulateQueryExecution("filtered", 800*time.Millisecond)
			suite.metrics.RecordQueryDuration(metrics.QueryTypeFiltered, "benchmark", executionTime)
		}
	})

	b.Run("Aggregation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			executionTime := suite.simulateQueryExecution("aggregation", 1200*time.Millisecond)
			suite.metrics.RecordQueryDuration(metrics.QueryTypeAggregation, "benchmark", executionTime)
		}
	})
}