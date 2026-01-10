package enrichment

import (
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// MetadataCache caches K8s pod metadata
type MetadataCache struct {
	mu        sync.RWMutex
	cache     map[string]*PodMetadata
	ttl       time.Duration
	lastClean time.Time
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

	now := time.Now()
	for key, metadata := range c.cache {
		// Remove entries older than TTL
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
