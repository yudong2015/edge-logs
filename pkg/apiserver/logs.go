package apiserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp        time.Time         `json:"timestamp"`
	Dataset          string            `json:"dataset"`
	Content          string            `json:"content"`
	Severity         string            `json:"severity"`
	ContainerID      string            `json:"container_id,omitempty"`
	ContainerName    string            `json:"container_name,omitempty"`
	PID              string            `json:"pid,omitempty"`
	HostIP           string            `json:"host_ip,omitempty"`
	HostName         string            `json:"host_name,omitempty"`
	K8sNamespace     string            `json:"k8s_namespace_name,omitempty"`
	K8sPodName       string            `json:"k8s_pod_name,omitempty"`
	K8sPodUID        string            `json:"k8s_pod_uid,omitempty"`
	K8sNodeName      string            `json:"k8s_node_name,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// FluentBitLogEntry represents the log format from Fluent Bit
type FluentBitLogEntry struct {
	Timestamp int64                  `json:"timestamp"`
	Log       string                 `json:"log"`
	Stream    string                 `json:"stream"`
	Time      string                 `json:"time"`
	Kubernetes map[string]interface{} `json:"kubernetes,omitempty"`
}

// IngestLogsRequest represents the request payload for log ingestion
type IngestLogsRequest struct {
	Logs []json.RawMessage `json:"logs,omitempty"`
	// For single log entry
	Timestamp  interface{}            `json:"timestamp,omitempty"`
	Log        string                 `json:"log,omitempty"`
	Stream     string                 `json:"stream,omitempty"`
	Time       string                 `json:"time,omitempty"`
	Kubernetes map[string]interface{} `json:"kubernetes,omitempty"`
	Tag        string                 `json:"tag,omitempty"`
}

// IngestLogsResponse represents the response for log ingestion
type IngestLogsResponse struct {
	Status   string `json:"status"`
	Ingested int    `json:"ingested"`
	Errors   int    `json:"errors,omitempty"`
	Message  string `json:"message,omitempty"`
}

// registerLogRoutes registers log-related routes
func (s *Server) registerLogRoutes() {
	ws := new(restful.WebService)
	ws.Path("/api/v1alpha1/logs")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON)

	// Log ingestion endpoint for collectors
	ws.Route(ws.POST("/ingest").To(s.ingestLogs).
		Doc("Ingest logs from collectors").
		Reads(IngestLogsRequest{}).
		Returns(http.StatusOK, "OK", IngestLogsResponse{}).
		Returns(http.StatusBadRequest, "Bad Request", IngestLogsResponse{}).
		Returns(http.StatusInternalServerError, "Internal Server Error", IngestLogsResponse{}))

	// Query logs endpoint
	ws.Route(ws.GET("/query").To(s.queryLogs).
		Doc("Query logs").
		Param(ws.QueryParameter("dataset", "Dataset to query").DataType("string")).
		Param(ws.QueryParameter("start", "Start time (RFC3339)").DataType("string")).
		Param(ws.QueryParameter("end", "End time (RFC3339)").DataType("string")).
		Param(ws.QueryParameter("limit", "Maximum number of logs to return").DataType("int")).
		Param(ws.QueryParameter("offset", "Number of logs to skip").DataType("int")).
		Returns(http.StatusOK, "OK", nil))

	s.container.Add(ws)
}

// ingestLogs handles log ingestion from collectors
func (s *Server) ingestLogs(req *restful.Request, resp *restful.Response) {
	var ingestReq IngestLogsRequest
	if err := req.ReadEntity(&ingestReq); err != nil {
		klog.ErrorS(err, "Failed to parse log ingestion request")
		resp.WriteErrorString(http.StatusBadRequest, "Invalid request payload")
		return
	}

	var logs []LogEntry
	ingested := 0
	errors := 0

	// Handle batch logs
	if len(ingestReq.Logs) > 0 {
		for _, rawLog := range ingestReq.Logs {
			entry, err := s.parseLogEntry(rawLog)
			if err != nil {
				klog.ErrorS(err, "Failed to parse log entry", "raw", string(rawLog))
				errors++
				continue
			}
			logs = append(logs, entry)
			ingested++
		}
	} else {
		// Handle single log entry
		entry, err := s.parseSingleLogEntry(&ingestReq)
		if err != nil {
			klog.ErrorS(err, "Failed to parse single log entry")
			errors++
		} else {
			logs = append(logs, entry)
			ingested++
		}
	}

	// Store logs (placeholder - implement actual storage)
	if err := s.storeLogs(logs); err != nil {
		klog.ErrorS(err, "Failed to store logs")
		resp.WriteErrorString(http.StatusInternalServerError, "Failed to store logs")
		return
	}

	klog.InfoS("Logs ingested successfully", "count", ingested, "errors", errors)

	response := IngestLogsResponse{
		Status:   "success",
		Ingested: ingested,
		Errors:   errors,
	}

	resp.WriteEntity(response)
}

// parseLogEntry parses a raw log entry from JSON
func (s *Server) parseLogEntry(rawLog json.RawMessage) (LogEntry, error) {
	var fbEntry FluentBitLogEntry
	if err := json.Unmarshal(rawLog, &fbEntry); err != nil {
		return LogEntry{}, err
	}

	return s.convertFluentBitEntry(fbEntry), nil
}

// parseSingleLogEntry parses a single log entry from the request
func (s *Server) parseSingleLogEntry(req *IngestLogsRequest) (LogEntry, error) {
	fbEntry := FluentBitLogEntry{
		Log:        req.Log,
		Stream:     req.Stream,
		Time:       req.Time,
		Kubernetes: req.Kubernetes,
	}

	// Handle timestamp
	switch ts := req.Timestamp.(type) {
	case float64:
		fbEntry.Timestamp = int64(ts)
	case int64:
		fbEntry.Timestamp = ts
	}

	return s.convertFluentBitEntry(fbEntry), nil
}

// convertFluentBitEntry converts Fluent Bit format to internal format
func (s *Server) convertFluentBitEntry(fbEntry FluentBitLogEntry) LogEntry {
	entry := LogEntry{
		Content:  fbEntry.Log,
		Severity: fbEntry.Stream, // stdout/stderr
		Dataset:  "default",
		Tags:     make(map[string]string),
	}

	// Parse timestamp
	if fbEntry.Timestamp > 0 {
		entry.Timestamp = time.Unix(fbEntry.Timestamp/1000, (fbEntry.Timestamp%1000)*1000000)
	} else if fbEntry.Time != "" {
		if ts, err := time.Parse(time.RFC3339, fbEntry.Time); err == nil {
			entry.Timestamp = ts
		} else {
			entry.Timestamp = time.Now()
		}
	} else {
		entry.Timestamp = time.Now()
	}

	// Extract Kubernetes metadata
	if fbEntry.Kubernetes != nil {
		if namespace, ok := fbEntry.Kubernetes["namespace_name"].(string); ok {
			entry.K8sNamespace = namespace
		}
		if podName, ok := fbEntry.Kubernetes["pod_name"].(string); ok {
			entry.K8sPodName = podName
		}
		if podUID, ok := fbEntry.Kubernetes["pod_id"].(string); ok {
			entry.K8sPodUID = podUID
		}
		if nodeName, ok := fbEntry.Kubernetes["host"].(string); ok {
			entry.K8sNodeName = nodeName
		}
		if containerName, ok := fbEntry.Kubernetes["container_name"].(string); ok {
			entry.ContainerName = containerName
		}
		if containerID, ok := fbEntry.Kubernetes["container_id"].(string); ok {
			entry.ContainerID = containerID
		}

		// Extract labels as tags
		if labels, ok := fbEntry.Kubernetes["labels"].(map[string]interface{}); ok {
			for k, v := range labels {
				if str, ok := v.(string); ok {
					entry.Tags[k] = str
				}
			}
		}
	}

	return entry
}

// storeLogs stores log entries (placeholder implementation)
func (s *Server) storeLogs(logs []LogEntry) error {
	// TODO: Implement actual ClickHouse storage
	klog.InfoS("Storing logs", "count", len(logs))
	for _, log := range logs {
		klog.V(2).InfoS("Log entry",
			"timestamp", log.Timestamp,
			"namespace", log.K8sNamespace,
			"pod", log.K8sPodName,
			"container", log.ContainerName,
			"content", log.Content[:min(len(log.Content), 100)],
		)
	}
	return nil
}

// queryLogs handles log queries
func (s *Server) queryLogs(req *restful.Request, resp *restful.Response) {
	dataset := req.QueryParameter("dataset")
	startTime := req.QueryParameter("start")
	endTime := req.QueryParameter("end")
	limit := req.QueryParameter("limit")
	offset := req.QueryParameter("offset")

	klog.InfoS("Query logs request",
		"dataset", dataset,
		"start", startTime,
		"end", endTime,
		"limit", limit,
		"offset", offset,
	)

	// TODO: Implement actual query logic
	response := map[string]interface{}{
		"status": "success",
		"logs":   []LogEntry{},
		"total":  0,
	}

	resp.WriteEntity(response)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}