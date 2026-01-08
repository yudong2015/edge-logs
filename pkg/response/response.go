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

// WriteSuccess writes a success response
func WriteSuccess(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndEntity(http.StatusOK, SuccessResponse{
		Data: data,
	})
}