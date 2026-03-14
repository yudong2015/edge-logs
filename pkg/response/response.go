package response

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/outpostos/edge-logs/pkg/model/response"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// WriteError writes an error response
func WriteError(resp *restful.Response, code int, message string) {
	resp.WriteHeaderAndEntity(code, ErrorResponse{
		Message: message,
		Code:    code,
	})
}

// LogQueryResponse is an alias for model.response.LogQueryResponse
type LogQueryResponse = response.LogQueryResponse

// HealthResponse represents health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Service string `json:"service"`
}

// DatasetsResponse represents datasets list response
type DatasetsResponse struct {
	Datasets []string `json:"datasets"`
	Count    int      `json:"count"`
}