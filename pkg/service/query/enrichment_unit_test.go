package query

import (
	"context"
	"testing"
	"time"

	"github.com/outpostos/edge-logs/pkg/model/clickhouse"
	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/service/enrichment"
)

func TestService_CollectPodUIDs(t *testing.T) {
	service := NewService(nil, nil)

	logs := []clickhouse.LogEntry{
		{K8sPodUID: "pod-1"},
		{K8sPodUID: "pod-2"},
		{K8sPodUID: "pod-1"},
		{K8sPodUID: ""},
		{K8sPodUID: "pod-3"},
		{K8sPodUID: "pod-2"},
	}

	podUIDs := service.collectPodUIDs(logs)

	expectedUIDs := []string{"pod-1", "pod-2", "pod-3"}
	if len(podUIDs) != len(expectedUIDs) {
		t.Errorf("Expected %d unique UIDs, got %d", len(expectedUIDs), len(podUIDs))
	}

	uidMap := make(map[string]bool)
	for _, uid := range podUIDs {
		uidMap[uid] = true
	}

	for _, expectedUID := range expectedUIDs {
		if !uidMap[expectedUID] {
			t.Errorf("Expected UID %s not found in collected UIDs", expectedUID)
		}
	}

	if uidMap[""] {
		t.Error("Empty UID should not be collected")
	}

	t.Logf("Successfully collected %d unique pod UIDs from %d logs", len(podUIDs), len(logs))
}

func TestService_TransformLogToResponse(t *testing.T) {
	service := NewService(nil, nil)

	log := clickhouse.LogEntry{
		Dataset:       "test-dataset",
		Timestamp:     time.Now(),
		Content:       "Test log message",
		Severity:      "ERROR",
		K8sNamespace:  "production",
		K8sPodName:    "api-server",
		K8sPodUID:     "api-uid-123",
		K8sNodeName:   "node-5",
		ContainerName: "backend",
		HostIP:        "10.0.0.100",
		HostName:      "worker-5",
		Tags:          map[string]string{"environment": "prod", "team": "backend"},
	}

	responseLog := service.transformLogToResponse(log)

	if responseLog.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if !responseLog.Timestamp.Equal(log.Timestamp) {
		t.Error("Timestamp mismatch")
	}

	if responseLog.Message != log.Content {
		t.Errorf("Expected message '%s', got '%s'", log.Content, responseLog.Message)
	}

	if responseLog.Level != log.Severity {
		t.Errorf("Expected level '%s', got '%s'", log.Severity, responseLog.Level)
	}

	if responseLog.Namespace != log.K8sNamespace {
		t.Errorf("Expected namespace '%s', got '%s'", log.K8sNamespace, responseLog.Namespace)
	}

	if responseLog.Pod != log.K8sPodName {
		t.Errorf("Expected pod '%s', got '%s'", log.K8sPodName, responseLog.Pod)
	}

	if responseLog.Container != log.ContainerName {
		t.Errorf("Expected container '%s', got '%s'", log.ContainerName, responseLog.Container)
	}

	if len(responseLog.Labels) != len(log.Tags)+3 {
		t.Errorf("Expected %d labels (tags + 3 enrichments), got %d", len(log.Tags)+3, len(responseLog.Labels))
	}

	if responseLog.Labels["environment"] != "prod" {
		t.Error("Expected tag 'environment=prod' to be preserved")
	}

	t.Logf("Log transformation successful: ID=%s, Labels=%d", responseLog.ID, len(responseLog.Labels))
}

func TestService_ApplyEnrichment(t *testing.T) {
	service := NewService(nil, nil)

	responseLog := service.transformLogToResponse(clickhouse.LogEntry{
		Timestamp:     time.Now(),
		Content:       "Test",
		Severity:      "INFO",
		K8sNamespace:  "default",
		K8sPodName:    "test-pod",
		ContainerName: "app",
	})

	podMetadata := &enrichment.PodMetadata{
		UID:       "test-uid-123",
		Namespace: "default",
		Name:      "test-pod",
		NodeName:  "node-1",
		PodIP:     "10.0.0.1",
		HostIP:    "192.168.1.1",
		Phase:     "Running",
		Labels: map[string]string{
			"app":     "myapp",
			"version": "v1.0",
		},
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
		},
	}

	service.applyEnrichment(&responseLog, podMetadata)

	if responseLog.PodUID != "test-uid-123" {
		t.Errorf("Expected PodUID 'test-uid-123', got '%s'", responseLog.PodUID)
	}

	if responseLog.NodeName != "node-1" {
		t.Errorf("Expected NodeName 'node-1', got '%s'", responseLog.NodeName)
	}

	if responseLog.PodIP != "10.0.0.1" {
		t.Errorf("Expected PodIP '10.0.0.1', got '%s'", responseLog.PodIP)
	}

	if responseLog.HostIP != "192.168.1.1" {
		t.Errorf("Expected HostIP '192.168.1.1', got '%s'", responseLog.HostIP)
	}

	if responseLog.PodPhase != "Running" {
		t.Errorf("Expected PodPhase 'Running', got '%s'", responseLog.PodPhase)
	}

	if len(responseLog.PodLabels) != 2 {
		t.Errorf("Expected 2 pod labels, got %d", len(responseLog.PodLabels))
	}

	if responseLog.PodLabels["app"] != "myapp" {
		t.Errorf("Expected label 'app=myapp', got '%s'", responseLog.PodLabels["app"])
	}

	if len(responseLog.PodAnnotations) != 1 {
		t.Errorf("Expected 1 annotation, got %d", len(responseLog.PodAnnotations))
	}

	t.Logf("Enrichment application successful: PodUID=%s, Labels=%d, Annotations=%d",
		responseLog.PodUID, len(responseLog.PodLabels), len(responseLog.PodAnnotations))
}

func TestService_TransformAndEnrichLogsWithoutEnrichment(t *testing.T) {
	service := NewService(nil, nil)

	logs := []clickhouse.LogEntry{
		{
			Timestamp:     time.Now(),
			Content:       "Test log 1",
			Severity:      "INFO",
			K8sNamespace:  "default",
			K8sPodName:    "test-pod",
			K8sPodUID:     "pod-uid-1",
			ContainerName: "app",
			Dataset:       "test-dataset",
		},
		{
			Timestamp:     time.Now(),
			Content:       "Test log 2",
			Severity:      "ERROR",
			K8sNamespace:  "default",
			K8sPodName:    "test-pod-2",
			K8sPodUID:     "pod-uid-2",
			ContainerName: "sidecar",
			Dataset:       "test-dataset",
		},
	}

	req := &request.LogQueryRequest{
		Dataset:       "test-dataset",
		EnrichMetadata: boolPtr(false),
	}

	responseLogs, enrichmentMeta, err := service.transformAndEnrichLogs(context.Background(), logs, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(responseLogs) != len(logs) {
		t.Errorf("Expected %d response logs, got %d", len(logs), len(responseLogs))
	}

	if enrichmentMeta != nil {
		t.Error("Expected no enrichment metadata when disabled")
	}

	for i, log := range responseLogs {
		if log.PodUID != "" {
			t.Errorf("Expected no PodUID in log %d, got '%s'", i, log.PodUID)
		}
		if len(log.PodLabels) > 0 {
			t.Errorf("Expected no PodLabels in log %d, got %d", i, len(log.PodLabels))
		}
		if len(log.PodAnnotations) > 0 {
			t.Errorf("Expected no PodAnnotations in log %d, got %d", i, len(log.PodAnnotations))
		}
	}

	t.Logf("Successfully transformed %d logs without enrichment", len(responseLogs))
}

func TestService_TransformAndEnrichLogsWithEnrichmentNoService(t *testing.T) {
	service := NewService(nil, nil) // No enrichment service

	logs := []clickhouse.LogEntry{
		{
			Timestamp:     time.Now(),
			Content:       "Test log",
			Severity:      "INFO",
			K8sNamespace:  "default",
			K8sPodName:    "test-pod",
			K8sPodUID:     "pod-uid-1",
			ContainerName: "app",
			Dataset:       "test-dataset",
		},
	}

	req := &request.LogQueryRequest{
		Dataset:       "test-dataset",
		EnrichMetadata: boolPtr(true), // Request enrichment
	}

	responseLogs, enrichmentMeta, err := service.transformAndEnrichLogs(context.Background(), logs, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(responseLogs) != len(logs) {
		t.Errorf("Expected %d response logs, got %d", len(logs), len(responseLogs))
	}

	// Should not have enrichment metadata since service is nil
	if enrichmentMeta != nil {
		t.Error("Expected no enrichment metadata when service is nil")
	}

	// Should still transform logs successfully
	for i, log := range responseLogs {
		if log.Message == "" {
			t.Errorf("Expected message in log %d", i)
		}
		if log.Namespace == "" {
			t.Errorf("Expected namespace in log %d", i)
		}
	}

	t.Logf("Successfully handled enrichment request without enrichment service")
}

func boolPtr(b bool) *bool {
	return &b
}