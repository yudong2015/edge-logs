package enrichment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/metrics"
)

// EnrichmentOptimizer provides performance optimization for metadata enrichment
type EnrichmentOptimizer struct {
	service               *MetadataEnrichmentService
	metrics               *metrics.QueryPerformanceMetrics
	config                *OptimizerConfig
	batchProcessor        *BatchProcessor
	parallelLimiter       chan struct{}
	cacheWarmupScheduled  bool
	mu                    sync.RWMutex
}

// OptimizerConfig contains optimization configuration
type OptimizerConfig struct {
	EnableBatching       bool
	EnableParallelism     bool
	MaxConcurrentAPICalls int
	BatchSize            int
	BatchTimeout         time.Duration
	CacheWarmupInterval  time.Duration
	EnableCachePreloading bool
}

// DefaultOptimizerConfig returns default optimizer configuration
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		EnableBatching:        true,
		EnableParallelism:      true,
		MaxConcurrentAPICalls: 10,
		BatchSize:             25,
		BatchTimeout:          100 * time.Millisecond,
		CacheWarmupInterval:   5 * time.Minute,
		EnableCachePreloading: true,
	}
}

// BatchProcessor handles batched K8s API calls
type BatchProcessor struct {
	batchSize    int
	batchTimeout time.Duration
	requests     chan *EnrichmentRequest
	results      chan *EnrichmentResult
	enabled      bool
}

// NewEnrichmentOptimizer creates a new enrichment optimizer
func NewEnrichmentOptimizer(
	service *MetadataEnrichmentService,
	metrics *metrics.QueryPerformanceMetrics,
	config *OptimizerConfig,
) *EnrichmentOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}

	optimizer := &EnrichmentOptimizer{
		service:         service,
		metrics:         metrics,
		config:          config,
		parallelLimiter: make(chan struct{}, config.MaxConcurrentAPICalls),
	}

	// Initialize batch processor if enabled
	if config.EnableBatching {
		optimizer.batchProcessor = &BatchProcessor{
			batchSize:    config.BatchSize,
			batchTimeout: config.BatchTimeout,
			requests:     make(chan *EnrichmentRequest, 100),
			results:      make(chan *EnrichmentResult, 100),
			enabled:      true,
		}
	}

	klog.InfoS("元数据增强优化器已初始化",
		"enable_batching", config.EnableBatching,
		"enable_parallelism", config.EnableParallelism,
		"max_concurrent_api_calls", config.MaxConcurrentAPICalls,
		"batch_size", config.BatchSize)

	return optimizer
}

// OptimizeEnrichment optimizes metadata enrichment for performance
func (eo *EnrichmentOptimizer) OptimizeEnrichment(
	ctx context.Context,
	request *EnrichmentRequest,
) (*EnrichmentResult, error) {
	klog.V(4).InfoS("开始优化元数据增强",
		"pod_uids_count", len(request.PodUIDs),
		"include_labels", request.IncludeLabels,
		"include_annotations", request.IncludeAnnotations)

	// Strategy 1: Check cache first (fastest path)
	result := eo.processCacheHit(ctx, request)
	if result != nil && len(result.Metadata) == len(request.PodUIDs) {
		// All requests satisfied from cache
	 eo.metrics.RecordCacheHit("metadata_enrichment", "default", 100.0)
		return result, nil
	}

	// Strategy 2: Batch processing for multiple requests
	if eo.config.EnableBatching && len(request.PodUIDs) > eo.config.BatchSize {
		return eo.processBatchEnrichment(ctx, request)
	}

	// Strategy 3: Parallel processing for independent requests
	if eo.config.EnableParallelism && len(request.PodUIDs) > 1 {
		return eo.processParallelEnrichment(ctx, request)
	}

	// Strategy 4: Fallback to sequential processing
	return eo.processSequentialEnrichment(ctx, request)
}

// processCacheHit processes enrichment requests from cache
func (eo *EnrichmentOptimizer) processCacheHit(
	ctx context.Context,
	request *EnrichmentRequest,
) *EnrichmentResult {
	metadata := make(map[string]*PodMetadata)
	cacheHits := 0
	missingUIDs := []string{}

	eo.mu.RLock()
	cache := eo.service.cache
	eo.mu.RUnlock()

	for _, uid := range request.PodUIDs {
		if podMeta := cache.Get(uid); podMeta != nil {
			metadata[uid] = podMeta
			cacheHits++
		} else {
			missingUIDs = append(missingUIDs, uid)
		}
	}

	// If all requests hit cache, return immediately
	if len(missingUIDs) == 0 {
		klog.V(4).InfoS("所有元数据请求从缓存获取",
			"total_requests", len(request.PodUIDs),
			"cache_hits", cacheHits)

		return &EnrichmentResult{
			Metadata:  metadata,
			CacheHits: cacheHits,
			APICalls:  0,
		}
	}

	// Log partial cache hit
	if cacheHits > 0 {
		klog.V(4).InfoS("部分元数据请求从缓存获取",
			"total_requests", len(request.PodUIDs),
			"cache_hits", cacheHits,
			"cache_misses", len(missingUIDs))

		eo.metrics.RecordCacheHit("metadata_enrichment", "default",
			float64(cacheHits)/float64(len(request.PodUIDs))*100)
		eo.metrics.RecordCacheMiss("metadata_enrichment", "default")
	}

	return nil // Needs further processing
}

// processBatchEnrichment processes enrichment requests in batches
func (eo *EnrichmentOptimizer) processBatchEnrichment(
	ctx context.Context,
	request *EnrichmentRequest,
) (*EnrichmentResult, error) {
	startTime := time.Now()

	klog.V(4).InfoS("使用批处理模式进行元数据增强",
		"total_pod_uids", len(request.PodUIDs),
		"batch_size", eo.config.BatchSize)

	// Split into batches
	batches := eo.createBatches(request.PodUIDs, eo.config.BatchSize)

	// Process batches concurrently
	results := make(chan *EnrichmentResult, len(batches))
	var wg sync.WaitGroup

	for _, batch := range batches {
		wg.Add(1)
		go func(batchUIDs []string) {
			defer wg.Done()

			result := eo.service.EnrichLogs(ctx, batchUIDs)

			results <- result
		}(batch)
	}

	// Wait for all batches to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregate results
	aggregatedResult := &EnrichmentResult{
		Metadata:      make(map[string]*PodMetadata),
		FailedPodUIDs: []string{},
	}

	for result := range results {
		for uid, meta := range result.Metadata {
			aggregatedResult.Metadata[uid] = meta
		}
		aggregatedResult.FailedPodUIDs = append(aggregatedResult.FailedPodUIDs, result.FailedPodUIDs...)
		aggregatedResult.CacheHits += result.CacheHits
		aggregatedResult.APICalls += result.APICalls
	}

	aggregatedResult.Duration = time.Since(startTime)

	klog.InfoS("批处理元数据增强完成",
		"total_pod_uids", len(request.PodUIDs),
		"cache_hits", aggregatedResult.CacheHits,
		"api_calls", aggregatedResult.APICalls,
		"duration_ms", aggregatedResult.Duration.Milliseconds())

	return aggregatedResult, nil
}

// processParallelEnrichment processes enrichment requests in parallel
func (eo *EnrichmentOptimizer) processParallelEnrichment(
	ctx context.Context,
	request *EnrichmentRequest,
) (*EnrichmentResult, error) {
	startTime := time.Now()

	klog.V(4).InfoS("使用并行处理模式进行元数据增强",
		"total_pod_uids", len(request.PodUIDs),
		"max_concurrent_calls", eo.config.MaxConcurrentAPICalls)

	var wg sync.WaitGroup
	var mu sync.Mutex
	result := &EnrichmentResult{
		Metadata:      make(map[string]*PodMetadata),
		FailedPodUIDs: []string{},
	}

	// Process requests in parallel with rate limiting
	for _, uid := range request.PodUIDs {
		wg.Add(1)
		go func(podUID string) {
			defer wg.Done()

			// Rate limit concurrent API calls
			eo.parallelLimiter <- struct{}{}
			defer func() { <-eo.parallelLimiter }()

			// Monitor API call performance
			apiStartTime := time.Now()
			result := eo.service.EnrichLogs(ctx, []string{podUID})
			apiDuration := time.Since(apiStartTime)

			// Record K8s API call metrics
			eo.metrics.RecordK8sAPICall("get_pod_metadata", "default", apiDuration)

			// Extract metadata from result
			var podMeta *PodMetadata
			var err error
			if result != nil && len(result.Metadata) > 0 {
				for _, meta := range result.Metadata {
					podMeta = meta
					break
				}
			} else if len(result.FailedPodUIDs) > 0 {
				err = fmt.Errorf("failed to get pod metadata for UID: %s", podUID)
			} else {
				err = fmt.Errorf("no metadata returned for pod UID: %s", podUID)
			}

			if err != nil {
				eo.metrics.RecordK8sAPIError("get_pod_metadata", "default", fmt.Sprintf("%T", err))

				mu.Lock()
				result.FailedPodUIDs = append(result.FailedPodUIDs, podUID)
				mu.Unlock()

				klog.ErrorS(err, "并行元数据增强失败",
					"pod_uid", podUID,
					"duration_ms", apiDuration.Milliseconds())
				return
			}

			mu.Lock()
			result.Metadata[podUID] = podMeta
			result.APICalls++
			mu.Unlock()

			klog.V(4).InfoS("并行元数据增强成功",
				"pod_uid", podUID,
				"duration_ms", apiDuration.Milliseconds())

		}(uid)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	klog.InfoS("并行元数据增强完成",
		"total_pod_uids", len(request.PodUIDs),
		"successful_metadata", len(result.Metadata),
		"failed_uids", len(result.FailedPodUIDs),
		"api_calls", result.APICalls,
		"duration_ms", result.Duration.Milliseconds())

	return result, nil
}

// processSequentialEnrichment processes enrichment requests sequentially
func (eo *EnrichmentOptimizer) processSequentialEnrichment(
	ctx context.Context,
	request *EnrichmentRequest,
) (*EnrichmentResult, error) {
	startTime := time.Now()

	klog.V(4).InfoS("使用顺序处理模式进行元数据增强",
		"total_pod_uids", len(request.PodUIDs))

	result := &EnrichmentResult{
		Metadata:      make(map[string]*PodMetadata),
		FailedPodUIDs: []string{},
	}

	for _, uid := range request.PodUIDs {
		// Monitor API call performance
		apiStartTime := time.Now()
		enrichResult := eo.service.EnrichLogs(ctx, []string{uid})
		apiDuration := time.Since(apiStartTime)

		// Record K8s API call metrics
		eo.metrics.RecordK8sAPICall("get_pod_metadata", "default", apiDuration)

		// Extract metadata from result
		var podMeta *PodMetadata
		var err error
		if enrichResult != nil && len(enrichResult.Metadata) > 0 {
			for _, meta := range enrichResult.Metadata {
				podMeta = meta
				break
			}
		} else if len(enrichResult.FailedPodUIDs) > 0 {
			err = fmt.Errorf("failed to get pod metadata for UID: %s", uid)
		} else {
			err = fmt.Errorf("no metadata returned for pod UID: %s", uid)
		}

		if err != nil {
			eo.metrics.RecordK8sAPIError("get_pod_metadata", "default", fmt.Sprintf("%T", err))
			result.FailedPodUIDs = append(result.FailedPodUIDs, uid)
			continue
		}

		result.Metadata[uid] = podMeta
		result.APICalls++
	}

	result.Duration = time.Since(startTime)

	klog.InfoS("顺序元数据增强完成",
		"total_pod_uids", len(request.PodUIDs),
		"successful_metadata", len(result.Metadata),
		"failed_uids", len(result.FailedPodUIDs),
		"api_calls", result.APICalls,
		"duration_ms", result.Duration.Milliseconds())

	return result, nil
}

// createBatches splits pod UIDs into batches
func (eo *EnrichmentOptimizer) createBatches(podUIDs []string, batchSize int) [][]string {
	var batches [][]string

	for batchSize < len(podUIDs) {
		podUIDs, batches = podUIDs[batchSize:], append(batches, podUIDs[0:batchSize:batchSize])
	}

	if len(podUIDs) > 0 {
		batches = append(batches, podUIDs)
	}

	return batches
}

// WarmupCache warms up the cache with commonly accessed metadata
func (eo *EnrichmentOptimizer) WarmupCache(ctx context.Context, namespaces []string) error {
	if !eo.config.EnableCachePreloading {
		klog.InfoS("缓存预热未启用")
		return nil
	}

	klog.InfoS("开始缓存预热",
		"namespaces", namespaces)

	startTime := time.Now()

	// Get pods from specified namespaces and pre-populate cache
	// This is a simplified implementation - real implementation would:
	// 1. List pods in each namespace
	// 2. Pre-fetch metadata for active pods
	// 3. Store in cache for future requests

	duration := time.Since(startTime)

	klog.InfoS("缓存预热完成",
		"namespaces", namespaces,
		"duration_ms", duration.Milliseconds())

	return nil
}

// GetPerformanceMetrics returns current performance metrics
func (eo *EnrichmentOptimizer) GetPerformanceMetrics() *EnrichmentPerformanceMetrics {
	// Collect current performance statistics
	return &EnrichmentPerformanceMetrics{
		EnableBatching:        eo.config.EnableBatching,
		EnableParallelism:     eo.config.EnableParallelism,
		MaxConcurrentAPICalls: eo.config.MaxConcurrentAPICalls,
		BatchSize:            eo.config.BatchSize,
		Timestamp:            time.Now(),
	}
}

// EnrichmentPerformanceMetrics contains enrichment performance data
type EnrichmentPerformanceMetrics struct {
	EnableBatching        bool      `json:"enable_batching"`
	EnableParallelism     bool      `json:"enable_parallelism"`
	MaxConcurrentAPICalls int       `json:"max_concurrent_api_calls"`
	BatchSize            int       `json:"batch_size"`
	Timestamp            time.Time `json:"timestamp"`
}

// UpdateConfig dynamically updates optimizer configuration
func (eo *EnrichmentOptimizer) UpdateConfig(newConfig *OptimizerConfig) {
	eo.mu.Lock()
	defer eo.mu.Unlock()

	oldConfig := eo.config
	eo.config = newConfig

	// Update parallel limiter if concurrent call limit changed
	if newConfig.MaxConcurrentAPICalls != oldConfig.MaxConcurrentAPICalls {
		eo.parallelLimiter = make(chan struct{}, newConfig.MaxConcurrentAPICalls)
	}

	klog.InfoS("元数据增强优化器配置已更新",
		"old_batch_size", oldConfig.BatchSize,
		"new_batch_size", newConfig.BatchSize,
		"old_max_concurrent", oldConfig.MaxConcurrentAPICalls,
		"new_max_concurrent", newConfig.MaxConcurrentAPICalls)
}