package middleware

import (
	"github.com/emicklei/go-restful/v3"
)

// RequestInfo stores information about incoming requests
type RequestInfo struct {
	Path     string
	Method   string
	UserInfo string
}

// RequestInfoMiddleware extracts and stores request information
func RequestInfoMiddleware(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	// TODO: Implement request information extraction
	// This is a placeholder for future request info middleware
	chain.ProcessFilter(req, resp)
}