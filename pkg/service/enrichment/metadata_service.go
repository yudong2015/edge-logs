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
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// MetadataEnrichmentService provides K8s metadata enrichment
type MetadataEnrichmentService struct {
	client          kubernetes.Interface
	informerFactory informers.SharedInformerFactory
	podInformer     cache.SharedIndexInformer
	cache           *MetadataCache
	config          *EnrichmentConfig
	metrics         *EnrichmentMetrics
	started         sync.Once
	startOnce       sync.Once
	stopCh          chan struct{}
}

// EnrichmentConfig configures enrichment behavior
type EnrichmentConfig struct {
	CacheTTL           time.Duration
	EnableInformer     bool
	MaxBatchSize       int
	APITimeout         time.Duration
	IncludeLabels      bool
	IncludeAnnotations bool
	IncludePodSpec     bool
	IncludeOwnerRefs   bool
}

// DefaultEnrichmentConfig returns default enrichment configuration
func DefaultEnrichmentConfig() *EnrichmentConfig {
	return &EnrichmentConfig{
		CacheTTL:           5 * time.Minute,
		EnableInformer:     true,
		MaxBatchSize:       50,
		APITimeout:         10 * time.Second,
		IncludeLabels:      true,
		IncludeAnnotations: true,
		IncludePodSpec:     false,
		IncludeOwnerRefs:   true,
	}
}

// PodMetadata represents enriched K8s pod metadata
type PodMetadata struct {
	UID             string
	Namespace       string
	Name            string
	Labels          map[string]string
	Annotations     map[string]string
	NodeName        string
	Phase           corev1.PodPhase
	PodIP           string
	HostIP          string
	StartTime       *metav1.Time
	OwnerReferences []metav1.OwnerReference

	// Optional fields
	ContainerNames []string
	InitContainers []string
}

// EnrichmentRequest specifies what to enrich
type EnrichmentRequest struct {
	PodUIDs            []string
	IncludeLabels      bool
	IncludeAnnotations bool
	IncludePodSpec     bool
}

// EnrichmentResult contains enriched metadata with status
type EnrichmentResult struct {
	Metadata      map[string]*PodMetadata
	FailedPodUIDs []string
	CacheHits     int
	APICalls      int
	Duration      time.Duration
}

// NewMetadataEnrichmentService creates enrichment service
func NewMetadataEnrichmentService(config *rest.Config, enrichmentConfig *EnrichmentConfig) (*MetadataEnrichmentService, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	if enrichmentConfig == nil {
		enrichmentConfig = DefaultEnrichmentConfig()
	}

	service := &MetadataEnrichmentService{
		client:  client,
		cache:   NewMetadataCache(enrichmentConfig.CacheTTL),
		config:  enrichmentConfig,
		metrics: NewEnrichmentMetrics(),
		stopCh:   make(chan struct{}),
	}

	// Initialize informers for cache invalidation
	if enrichmentConfig.EnableInformer {
		service.startOnce.Do(func() {
			service.informerFactory = informers.NewSharedInformerFactoryWithOptions(client, time.Minute, informers.WithNamespace(metav1.NamespaceAll))
			service.setupInformers()
			go service.informerFactory.Start(service.stopCh)
			klog.InfoS("K8s enrichment service started with informer")
		})
	} else {
		klog.InfoS("K8s enrichment service started without informer")
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
				klog.V(5).InfoS("Pod added to enrichment cache", "uid", pod.UID, "name", pod.Name)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if pod, ok := newObj.(*corev1.Pod); ok {
				s.cache.Update(pod)
				klog.V(5).InfoS("Pod updated in enrichment cache", "uid", pod.UID, "name", pod.Name)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				s.cache.Delete(string(pod.UID))
				klog.V(5).InfoS("Pod deleted from enrichment cache", "uid", pod.UID, "name", pod.Name)
			}
		},
	})
	s.podInformer = podInformer
}

// Shutdown gracefully stops the enrichment service
func (s *MetadataEnrichmentService) Shutdown() {
	close(s.stopCh)
	if s.informerFactory != nil {
		s.informerFactory.Shutdown()
	}
	klog.InfoS("K8s enrichment service stopped")
}

// EnrichLogs enriches log entries with K8s metadata
func (s *MetadataEnrichmentService) EnrichLogs(ctx context.Context, podUIDs []string) *EnrichmentResult {
	startTime := time.Now()
	result := &EnrichmentResult{
		Metadata: make(map[string]*PodMetadata),
	}

	// Deduplicate pod UIDs
	uniqueUIDs := s.deduplicatePodUIDs(podUIDs)
	if len(uniqueUIDs) == 0 {
		result.Duration = time.Since(startTime)
		return result
	}

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
		ctx, cancel := context.WithTimeout(ctx, s.config.APITimeout)
		defer cancel()

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

// fetchBatchMetadata fetches pod metadata from K8s API
func (s *MetadataEnrichmentService) fetchBatchMetadata(ctx context.Context, podUIDs []string) *EnrichmentResult {
	result := &EnrichmentResult{
		Metadata: make(map[string]*PodMetadata),
	}

	batchSize := s.config.MaxBatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	// Since we can't query pods by UID directly across all namespaces efficiently,
	// we'll query each namespace that has pods from our UIDs
	// For now, use a simpler approach - try to get each pod by UID using field selector
	for _, uid := range podUIDs {
		podList, err := s.client.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.uid=%s", uid),
			Limit:         1,
		})
		if err != nil {
			klog.V(5).InfoS("Failed to fetch pod metadata", "uid", uid, "error", err)
			result.FailedPodUIDs = append(result.FailedPodUIDs, uid)
			continue
		}

		if len(podList.Items) > 0 {
			pod := &podList.Items[0]
			result.Metadata[uid] = s.convertPodMetadata(pod)
			result.APICalls++
		} else {
			result.FailedPodUIDs = append(result.FailedPodUIDs, uid)
		}
	}

	return result
}

// convertPodMetadata converts K8s Pod to PodMetadata
func (s *MetadataEnrichmentService) convertPodMetadata(pod *corev1.Pod) *PodMetadata {
	metadata := &PodMetadata{
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
