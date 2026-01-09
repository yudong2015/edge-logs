package v1alpha1

import (
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// DatasetMetrics provides dataset-specific metrics tracking
type DatasetMetrics struct {
	mu                sync.RWMutex
	requestCounts     map[string]int64
	errorCounts       map[string]map[string]int64 // dataset -> error_type -> count
	lastQueryTime     map[string]time.Time
	totalDurations    map[string]time.Duration
	requestDurations  map[string][]time.Duration
	maxDuration       map[string]time.Duration
	minDuration       map[string]time.Duration
}

// NewDatasetMetrics creates a new dataset metrics tracker
func NewDatasetMetrics() *DatasetMetrics {
	return &DatasetMetrics{
		requestCounts:    make(map[string]int64),
		errorCounts:     make(map[string]map[string]int64),
		lastQueryTime:   make(map[string]time.Time),
		totalDurations:  make(map[string]time.Duration),
		requestDurations: make(map[string][]time.Duration),
		maxDuration:     make(map[string]time.Duration),
		minDuration:     make(map[string]time.Duration),
	}
}

// RecordDatasetSuccess records successful dataset query metrics
func (m *DatasetMetrics) RecordDatasetSuccess(dataset string, resultCount int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCounts[dataset]++
	m.lastQueryTime[dataset] = time.Now()
	m.totalDurations[dataset] += duration

	// Track duration statistics
	if m.requestDurations[dataset] == nil {
		m.requestDurations[dataset] = make([]time.Duration, 0, 100)
	}

	// Keep only last 100 durations for memory efficiency
	if len(m.requestDurations[dataset]) >= 100 {
		m.requestDurations[dataset] = m.requestDurations[dataset][1:]
	}
	m.requestDurations[dataset] = append(m.requestDurations[dataset], duration)

	// Update min/max durations
	if m.maxDuration[dataset] == 0 || duration > m.maxDuration[dataset] {
		m.maxDuration[dataset] = duration
	}
	if m.minDuration[dataset] == 0 || duration < m.minDuration[dataset] {
		m.minDuration[dataset] = duration
	}

	klog.V(3).InfoS("Dataset query metrics recorded",
		"dataset", dataset,
		"result_count", resultCount,
		"duration_ms", duration.Milliseconds(),
		"total_requests", m.requestCounts[dataset])
}

// RecordDatasetError records dataset error metrics
func (m *DatasetMetrics) RecordDatasetError(dataset, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errorCounts[dataset] == nil {
		m.errorCounts[dataset] = make(map[string]int64)
	}
	m.errorCounts[dataset][errorType]++

	klog.V(3).InfoS("Dataset error metrics recorded",
		"dataset", dataset,
		"error_type", errorType,
		"error_count", m.errorCounts[dataset][errorType])
}

// GetDatasetStats returns statistics for a specific dataset
func (m *DatasetMetrics) GetDatasetStats(dataset string) DatasetStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := DatasetStats{
		Dataset:       dataset,
		RequestCount:  m.requestCounts[dataset],
		LastQueryTime: m.lastQueryTime[dataset],
		ErrorCounts:   make(map[string]int64),
	}

	// Copy error counts
	if errs := m.errorCounts[dataset]; errs != nil {
		for errorType, count := range errs {
			stats.ErrorCounts[errorType] = count
		}
	}

	// Calculate average duration
	if stats.RequestCount > 0 {
		stats.AverageDuration = m.totalDurations[dataset] / time.Duration(stats.RequestCount)
		stats.MaxDuration = m.maxDuration[dataset]
		stats.MinDuration = m.minDuration[dataset]
	}

	return stats
}

// GetAllDatasetStats returns statistics for all tracked datasets
func (m *DatasetMetrics) GetAllDatasetStats() map[string]DatasetStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]DatasetStats)
	for dataset := range m.requestCounts {
		stats[dataset] = m.GetDatasetStats(dataset)
	}

	return stats
}

// ResetStats resets all metrics for a dataset (useful for testing)
func (m *DatasetMetrics) ResetStats(dataset string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.requestCounts, dataset)
	delete(m.errorCounts, dataset)
	delete(m.lastQueryTime, dataset)
	delete(m.totalDurations, dataset)
	delete(m.requestDurations, dataset)
	delete(m.maxDuration, dataset)
	delete(m.minDuration, dataset)

	klog.V(4).InfoS("Dataset metrics reset", "dataset", dataset)
}

// GetTopErrorsByDataset returns top errors for dataset analysis
func (m *DatasetMetrics) GetTopErrorsByDataset(dataset string, limit int) []ErrorStat {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errors []ErrorStat
	if errs := m.errorCounts[dataset]; errs != nil {
		for errorType, count := range errs {
			errors = append(errors, ErrorStat{
				ErrorType: errorType,
				Count:     count,
			})
		}
	}

	// Simple sort by count (descending)
	for i := 0; i < len(errors); i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[j].Count > errors[i].Count {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}

	if len(errors) > limit {
		errors = errors[:limit]
	}

	return errors
}

// DatasetStats represents statistics for a dataset
type DatasetStats struct {
	Dataset         string                 `json:"dataset"`
	RequestCount    int64                  `json:"request_count"`
	ErrorCounts     map[string]int64       `json:"error_counts"`
	LastQueryTime   time.Time              `json:"last_query_time"`
	AverageDuration time.Duration          `json:"average_duration"`
	MaxDuration     time.Duration          `json:"max_duration"`
	MinDuration     time.Duration          `json:"min_duration"`
}

// ErrorStat represents error statistics
type ErrorStat struct {
	ErrorType string `json:"error_type"`
	Count     int64  `json:"count"`
}