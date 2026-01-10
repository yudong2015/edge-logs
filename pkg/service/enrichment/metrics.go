package enrichment

import (
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// EnrichmentMetrics tracks enrichment performance metrics
type EnrichmentMetrics struct {
	mu                sync.RWMutex
	totalEnrichments  int64
	totalCacheHits   int64
	totalAPICalls    int64
	totalFailures    int64
	totalDuration     time.Duration
}

// NewEnrichmentMetrics creates new enrichment metrics
func NewEnrichmentMetrics() *EnrichmentMetrics {
	return &EnrichmentMetrics{}
}

// RecordEnrichment records metrics for enrichment operation
func (m *EnrichmentMetrics) RecordEnrichment(totalPods, cacheHits, apiCalls int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalEnrichments++
	m.totalCacheHits += int64(cacheHits)
	m.totalAPICalls += int64(apiCalls)
	m.totalDuration += duration

	if apiCalls < totalPods {
		// Calculate failures (pods that couldn't be enriched)
		failed := int64(totalPods - cacheHits - apiCalls)
		m.totalFailures += failed
	}

	klog.V(4).InfoS("Enrichment metrics recorded",
		"total_pods", totalPods,
		"cache_hits", cacheHits,
		"api_calls", apiCalls,
		"duration_ms", duration.Milliseconds())
}

// GetMetrics returns current enrichment metrics
func (m *EnrichmentMetrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cacheHitRate := 0.0
	if m.totalEnrichments > 0 {
		cacheHitRate = float64(m.totalCacheHits) / float64(m.totalEnrichments)
	}

	avgDuration := time.Duration(0)
	if m.totalEnrichments > 0 {
		avgDuration = time.Duration(int64(m.totalDuration) / m.totalEnrichments)
	}

	return map[string]interface{}{
		"total_enrichments": m.totalEnrichments,
		"cache_hits":       m.totalCacheHits,
		"api_calls":        m.totalAPICalls,
		"failures":         m.totalFailures,
		"cache_hit_rate":   cacheHitRate,
		"avg_duration_ms":  avgDuration.Milliseconds(),
	}
}
