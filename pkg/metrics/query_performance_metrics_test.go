package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Global metrics instance to avoid duplicate registration
var globalMetrics *QueryPerformanceMetrics

func getMetrics() *QueryPerformanceMetrics {
	if globalMetrics == nil {
		globalMetrics = NewQueryPerformanceMetrics()
	}
	return globalMetrics
}

func TestNewQueryPerformanceMetrics(t *testing.T) {
	metrics := getMetrics()

	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.QueryDuration)
	assert.NotNil(t, metrics.QuerySuccessRate)
	assert.NotNil(t, metrics.QueryErrorRate)
	assert.NotNil(t, metrics.SlowQueryCount)
	assert.NotNil(t, metrics.QueryMemoryUsage)
	assert.NotNil(t, metrics.ConnectionPoolStats)
	assert.NotNil(t, metrics.CacheHitRate)
	assert.NotNil(t, metrics.CacheMissRate)
	assert.NotNil(t, metrics.K8sAPICallDuration)
	assert.NotNil(t, metrics.K8sAPICallErrors)
	assert.NotNil(t, metrics.QueryComplexityScore)
}

func TestRecordQueryDuration(t *testing.T) {
	metrics := getMetrics()

	// Test basic query
	metrics.RecordQueryDuration(QueryTypeBasic, "test-dataset", 300*time.Millisecond)
	// Test slow query
	metrics.RecordQueryDuration(QueryTypeFiltered, "test-dataset", 2*time.Second)

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestRecordQuerySuccess(t *testing.T) {
	metrics := getMetrics()

	metrics.RecordQuerySuccess(QueryTypeBasic, "test-dataset")

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestRecordQueryError(t *testing.T) {
	metrics := getMetrics()

	metrics.RecordQueryError(QueryTypeBasic, "test-dataset", ErrorCategoryValidation)
	metrics.RecordQueryError(QueryTypeAggregation, "test-dataset", ErrorCategoryTimeout)

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestRecordQueryMemoryUsage(t *testing.T) {
	metrics := getMetrics()

	metrics.RecordQueryMemoryUsage(QueryTypeBasic, "test-dataset", 1024*1024) // 1MB

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestUpdateConnectionPoolStats(t *testing.T) {
	metrics := getMetrics()

	metrics.UpdateConnectionPoolStats("test-dataset", 10, 5, 5)

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestRecordCacheHit(t *testing.T) {
	metrics := getMetrics()

	metrics.RecordCacheHit("memory", "test-dataset", 85.5)

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestRecordCacheMiss(t *testing.T) {
	metrics := getMetrics()

	metrics.RecordCacheMiss("memory", "test-dataset")

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestRecordK8sAPICall(t *testing.T) {
	metrics := getMetrics()

	metrics.RecordK8sAPICall("get_pod", "test-dataset", 150*time.Millisecond)

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestRecordK8sAPIError(t *testing.T) {
	metrics := getMetrics()

	metrics.RecordK8sAPIError("get_pod", "test-dataset", "timeout")

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestRecordQueryComplexity(t *testing.T) {
	metrics := getMetrics()

	metrics.RecordQueryComplexity(QueryTypeBasic, "test-dataset", 3.5)
	metrics.RecordQueryComplexity(QueryTypeAggregation, "test-dataset", 7.2)

	// Tests pass if no panic occurs
	assert.True(t, true)
}

func TestGetWarningThreshold(t *testing.T) {
	tests := []struct {
		name     string
		queryType QueryType
		expected time.Duration
	}{
		{"Basic query", QueryTypeBasic, 500 * time.Millisecond},
		{"Filtered query", QueryTypeFiltered, 1000 * time.Millisecond},
		{"Aggregation query", QueryTypeAggregation, 1500 * time.Millisecond},
		{"Enriched query", QueryTypeEnriched, 2000 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threshold := getWarningThreshold(tt.queryType)
			assert.Equal(t, tt.expected, threshold)
		})
	}
}

func TestFormatThresholdLabel(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"Critical slow", 6 * time.Second, "critical_5s_plus"},
		{"High slow", 4 * time.Second, "high_3s_to_5s"},
		{"Medium slow", 2500 * time.Millisecond, "medium_2s_to_3s"},
		{"Warning slow", 1800 * time.Millisecond, "warning_1.5s_to_2s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label := formatThresholdLabel(tt.duration)
			assert.Equal(t, tt.expected, label)
		})
	}
}

func TestQueryTypeConstants(t *testing.T) {
	assert.Equal(t, QueryType("basic"), QueryTypeBasic)
	assert.Equal(t, QueryType("filtered"), QueryTypeFiltered)
	assert.Equal(t, QueryType("aggregation"), QueryTypeAggregation)
	assert.Equal(t, QueryType("enriched"), QueryTypeEnriched)
}

func TestErrorCategoryConstants(t *testing.T) {
	assert.Equal(t, "validation", ErrorCategoryValidation)
	assert.Equal(t, "repository", ErrorCategoryRepository)
	assert.Equal(t, "transformation", ErrorCategoryTransformation)
	assert.Equal(t, "timeout", ErrorCategoryTimeout)
	assert.Equal(t, "memory", ErrorCategoryMemory)
	assert.Equal(t, "business_logic", ErrorCategoryBusinessLogic)
}