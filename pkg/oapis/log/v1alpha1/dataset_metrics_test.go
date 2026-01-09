package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDatasetMetrics_RecordDatasetSuccess(t *testing.T) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"
	resultCount := 150
	duration := 500 * time.Millisecond

	metrics.RecordDatasetSuccess(dataset, resultCount, duration)

	stats := metrics.GetDatasetStats(dataset)
	assert.Equal(t, dataset, stats.Dataset)
	assert.Equal(t, int64(1), stats.RequestCount)
	assert.Equal(t, duration, stats.AverageDuration)
	assert.Equal(t, duration, stats.MaxDuration)
	assert.Equal(t, duration, stats.MinDuration)
	assert.True(t, stats.LastQueryTime.After(time.Now().Add(-time.Second)))
}

func TestDatasetMetrics_RecordDatasetError(t *testing.T) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"
	errorType := "not_found"

	metrics.RecordDatasetError(dataset, errorType)
	metrics.RecordDatasetError(dataset, errorType)
	metrics.RecordDatasetError(dataset, "validation_failed")

	stats := metrics.GetDatasetStats(dataset)
	assert.Equal(t, int64(2), stats.ErrorCounts[errorType])
	assert.Equal(t, int64(1), stats.ErrorCounts["validation_failed"])
	assert.Equal(t, 2, len(stats.ErrorCounts))
}

func TestDatasetMetrics_GetAllDatasetStats(t *testing.T) {
	metrics := NewDatasetMetrics()

	// Record metrics for multiple datasets
	metrics.RecordDatasetSuccess("dataset1", 100, 200*time.Millisecond)
	metrics.RecordDatasetSuccess("dataset2", 50, 300*time.Millisecond)
	metrics.RecordDatasetSuccess("dataset3", 25, 150*time.Millisecond)
	metrics.RecordDatasetError("dataset3", "not_found")

	allStats := metrics.GetAllDatasetStats()
	assert.Equal(t, 3, len(allStats))

	assert.Contains(t, allStats, "dataset1")
	assert.Contains(t, allStats, "dataset2")
	assert.Contains(t, allStats, "dataset3")

	assert.Equal(t, int64(1), allStats["dataset1"].RequestCount)
	assert.Equal(t, int64(1), allStats["dataset2"].RequestCount)
	assert.Equal(t, int64(1), allStats["dataset3"].RequestCount)
}

func TestDatasetMetrics_DurationStatistics(t *testing.T) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"

	durations := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		150 * time.Millisecond,
		300 * time.Millisecond,
	}

	for _, duration := range durations {
		metrics.RecordDatasetSuccess(dataset, 100, duration)
	}

	stats := metrics.GetDatasetStats(dataset)
	expectedAvg := (100 + 200 + 150 + 300) * time.Millisecond / 4

	assert.Equal(t, int64(4), stats.RequestCount)
	assert.Equal(t, expectedAvg, stats.AverageDuration)
	assert.Equal(t, 300*time.Millisecond, stats.MaxDuration)
	assert.Equal(t, 100*time.Millisecond, stats.MinDuration)
}

func TestDatasetMetrics_GetTopErrorsByDataset(t *testing.T) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"

	// Record different types of errors with different frequencies
	for i := 0; i < 5; i++ {
		metrics.RecordDatasetError(dataset, "not_found")
	}
	for i := 0; i < 3; i++ {
		metrics.RecordDatasetError(dataset, "validation_failed")
	}
	for i := 0; i < 7; i++ {
		metrics.RecordDatasetError(dataset, "unauthorized")
	}

	topErrors := metrics.GetTopErrorsByDataset(dataset, 2)

	assert.Equal(t, 2, len(topErrors))
	assert.Equal(t, "unauthorized", topErrors[0].ErrorType)
	assert.Equal(t, int64(7), topErrors[0].Count)
	assert.Equal(t, "not_found", topErrors[1].ErrorType)
	assert.Equal(t, int64(5), topErrors[1].Count)
}

func TestDatasetMetrics_ResetStats(t *testing.T) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"

	// Record some metrics
	metrics.RecordDatasetSuccess(dataset, 100, 200*time.Millisecond)
	metrics.RecordDatasetError(dataset, "not_found")

	// Verify metrics exist
	stats := metrics.GetDatasetStats(dataset)
	assert.Equal(t, int64(1), stats.RequestCount)
	assert.Equal(t, 1, len(stats.ErrorCounts))

	// Reset and verify cleanup
	metrics.ResetStats(dataset)
	stats = metrics.GetDatasetStats(dataset)
	assert.Equal(t, int64(0), stats.RequestCount)
	assert.Equal(t, 0, len(stats.ErrorCounts))
	assert.Equal(t, time.Duration(0), stats.AverageDuration)
}

func TestDatasetMetrics_DurationLimiting(t *testing.T) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"

	// Record more than 100 durations to test memory limiting
	for i := 0; i < 150; i++ {
		metrics.RecordDatasetSuccess(dataset, 1, time.Duration(i)*time.Millisecond)
	}

	// Verify the internal slice is limited
	metrics.mu.RLock()
	durationCount := len(metrics.requestDurations[dataset])
	metrics.mu.RUnlock()

	assert.LessOrEqual(t, durationCount, 100, "Duration history should be limited to 100 entries")
}

func TestDatasetMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"

	// Test concurrent access doesn't cause races
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			metrics.RecordDatasetSuccess(dataset, i, time.Duration(i)*time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			metrics.RecordDatasetError(dataset, "test_error")
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	stats := metrics.GetDatasetStats(dataset)
	assert.Equal(t, int64(100), stats.RequestCount)
	assert.Equal(t, int64(100), stats.ErrorCounts["test_error"])
}

func TestDatasetStats_ZeroValues(t *testing.T) {
	metrics := NewDatasetMetrics()
	dataset := "non-existent"

	stats := metrics.GetDatasetStats(dataset)
	assert.Equal(t, dataset, stats.Dataset)
	assert.Equal(t, int64(0), stats.RequestCount)
	assert.Equal(t, 0, len(stats.ErrorCounts))
	assert.True(t, stats.LastQueryTime.IsZero())
	assert.Equal(t, time.Duration(0), stats.AverageDuration)
}

func BenchmarkDatasetMetrics_RecordSuccess(b *testing.B) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"
	duration := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordDatasetSuccess(dataset, 100, duration)
	}
}

func BenchmarkDatasetMetrics_GetStats(b *testing.B) {
	metrics := NewDatasetMetrics()
	dataset := "test-dataset"

	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		metrics.RecordDatasetSuccess(dataset, i, time.Duration(i)*time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.GetDatasetStats(dataset)
	}
}