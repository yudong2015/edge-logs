package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"
)

// LogEntry represents a single log entry for query response
type LogEntry struct {
	Timestamp     time.Time         `json:"timestamp"`
	Body          string            `json:"body"`
	Severity      string            `json:"severity,omitempty"`
	ServiceName   string            `json:"service_name,omitempty"`
	TraceID       string            `json:"trace_id,omitempty"`
	SpanID        string            `json:"span_id,omitempty"`
	K8sNamespace  string            `json:"k8s_namespace,omitempty"`
	K8sPodName    string            `json:"k8s_pod_name,omitempty"`
	K8sNodeName   string            `json:"k8s_node_name,omitempty"`
	ContainerName string            `json:"container_name,omitempty"`
	HostName      string            `json:"host_name,omitempty"`
	HostIP        string            `json:"host_ip,omitempty"`
	LogFilePath   string            `json:"log_file_path,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// LogQueryResponse represents the response for log queries
type LogQueryResponse struct {
	Items      []LogEntry `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"total_pages"`
}

// registerLogRoutes registers log query routes
func (s *Server) registerLogRoutes() {
	ws := new(restful.WebService)
	ws.Path("/api/v1alpha1/logs")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON)

	// Query logs endpoint
	ws.Route(ws.GET("/query").To(s.queryLogs).
		Doc("Query logs from ClickHouse").
		Param(ws.QueryParameter("start", "Start time (RFC3339 or relative like -1h, -30m)").DataType("string")).
		Param(ws.QueryParameter("end", "End time (RFC3339 or relative)").DataType("string")).
		Param(ws.QueryParameter("namespace", "Filter by K8s namespace (from log path)").DataType("string")).
		Param(ws.QueryParameter("pod", "Filter by K8s pod name (from log path)").DataType("string")).
		Param(ws.QueryParameter("container", "Filter by container name").DataType("string")).
		Param(ws.QueryParameter("node", "Filter by node/host name").DataType("string")).
		Param(ws.QueryParameter("filter", "Log content filter (substring match)").DataType("string")).
		Param(ws.QueryParameter("severity", "Filter by severity level").DataType("string")).
		Param(ws.QueryParameter("limit", "Maximum number of logs to return (default 100, max 1000)").DataType("int")).
		Param(ws.QueryParameter("offset", "Number of logs to skip").DataType("int")).
		Returns(http.StatusOK, "OK", LogQueryResponse{}))

	s.container.Add(ws)
	klog.InfoS("Log query routes registered", "path", "/api/v1alpha1/logs/query")
}

// queryLogs handles log queries from ClickHouse otel_logs table
func (s *Server) queryLogs(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()

	// Parse parameters
	startStr := req.QueryParameter("start")
	endStr := req.QueryParameter("end")
	namespace := req.QueryParameter("namespace")
	pod := req.QueryParameter("pod")
	container := req.QueryParameter("container")
	node := req.QueryParameter("node")
	filter := req.QueryParameter("filter")
	severity := req.QueryParameter("severity")
	limitStr := req.QueryParameter("limit")
	offsetStr := req.QueryParameter("offset")

	// Parse limit and offset
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse time range
	startTime, endTime := s.parseTimeRange(startStr, endStr)

	klog.InfoS("Query logs request",
		"start", startTime,
		"end", endTime,
		"namespace", namespace,
		"pod", pod,
		"container", container,
		"node", node,
		"filter", filter,
		"severity", severity,
		"limit", limit,
		"offset", offset,
	)

	// Build and execute query
	items, total, err := s.executeLogQuery(ctx, startTime, endTime, namespace, pod, container, node, filter, severity, limit, offset)
	if err != nil {
		klog.ErrorS(err, "Failed to query logs")
		resp.WriteHeaderAndEntity(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Query failed: %v", err),
		})
		return
	}

	// Calculate pagination
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	page := (offset / limit) + 1

	response := LogQueryResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	klog.InfoS("Query logs completed",
		"returned", len(items),
		"total", total,
		"page", page,
		"total_pages", totalPages,
	)

	resp.WriteEntity(response)
}

// parseTimeRange parses start and end time strings
func (s *Server) parseTimeRange(startStr, endStr string) (time.Time, time.Time) {
	now := time.Now().UTC()
	var startTime, endTime time.Time

	// Default: last 1 hour
	endTime = now
	startTime = now.Add(-1 * time.Hour)

	// Parse start time
	if startStr != "" {
		if t, err := s.parseTime(startStr); err == nil {
			startTime = t
		}
	}

	// Parse end time
	if endStr != "" {
		if t, err := s.parseTime(endStr); err == nil {
			endTime = t
		}
	}

	return startTime, endTime
}

// parseTime parses a time string (RFC3339 or relative like -1h, -30m)
func (s *Server) parseTime(str string) (time.Time, error) {
	now := time.Now().UTC()

	// Check for relative time format
	if strings.HasPrefix(str, "-") {
		duration, err := time.ParseDuration(str)
		if err == nil {
			return now.Add(duration), nil
		}
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, str); err == nil {
		return t, nil
	}

	// Try RFC3339Nano
	if t, err := time.Parse(time.RFC3339Nano, str); err == nil {
		return t, nil
	}

	// Try simple datetime format
	if t, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s", str)
}

// executeLogQuery executes the actual ClickHouse query
func (s *Server) executeLogQuery(ctx context.Context, startTime, endTime time.Time, namespace, pod, container, node, filter, severity string, limit, offset int) ([]LogEntry, int64, error) {
	// Check repository
	if s.repo == nil {
		return nil, 0, fmt.Errorf("ClickHouse repository not initialized")
	}

	// Get database connection
	db := s.repo.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database connection not available")
	}

	// Build WHERE clause
	var conditions []string
	var args []interface{}

	// Time range filter
	conditions = append(conditions, "Timestamp >= ? AND Timestamp <= ?")
	args = append(args, startTime, endTime)

	// Namespace filter (extract from log path: /var/log/containers/podname_namespace_container-xxx.log)
	if namespace != "" {
		conditions = append(conditions, "LogAttributes['log.file.path'] LIKE ?")
		args = append(args, fmt.Sprintf("%%_%s_%%", namespace))
	}

	// Pod filter (extract from log path)
	if pod != "" {
		conditions = append(conditions, "LogAttributes['log.file.path'] LIKE ?")
		args = append(args, fmt.Sprintf("%%/%s_%%", pod))
	}

	// Container filter
	if container != "" {
		conditions = append(conditions, "LogAttributes['log.file.path'] LIKE ?")
		args = append(args, fmt.Sprintf("%%_%s-%%", container))
	}

	// Node/host filter
	if node != "" {
		conditions = append(conditions, "(LogAttributes['host.name'] = ? OR ResourceAttributes['__hostname__'] = ?)")
		args = append(args, node, node)
	}

	// Content filter
	if filter != "" {
		conditions = append(conditions, "Body LIKE ?")
		args = append(args, fmt.Sprintf("%%%s%%", filter))
	}

	// Severity filter
	if severity != "" {
		conditions = append(conditions, "SeverityText = ?")
		args = append(args, severity)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT count() FROM otel_logs WHERE %s", whereClause)
	var total int64
	if err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT
			Timestamp,
			Body,
			SeverityText,
			ServiceName,
			TraceId,
			SpanId,
			LogAttributes,
			ResourceAttributes
		FROM otel_logs
		WHERE %s
		ORDER BY Timestamp DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var items []LogEntry
	for rows.Next() {
		var entry LogEntry
		var timestamp time.Time
		var body, severityText, serviceName, traceID, spanID string
		var logAttrs, resourceAttrs map[string]string

		if err := rows.Scan(&timestamp, &body, &severityText, &serviceName, &traceID, &spanID, &logAttrs, &resourceAttrs); err != nil {
			klog.ErrorS(err, "Failed to scan row")
			continue
		}

		entry.Timestamp = timestamp
		entry.Body = body
		entry.Severity = severityText
		entry.ServiceName = serviceName
		entry.TraceID = traceID
		entry.SpanID = spanID

		// Extract K8s metadata from log path
		// Format: /var/log/containers/podname_namespace_containername-containerid.log
		if logPath, ok := logAttrs["log.file.path"]; ok {
			entry.LogFilePath = logPath
			entry.K8sNamespace, entry.K8sPodName, entry.ContainerName = parseLogPath(logPath)
		}

		// Extract host info
		if hostName, ok := logAttrs["host.name"]; ok {
			entry.HostName = hostName
		} else if hostName, ok := resourceAttrs["__hostname__"]; ok {
			entry.HostName = hostName
		}

		if hostIP, ok := logAttrs["host.ip"]; ok {
			entry.HostIP = hostIP
		} else if hostIP, ok := resourceAttrs["source"]; ok {
			entry.HostIP = hostIP
		}

		// Merge attributes for additional info
		entry.Attributes = make(map[string]string)
		for k, v := range resourceAttrs {
			if !strings.HasPrefix(k, "__") { // Skip internal attributes
				entry.Attributes[k] = v
			}
		}
		for k, v := range logAttrs {
			entry.Attributes[k] = v
		}

		items = append(items, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration failed: %w", err)
	}

	return items, total, nil
}

// parseLogPath extracts namespace, pod name, and container name from K8s log path
// Format: /var/log/containers/podname_namespace_containername-containerid.log
func parseLogPath(path string) (namespace, podName, containerName string) {
	// Extract filename from path
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return
	}
	filename := parts[len(parts)-1]

	// Remove .log extension
	filename = strings.TrimSuffix(filename, ".log")

	// Split by underscore: podname_namespace_containername-containerid
	segments := strings.Split(filename, "_")
	if len(segments) >= 3 {
		podName = segments[0]
		namespace = segments[1]
		// Container part: containername-containerid
		containerPart := segments[2]
		// Find last dash to separate container name from container ID
		lastDash := strings.LastIndex(containerPart, "-")
		if lastDash > 0 {
			containerName = containerPart[:lastDash]
		} else {
			containerName = containerPart
		}
	}

	return
}
