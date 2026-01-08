package v1alpha1

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/response"
	"github.com/outpostos/edge-logs/pkg/service/query"
)

// LogHandler handles log API requests
type LogHandler struct {
	queryService *query.Service
}

// NewLogHandler creates a new log handler
func NewLogHandler(queryService *query.Service) *LogHandler {
	return &LogHandler{
		queryService: queryService,
	}
}

// InstallHandler installs log API routes
func (h *LogHandler) InstallHandler(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/api/v1alpha1/logs")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").To(h.queryLogs).
		Doc("Query logs").
		Param(ws.QueryParameter("query", "Log query string").DataType("string")).
		Param(ws.QueryParameter("namespace", "Kubernetes namespace").DataType("string")).
		Param(ws.QueryParameter("pod", "Pod name").DataType("string")).
		Param(ws.QueryParameter("container", "Container name").DataType("string")).
		Param(ws.QueryParameter("page", "Page number").DataType("integer")).
		Param(ws.QueryParameter("page_size", "Page size").DataType("integer")).
		Returns(http.StatusOK, "OK", response.LogQueryResponse{}))

	container.Add(ws)
}

// queryLogs handles log query requests
func (h *LogHandler) queryLogs(req *restful.Request, resp *restful.Response) {
	// Parse query parameters
	queryReq := &request.LogQueryRequest{
		Query:     req.QueryParameter("query"),
		Namespace: req.QueryParameter("namespace"),
		Pod:       req.QueryParameter("pod"),
		Container: req.QueryParameter("container"),
		// TODO: Parse page, page_size, start_time, end_time
	}

	// Query logs
	result, err := h.queryService.QueryLogs(req.Request.Context(), queryReq)
	if err != nil {
		response.WriteError(resp, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteSuccess(resp, result)
}