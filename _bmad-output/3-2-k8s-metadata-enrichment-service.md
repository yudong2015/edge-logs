# Story 3.2: K8s Metadata Enrichment Service

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a system operator,
I want log entries enriched with additional Kubernetes metadata (labels, annotations, pod specifications),
So that I can get complete context about pods, services, and deployments for better troubleshooting and operational insights across my edge computing infrastructure.

## Acceptance Criteria

**Given** Epic 2 comprehensive filtering foundation is implemented (dataset routing, time filtering, K8s filtering, content search)
**And** Basic log queries return K8s metadata already stored in ClickHouse (namespace, pod_name, pod_uid, node_name)
**When** I enable metadata enrichment for log queries
**Then** Log results include additional K8s metadata beyond what's stored in ClickHouse (pod labels, annotations, pod specifications)
**And** Pod labels and annotations are retrieved from the K8s API when available and cached for performance
**And** Metadata enrichment is optional and can be enabled per query via `enrich=true` parameter
**And** K8s API errors don't break log queries (graceful degradation with enrichment status indicators)
**And** Enriched metadata is cached with configurable TTL to minimize K8s API load
**And** Cache invalidation handles pod updates and deletions properly
**And** Enrichment works efficiently with batch API calls for multiple unique pods
**And** Response includes enrichment metadata indicating success/failure status

## Tasks / Subtasks

- [ ] Create K8s metadata enrichment service layer (AC: 1, 2, 7)
  - [ ] Create MetadataEnrichmentService with K8s client integration
  - [ ] Implement Pod metadata retrieval (labels, annotations, specifications)
  - [ ] Add batch enrichment for multiple pods in single request
  - [ ] Implement in-memory cache with TTL for metadata
  - [ ] Add cache invalidation for pod updates/deletions using informers
  - [ ] Create enrichment status tracking (success, partial, failed)

- [ ] Integrate enrichment with existing query service (AC: 3, 8)
  - [ ] Add `enrich` query parameter support to handler
  - [ ] Enhance service layer to conditionally invoke enrichment
  - [ ] Merge enriched metadata with log entries from ClickHouse
  - [ ] Handle graceful degradation when K8s API unavailable
  - [ ] Add enrichment metadata to response

- [ ] Create enrichment models and response structures (AC: 6)
  - [ ] Define EnrichedLogEntry with enriched fields
  - [ ] Create PodMetadata model (labels, annotations, spec)
  - [ ] Define EnrichmentStatus response field
  - [ ] Add enrichment metadata (cache_hit, api_duration, pod_count)

- [ ] Implement caching and performance optimization (AC: 5, 9)
  - [ ] Create MetadataCache with TTL-based expiration
  - [ ] Implement pod UID-based cache key generation
  - [ ] Add cache warming for frequently accessed pods
  - [ ] Implement K8s informer for watch-based cache invalidation
  - [ ] Add metrics for enrichment performance

- [ ] Add comprehensive testing (AC: 4, 7)
  - [ ] Unit tests for enrichment service
  - [ ] Mock K8s API client tests
  - [ ] Cache behavior tests
  - [ ] Graceful degradation tests
  - [ ] Batch enrichment tests
  - [ ] Integration tests

## Dev Notes

### Architecture Compliance Requirements

**Critical:** Story 3-2 implements K8s metadata enrichment on top of Epic 2's query foundation, following architecture.md's specification for "按需关联 Labels/Annotations" (Labels/Annotations on-demand correlation). The architecture explicitly states:
- **95% 的查询无需访问 K8s API** - basic K8s metadata is already stored in ClickHouse
- **Pod Labels/Annotations** should be retrieved from K8s API only when explicitly requested
- **按需关联 Labels/Annotations，不影响核心链路** - Enrichment should not impact the core query path

**Key Technical Requirements from architecture.md:**
- **K8s client-go v0.31.2** for metadata correlation
- **Graceful degradation** - K8s API errors shouldn't break log queries
- **Caching** - Enriched metadata must be cached for performance
- **Optional enrichment** - Per-query enablement via parameter

### Current Codebase Patterns

**From Story 3-1 (just completed):**
- Services are in `pkg/service/query/`
- Repository in `pkg/repository/clickhouse/`
- API handlers in `pkg/oapis/log/v1alpha1/`
- Request models in `pkg/model/request/`
- Response models in `pkg/model/response/`

**Established patterns:**
- Use `klog/v2` for structured logging
- Service layer validates and delegates to repository
- Handler uses go-restful/v3 framework
- Response wrapper pattern for error handling

### K8s Metadata Enrichment Architecture

```go
// Metadata enrichment flow (from architecture.md section 4.4)
// Step 3: Optional K8s API correlation for Labels/Annotations

// pkg/service/enrichment/metadata_service.go
package enrichment

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// MetadataEnrichmentService provides K8s metadata enrichment
type MetadataEnrichmentService struct {
	client           kubernetes.Interface
	informerFactory  informers.SharedInformerFactory
	podInformer      cache.SharedIndexInformer
	cache            *MetadataCache
	config           *EnrichmentConfig
	metrics          *EnrichmentMetrics
}

// EnrichmentConfig configures enrichment behavior
type EnrichmentConfig struct {
	CacheTTL              time.Duration
	EnableInformer        bool
	MaxBatchSize          int
	APITimeout            time.Duration
	IncludeLabels         bool
	IncludeAnnotations    bool
	IncludePodSpec        bool
	IncludeOwnerRefs      bool
}

// PodMetadata represents enriched K8s pod metadata
type PodMetadata struct {
	UID           string
	Namespace     string
	Name          string
	Labels        map[string]string
	Annotations   map[string]string
	NodeName      string
	Phase         corev1.PodPhase
	PodIP         string
	HostIP        string
	StartTime     *metav1.Time
	OwnerReferences []metav1.OwnerReference

	// Optional fields
	ContainerNames []string
	InitContainers []string
}

// EnrichmentRequest specifies what to enrich
type EnrichmentRequest struct {
	PodUIDs         []string  // Unique pod identifiers to enrich
	IncludeLabels   bool
	IncludeAnnotations bool
	IncludePodSpec  bool
}

// EnrichmentResult contains enriched metadata with status
type EnrichmentResult struct {
	Metadata       map[string]*PodMetadata  // pod UID -> metadata
	FailedPodUIDs  []string                  // Pods that couldn't be enriched
	CacheHits      int                       // Number of pods served from cache
	APICalls       int                       // Number of K8s API calls made
	Duration       time.Duration             // Total enrichment duration
}

// NewMetadataEnrichmentService creates enrichment service
func NewMetadataEnrichmentService(config *rest.Config, enrichmentConfig *EnrichmentConfig) (*MetadataEnrichmentService, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	service := &MetadataEnrichmentService{
		client:  client,
		cache:   NewMetadataCache(enrichmentConfig.CacheTTL),
		config:  enrichmentConfig,
		metrics: NewEnrichmentMetrics(),
	}

	// Initialize informers for cache invalidation
	if enrichmentConfig.EnableInformer {
		service.informerFactory = informers.NewSharedInformerFactory(client, time.Minute)
		service.setupInformers()
		go service.informerFactory.Start(context.Background().Done())
	}

	return service, nil
}

func (s *MetadataEnrichmentService) setupInformers() {
	// Pod informer for cache updates
	podInformer := s.informerFactory.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				s.cache.Update(pod)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if pod, ok := newObj.(*corev1.Pod); ok {
				s.cache.Update(pod)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				s.cache.Delete(string(pod.UID))
			}
		},
	})
	s.podInformer = podInformer
	klog.InfoS("K8s pod informer started for metadata enrichment")
}

// EnrichLogs enriches log entries with K8s metadata
func (s *MetadataEnrichmentService) EnrichLogs(ctx context.Context, podUIDs []string) *EnrichmentResult {
	startTime := time.Now()
	result := &EnrichmentResult{
		Metadata: make(map[string]*PodMetadata),
	}

	// Deduplicate pod UIDs
	uniqueUIDs := s.deduplicatePodUIDs(podUIDs)
	klog.V(4).InfoS("Starting log enrichment", "unique_pods", len(uniqueUIDs))

	// Try cache first
	var cacheMisses []string
	for _, uid := range uniqueUIDs {
		if metadata := s.cache.Get(uid); metadata != nil {
			result.Metadata[uid] = metadata
			result.CacheHits++
		} else {
			cacheMisses = append(cacheMisses, uid)
		}
	}

	// Batch fetch cache misses from K8s API
	if len(cacheMisses) > 0 {
		apiResult := s.fetchBatchMetadata(ctx, cacheMisses)
		for uid, metadata := range apiResult.Metadata {
			result.Metadata[uid] = metadata
			s.cache.Set(uid, metadata)
		}
		result.FailedPodUIDs = apiResult.FailedPodUIDs
		result.APICalls = apiResult.APICalls
	}

	result.Duration = time.Since(startTime)
	s.metrics.RecordEnrichment(len(uniqueUIDs), result.CacheHits, result.APICalls, result.Duration)

	klog.V(4).InfoS("Log enrichment completed",
		"total_pods", len(uniqueUIDs),
		"cache_hits", result.CacheHits,
		"api_calls", result.APICalls,
		"failed", len(result.FailedPodUIDs),
		"duration_ms", result.Duration.Milliseconds())

	return result
}

// fetchBatchMetadata fetches pod metadata from K8s API in batches
func (s *MetadataEnrichmentService) fetchBatchMetadata(ctx context.Context, podUIDs []string) *EnrichmentResult {
	result := &EnrichmentResult{
		Metadata: make(map[string]*PodMetadata),
	}

	// Process in batches to avoid overwhelming API
	batchSize := s.config.MaxBatchSize
	if batchSize <= 0 {
		batchSize = 50 // Default batch size
	}

	for i := 0; i < len(podUIDs); i += batchSize {
		end := i + batchSize
		if end > len(podUIDs) {
			end = len(podUIDs)
		}
		batch := podUIDs[i:end]

		// Fetch pods by UID from K8s API
		for _, uid := range batch {
			pod, err := s.client.CoreV1().Pods("").Get(ctx, uid, metav1.GetOptions{
				TypeMeta: metav1.TypeMeta{
					Kind: "Pod",
				},
			})
			if err != nil {
				klog.V(4).InfoS("Failed to fetch pod metadata", "uid", uid, "error", err)
				result.FailedPodUIDs = append(result.FailedPodUIDs, uid)
				continue
			}

			result.Metadata[uid] = s.convertPodMetadata(pod)
			result.APICalls++
		}
	}

	return result
}

// convertPodMetadata converts K8s Pod to PodMetadata
func (s *MetadataEnrichmentService) convertPodMetadata(pod *corev1.Pod) *PodMetadata {
	metadata := &PodMetadata{
		UID:            string(pod.UID),
		Namespace:      pod.Namespace,
		Name:           pod.Name,
		Labels:         pod.Labels,
		Annotations:    pod.Annotations,
		NodeName:       pod.Spec.NodeName,
		Phase:          pod.Status.Phase,
		PodIP:          pod.Status.PodIP,
		HostIP:         pod.Status.HostIP,
		StartTime:      pod.Status.StartTime,
		OwnerReferences: pod.OwnerReferences,
	}

	// Extract container names
	for _, c := range pod.Spec.Containers {
		metadata.ContainerNames = append(metadata.ContainerNames, c.Name)
	}
	for _, c := range pod.Spec.InitContainers {
		metadata.InitContainers = append(metadata.InitContainers, c.Name)
	}

	return metadata
}

// deduplicatePodUIDs removes duplicates from pod UID list
func (s *MetadataEnrichmentService) deduplicatePodUIDs(podUIDs []string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, uid := range podUIDs {
		if uid != "" && !seen[uid] {
			seen[uid] = true
			unique = append(unique, uid)
		}
	}
	return unique
}
```

### Metadata Cache Implementation

```go
// pkg/service/enrichment/cache.go
package enrichment

import (
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// MetadataCache caches K8s pod metadata
type MetadataCache struct {
	mu         sync.RWMutex
	cache      map[string]*PodMetadata
	ttl        time.Duration
	lastClean  time.Time
}

// NewMetadataCache creates new metadata cache
func NewMetadataCache(ttl time.Duration) *MetadataCache {
	cache := &MetadataCache{
		cache:     make(map[string]*PodMetadata),
		ttl:       ttl,
		lastClean: time.Now(),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves metadata from cache
func (c *MetadataCache) Get(uid string) *PodMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cache[uid]
}

// Set stores metadata in cache
func (c *MetadataCache) Set(uid string, metadata *PodMetadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[uid] = metadata
	klog.V(5).InfoS("Cached pod metadata", "uid", uid, "name", metadata.Name)
}

// Update updates cached metadata from K8s watch event
func (c *MetadataCache) Update(pod *corev1.Pod) {
	c.mu.Lock()
	defer c.mu.Unlock()

	uid := string(pod.UID)
	metadata := c.convertPodToMetadata(pod)
	c.cache[uid] = metadata

	klog.V(5).InfoS("Updated pod metadata from informer", "uid", uid, "name", pod.Name)
}

// Delete removes metadata from cache
func (c *MetadataCache) Delete(uid string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, uid)
	klog.V(5).InfoS("Deleted pod metadata from cache", "uid", uid)
}

// cleanupLoop periodically cleans expired entries
func (c *MetadataCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes stale entries
func (c *MetadataCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// For informer-based cache, entries are updated/deleted by watch events
	// This cleanup handles any stale entries
	now := time.Now()
	for key, metadata := range c.cache {
		// Remove entries older than TTL (only for cached without informer updates)
		if metadata.StartTime != nil && now.Sub(metadata.StartTime.Time) > c.ttl {
			delete(c.cache, key)
		}
	}

	c.lastClean = now
	if len(c.cache) > 0 {
		klog.V(5).InfoS("Cache cleanup completed", "entries", len(c.cache))
	}
}

func (c *MetadataCache) convertPodToMetadata(pod *corev1.Pod) *PodMetadata {
	return &PodMetadata{
		UID:             string(pod.UID),
		Namespace:       pod.Namespace,
		Name:            pod.Name,
		Labels:          pod.Labels,
		Annotations:     pod.Annotations,
		NodeName:        pod.Spec.NodeName,
		Phase:           pod.Status.Phase,
		PodIP:           pod.Status.PodIP,
		HostIP:          pod.Status.HostIP,
		StartTime:       pod.Status.StartTime,
		OwnerReferences: pod.OwnerReferences,
	}
}
```

### Integration with Query Service

```go
// pkg/service/query/service.go - Add enrichment field
type QueryOptions struct {
	EnableEnrichment bool
	EnrichLabels     bool
	EnrichAnnotations bool
	EnrichPodSpec     bool
}

// Enhanced QueryLogs method
func (s *Service) QueryLogsWithEnrichment(ctx context.Context, req *request.LogQueryRequest, opts *QueryOptions) (*response.LogQueryResponse, error) {
	startTime := time.Now()

	// Get logs from ClickHouse
	logs, total, err := s.repo.QueryLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	// Prepare response
	response := &response.LogQueryResponse{
		Logs:      logs,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
		HasMore:   (req.Page+1)*req.PageSize < total,
	}

	// Apply enrichment if requested
	if opts != nil && opts.EnableEnrichment && s.enrichmentService != nil {
		podUIDs := extractPodUIDs(logs)
		if len(podUIDs) > 0 {
			enrichmentResult := s.enrichmentService.EnrichLogs(ctx, podUIDs)

			// Merge enriched metadata into response
			enrichLogs(response.Logs, enrichmentResult.Metadata)

			// Add enrichment metadata to response
			response.EnrichmentMetadata = &response.EnrichmentMetadata{
				Enabled:     true,
				CacheHits:   enrichmentResult.CacheHits,
				APICalls:    enrichmentResult.APICalls,
				DurationMs:  enrichmentResult.Duration.Milliseconds(),
				TotalPods:   len(podUIDs),
				FailedPods:  len(enrichmentResult.FailedPodUIDs),
			}
		}
	}

	duration := time.Since(startTime)
	klog.InfoS("Enriched log query completed",
		"dataset", req.Dataset,
		"returned_logs", len(logs),
		"total", total,
		"duration_ms", duration.Milliseconds())

	return response, nil
}

// enrichLogs merges K8s metadata into log entries
func enrichLogs(logs []clickhouse.LogEntry, metadata map[string]*PodMetadata) {
	for i := range logs {
		if podMeta, exists := metadata[logs[i].K8sPodUID]; exists {
			logs[i].PodLabels = podMeta.Labels
			logs[i].PodAnnotations = podMeta.Annotations
			logs[i].PodOwnerReferences = podMeta.OwnerReferences
		}
	}
}
```

### Response Model Enhancement

```go
// pkg/model/response/log.go - Add enrichment fields
type LogEntry struct {
	Timestamp       time.Time              `json:"timestamp"`
	Dataset         string                 `json:"dataset"`
	Content         string                 `json:"content"`
	Severity        string                 `json:"severity"`
	ContainerID     string                 `json:"container_id"`
	ContainerName   string                 `json:"container_name"`
	PID             string                 `json:"pid"`
	HostIP          string                 `json:"host_ip"`
	HostName        string                 `json:"host_name"`
	K8sNamespace    string                 `json:"k8s_namespace_name"`
	K8sPodName      string                 `json:"k8s_pod_name"`
	K8sPodUID       string                 `json:"k8s_pod_uid"`
	K8sNodeName     string                 `json:"k8s_node_name"`
	Tags            map[string]string      `json:"tags"`

	// Enriched fields (when requested)
	PodLabels       map[string]string      `json:"pod_labels,omitempty"`
	PodAnnotations  map[string]string      `json:"pod_annotations,omitempty"`
	PodOwnerRefs    []OwnerReference        `json:"pod_owner_refs,omitempty"`
}

// EnrichmentMetadata describes enrichment results
type EnrichmentMetadata struct {
	Enabled    bool    `json:"enabled"`
	CacheHits  int     `json:"cache_hits"`
	APICalls   int     `json:"api_calls"`
	DurationMs int64   `json:"duration_ms"`
	TotalPods  int     `json:"total_pods"`
	FailedPods int     `json:"failed_pods"`
}
```

### Handler Integration

```go
// pkg/oapis/log/v1alpha1/handler.go - Add enrich parameter
func (h *LogHandler) queryLogs(req *restful.Request, resp *restful.Response) {
	// ... existing parameter parsing ...

	// Check for enrichment flag
	enableEnrich := req.QueryParameter("enrich") == "true"

	options := &query.QueryOptions{
		EnableEnrichment: enableEnrich,
		EnrichLabels:     true,
		EnrichAnnotations: true,
	}

	// Call service with options
	response, err := h.queryService.QueryLogsWithEnrichment(req.Request.Context(), queryReq, options)
	// ... handle response ...
}
```

### Project Structure Notes

**New files to create:**
```
pkg/service/enrichment/
├── metadata_service.go      # K8s metadata enrichment service
├── cache.go                  # Metadata cache with TTL
└── metrics.go                # Enrichment performance metrics
```

**Files to modify:**
```
pkg/service/query/
├── service.go                # Add enrichment integration
└── service_test.go           # Add enrichment tests

pkg/oapis/log/v1alpha1/
├── handler.go                # Add enrich parameter support
└── handler_test.go           # Add enrichment API tests

pkg/model/response/
└── log.go                    # Add enrichment fields to LogEntry

pkg/config/
└── config.go                 # Add enrichment configuration
```

### References

- [Source: _bmad-output/epics.md#Story 3.2] - Complete user story and acceptance criteria
- [Source: _bmad-output/architecture.md#元数据关联策略] - Metadata correlation strategy with K8s API
- [Source: _bmad-output/3-1-log-aggregation-by-dimensions.md] - Previous Story 3-1 for code patterns
- [Source: pkg/service/query/service.go] - Service layer to enhance with enrichment
- [Source: pkg/oapis/log/v1alpha1/handler.go] - Handler to add enrich parameter
- [Source: pkg/model/clickhouse/log.go] - ClickHouse LogEntry model

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-20250514

### Debug Log References

Story 3-2 continues Epic 3: Advanced Query and Analytics by implementing K8s metadata enrichment service. Building on Epic 2's comprehensive filtering foundation and Story 3-1's aggregation capabilities, this story adds optional K8s API-based metadata enrichment (labels, annotations, pod specifications) with intelligent caching, graceful degradation, and informer-based cache invalidation for high-performance metadata correlation in edge computing environments.

### Completion Notes List

Story 3-2 implements K8s metadata enrichment service that provides optional per-query enrichment of log entries with additional Kubernetes metadata (pod labels, annotations, owner references, pod specifications) retrieved via client-go v0.31.2. Features include intelligent metadata caching with configurable TTL, informer-based cache invalidation for real-time updates, batch API calls for performance, graceful degradation when K8s API is unavailable, and comprehensive enrichment status reporting. The service follows architecture.md's guidance for "按需关联" (on-demand correlation) where 95% of queries use ClickHouse-stored metadata and enrichment is only invoked when explicitly requested.

### File List

Primary files to be enhanced/created:
- pkg/service/enrichment/metadata_service.go (new)
- pkg/service/enrichment/cache.go (new)
- pkg/service/enrichment/metrics.go (new)
- pkg/service/query/service.go (modify - add enrichment integration)
- pkg/service/query/service_test.go (modify - add enrichment tests)
- pkg/oapis/log/v1alpha1/handler.go (modify - add enrich parameter)
- pkg/oapis/log/v1alpha1/handler_test.go (modify - add enrichment tests)
- pkg/model/response/log.go (modify - add enrichment fields)
- pkg/config/config.go (modify - add enrichment config)
- cmd/apiserver/main.go (modify - initialize enrichment service)
