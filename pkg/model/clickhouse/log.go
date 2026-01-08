package clickhouse

import "time"

// LogEntry represents a log entry in ClickHouse
type LogEntry struct {
	ID           string            `ch:"id"`
	Timestamp    time.Time         `ch:"timestamp"`
	Message      string            `ch:"message"`
	Level        string            `ch:"level"`
	Namespace    string            `ch:"namespace"`
	Pod          string            `ch:"pod"`
	Container    string            `ch:"container"`
	Labels       map[string]string `ch:"labels"`
	CreatedAt    time.Time         `ch:"created_at"`
	UpdatedAt    time.Time         `ch:"updated_at"`
}

// TableName returns the ClickHouse table name
func (LogEntry) TableName() string {
	return "logs"
}