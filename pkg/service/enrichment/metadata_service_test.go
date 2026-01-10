package enrichment

import (
	"testing"
	"time"
)

func TestDefaultEnrichmentConfig(t *testing.T) {
	config := DefaultEnrichmentConfig()

	if config.CacheTTL != 5*time.Minute {
		t.Errorf("Expected default CacheTTL 5m, got %v", config.CacheTTL)
	}

	if !config.EnableInformer {
		t.Error("Expected EnableInformer to be true")
	}

	if config.MaxBatchSize != 50 {
		t.Errorf("Expected default MaxBatchSize 50, got %d", config.MaxBatchSize)
	}

	if config.APITimeout != 10*time.Second {
		t.Errorf("Expected default APITimeout 10s, got %v", config.APITimeout)
	}

	if !config.IncludeLabels {
		t.Error("Expected IncludeLabels to be true")
	}

	if !config.IncludeAnnotations {
		t.Error("Expected IncludeAnnotations to be true")
	}

	if config.IncludePodSpec {
		t.Error("Expected IncludePodSpec to be false")
	}

	if !config.IncludeOwnerRefs {
		t.Error("Expected IncludeOwnerRefs to be true")
	}

	t.Logf("Default config validated: %+v", config)
}

func TestMetadataEnrichmentService_Deduplication(t *testing.T) {
	// Create a minimal service for testing
	service := &MetadataEnrichmentService{}

	inputPodUIDs := []string{
		"pod-1",
		"pod-2",
		"pod-1",
		"",
		"pod-3",
		"pod-2",
		"",
		"pod-1",
	}

	uniqueUIDs := service.deduplicatePodUIDs(inputPodUIDs)

	expectedCount := 3
	if len(uniqueUIDs) != expectedCount {
		t.Errorf("Expected %d unique UIDs, got %d", expectedCount, len(uniqueUIDs))
	}

	for _, uid := range uniqueUIDs {
		if uid == "" {
			t.Error("Empty string should not be in unique UIDs")
		}
	}

	t.Logf("Deduplication successful: input %d, unique %d", len(inputPodUIDs), len(uniqueUIDs))
}

func TestMetadataCache_BasicOperations(t *testing.T) {
	cache := NewMetadataCache(5 * time.Minute)

	// Test cache miss
	pod1 := cache.Get("pod-1")
	if pod1 != nil {
		t.Error("Expected nil for non-existent pod")
	}

	// Test cache set and get
	metadata := &PodMetadata{
		UID:        "pod-1",
		Name:       "test-pod",
		Namespace:  "default",
		NodeName:   "node-1",
		Labels:     map[string]string{"app": "test"},
		Phase:      "Running",
	}

	cache.Set("pod-1", metadata)

	retrieved := cache.Get("pod-1")
	if retrieved == nil {
		t.Error("Expected to retrieve cached pod metadata")
	}

	if retrieved.UID != "pod-1" {
		t.Errorf("Expected UID 'pod-1', got '%s'", retrieved.UID)
	}

	if retrieved.Name != "test-pod" {
		t.Errorf("Expected Name 'test-pod', got '%s'", retrieved.Name)
	}

	// Test cache delete
	cache.Delete("pod-1")
	deleted := cache.Get("pod-1")
	if deleted != nil {
		t.Error("Expected nil after deletion")
	}

	t.Log("Basic cache operations successful")
}

func TestEnrichmentMetrics_Creation(t *testing.T) {
	metrics := NewEnrichmentMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to be created")
	}

	// Test recording metrics
	metrics.RecordEnrichment(10, 5, 3, 100*time.Millisecond)

	// We can't easily test the actual metrics values without exposing them,
	// but we can at least ensure the method doesn't panic
	t.Log("Enrichment metrics created and recording works")
}

func TestEnrichmentConfig_Customization(t *testing.T) {
	config := &EnrichmentConfig{
		CacheTTL:           15 * time.Minute,
		EnableInformer:     false,
		MaxBatchSize:       100,
		APITimeout:         30 * time.Second,
		IncludeLabels:      false,
		IncludeAnnotations: false,
		IncludePodSpec:     true,
		IncludeOwnerRefs:   false,
	}

	if config.CacheTTL != 15*time.Minute {
		t.Errorf("Expected CacheTTL 15m, got %v", config.CacheTTL)
	}

	if config.EnableInformer {
		t.Error("Expected EnableInformer to be false")
	}

	if config.MaxBatchSize != 100 {
		t.Errorf("Expected MaxBatchSize 100, got %d", config.MaxBatchSize)
	}

	if config.APITimeout != 30*time.Second {
		t.Errorf("Expected APITimeout 30s, got %v", config.APITimeout)
	}

	if config.IncludeLabels {
		t.Error("Expected IncludeLabels to be false")
	}

	if config.IncludeAnnotations {
		t.Error("Expected IncludeAnnotations to be false")
	}

	if !config.IncludePodSpec {
		t.Error("Expected IncludePodSpec to be true")
	}

	if config.IncludeOwnerRefs {
		t.Error("Expected IncludeOwnerRefs to be false")
	}

	t.Log("Custom enrichment config validated")
}