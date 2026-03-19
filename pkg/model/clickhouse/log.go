package clickhouse

import "time"

// LogEntry represents a log entry in ClickHouse using OTEL standard format (ADR-001)
// Maps to the otel_logs table structure created by OTEL Collector ClickHouse exporter
type LogEntry struct {
	// Core OTEL fields
	Timestamp      time.Time `ch:"Timestamp"`
	TimestampTime  time.Time `ch:"TimestampTime"`
	TraceID        string    `ch:"TraceId"`
	SpanID         string    `ch:"SpanId"`
	TraceFlags     uint8     `ch:"TraceFlags"`
	SeverityText   string    `ch:"SeverityText"`
	SeverityNumber uint8     `ch:"SeverityNumber"`
	ServiceName    string    `ch:"ServiceName"`
	Body           string    `ch:"Body"`

	// Resource metadata
	ResourceSchemaUrl  string            `ch:"ResourceSchemaUrl"`
	ResourceAttributes map[string]string `ch:"ResourceAttributes"`

	// Scope metadata
	ScopeSchemaUrl string            `ch:"ScopeSchemaUrl"`
	ScopeName      string            `ch:"ScopeName"`
	ScopeVersion   string            `ch:"ScopeVersion"`
	ScopeAttributes map[string]string `ch:"ScopeAttributes"`

	// Log attributes (includes K8s metadata)
	LogAttributes map[string]string `ch:"LogAttributes"`

	// K8s metadata extracted from __path__ (for backward compatibility)
	K8sPodName       string `json:"k8s_pod_name" ch:"k8s_pod_name"`
	K8sNamespaceName string `json:"k8s_namespace_name" ch:"k8s_namespace_name"`
	K8sContainerName string `json:"k8s_container_name" ch:"k8s_container_name"`
	K8sContainerID    string `json:"k8s_container_id" ch:"k8s_container_id"`
}

// TableName returns the ClickHouse table name
func (LogEntry) TableName() string {
	return "otel_logs"
}

// GetContent returns the log message body (OTEL Body field)
func (l *LogEntry) GetContent() string {
	return l.Body
}

// GetSeverity returns the severity level (OTEL SeverityText field)
func (l *LogEntry) GetSeverity() string {
	return l.SeverityText
}

// GetK8sNamespace extracts namespace from ResourceAttributes (data collection stage)
func (l *LogEntry) GetK8sNamespace() string {
	// Priority: ResourceAttributes (from transform processor) > LogAttributes
	if ns, ok := l.ResourceAttributes["k8s.namespace.name"]; ok && ns != "" {
		return ns
	}
	if ns, ok := l.LogAttributes["k8s.namespace.name"]; ok {
		return ns
	}
	return ""
}

// GetK8sPodName extracts pod name from ResourceAttributes (data collection stage)
func (l *LogEntry) GetK8sPodName() string {
	// Priority: ResourceAttributes (from transform processor) > LogAttributes
	if pod, ok := l.ResourceAttributes["k8s.pod.name"]; ok && pod != "" {
		return pod
	}
	if pod, ok := l.LogAttributes["k8s.pod.name"]; ok {
		return pod
	}
	return ""
}

// GetK8sNodeName extracts node name from LogAttributes
func (l *LogEntry) GetK8sNodeName() string {
	if node, ok := l.LogAttributes["k8s.node.name"]; ok {
		return node
	}
	return ""
}

// GetK8sContainerName extracts container name from ResourceAttributes (data collection stage)
func (l *LogEntry) GetK8sContainerName() string {
	// Priority: ResourceAttributes (from transform processor) > LogAttributes
	if container, ok := l.ResourceAttributes["k8s.container.name"]; ok && container != "" {
		return container
	}
	if container, ok := l.LogAttributes["k8s.container.name"]; ok {
		return container
	}
	return ""
}

// GetK8sContainerID extracts container ID from extracted field or LogAttributes
func (l *LogEntry) GetK8sContainerID() string {
	// Prefer extracted field from SQL query
	if l.K8sContainerID != "" {
		return l.K8sContainerID
	}
	// Fallback to LogAttributes
	if id, ok := l.LogAttributes["container.id"]; ok {
		return id
	}
	return ""
}

// GetContainerName extracts container name (alias for GetK8sContainerName)
func (l *LogEntry) GetContainerName() string {
	return l.GetK8sContainerName()
}

// GetContainerID extracts container ID from LogAttributes
func (l *LogEntry) GetContainerID() string {
	if id, ok := l.LogAttributes["container.id"]; ok {
		return id
	}
	return ""
}

// GetHostIP extracts host IP from ResourceAttributes
func (l *LogEntry) GetHostIP() string {
	if ip, ok := l.ResourceAttributes["host.ip"]; ok {
		return ip
	}
	return ""
}

// GetHostName extracts host name from ResourceAttributes
func (l *LogEntry) GetHostName() string {
	if name, ok := l.ResourceAttributes["host.name"]; ok {
		return name
	}
	return ""
}

// GetK8sPodUID extracts pod UID from LogAttributes
func (l *LogEntry) GetK8sPodUID() string {
	if uid, ok := l.LogAttributes["k8s.pod.uid"]; ok {
		return uid
	}
	return ""
}

// GetDataset returns ServiceName as dataset identifier for compatibility
func (l *LogEntry) GetDataset() string {
	return l.ServiceName
}