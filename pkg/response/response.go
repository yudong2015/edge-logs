package response

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// WriteError writes an error response
func WriteError(resp *restful.Response, code int, message string) {
	resp.WriteHeaderAndEntity(code, ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}

// LogQueryResponse represents a log query API response
type LogQueryResponse struct {
	Logs       []LogEntry `json:"logs"`
	TotalCount int64      `json:"total_count"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	HasMore    bool       `json:"has_more"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp   string            `json:"timestamp"`
	Level       string            `json:"level"`
	Message     string            `json:"message"`
	Namespace   string            `json:"namespace,omitempty"`
	Pod         string            `json:"pod,omitempty"`
	Container   string            `json:"container,omitempty"`
	HostIP      string            `json:"host_ip,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// WriteSuccess writes a success response
func WriteSuccess(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndEntity(http.StatusOK, SuccessResponse{
		Data: data,
	})
}