package clickhouse

import "time"

// LogEntry represents a log entry in ClickHouse (mapping to Story 1-2 schema)
type LogEntry struct {
	// Time and data isolation
	Timestamp time.Time `ch:"timestamp"`
	Dataset   string    `ch:"dataset"`

	// Log content
	Content  string `ch:"content"`
	Severity string `ch:"severity"`

	// Container information
	ContainerID   string `ch:"container_id"`
	ContainerName string `ch:"container_name"`
	PID           string `ch:"pid"`

	// Host information
	HostIP   string `ch:"host_ip"`
	HostName string `ch:"host_name"`

	// K8s metadata
	K8sNamespace string `ch:"k8s_namespace_name"`
	K8sPodName   string `ch:"k8s_pod_name"`
	K8sPodUID    string `ch:"k8s_pod_uid"`
	K8sNodeName  string `ch:"k8s_node_name"`

	// Analysis dimensions tags
	Tags map[string]string `ch:"tags"`
}

// TableName returns the ClickHouse table name
func (LogEntry) TableName() string {
	return "logs"
}