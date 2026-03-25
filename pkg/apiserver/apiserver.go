package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/config"
	logv1alpha1 "github.com/outpostos/edge-logs/pkg/oapis/log/v1alpha1"
	"github.com/outpostos/edge-logs/pkg/repository/clickhouse"
	"github.com/outpostos/edge-logs/pkg/service/query"
)

// Server represents the API server
type Server struct {
	config     *config.Config
	container  *restful.Container
	httpServer *http.Server
	repo       *clickhouse.ClickHouseRepository
}

// New creates a new API server instance
func New(cfg *config.Config) (*Server, error) {
	// Initialize ClickHouse repository
	repo, err := clickhouse.NewClickHouseRepository(&cfg.ClickHouse)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClickHouse repository: %w", err)
	}
	klog.InfoS("ClickHouse repository initialized",
		"host", cfg.ClickHouse.Host,
		"database", cfg.ClickHouse.Database)

	container := restful.NewContainer()

	// Add CORS filter
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept", "Authorization"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CookiesAllowed: false,
		Container:      container,
	}
	container.Filter(cors.Filter)

	// Add logging filter
	container.Filter(restful.NoBrowserCacheFilter)
	container.Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		start := time.Now()
		chain.ProcessFilter(req, resp)
		klog.InfoS("Request processed",
			"method", req.Request.Method,
			"path", req.Request.URL.Path,
			"status", resp.StatusCode(),
			"duration", time.Since(start),
		)
	})

	// Register basic health check
	ws := new(restful.WebService)
	ws.Path("/api/v1alpha1")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/health").To(healthCheck).
		Doc("Health check endpoint").
		Returns(http.StatusOK, "OK", nil))

	container.Add(ws)

	server := &Server{
		config:    cfg,
		container: container,
		repo:      repo,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:      container,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
	}

	// Register log routes (legacy API)
	server.registerLogRoutes()

	// Register K8s-style API routes (/apis/log.theriseunion.io/v1alpha1/*)
	// Note: enrichment service is optional, pass nil if not needed
	queryService := query.NewService(repo, nil)
	logHandler := logv1alpha1.NewLogHandler(queryService)
	logHandler.InstallHandler(container)

	// Register OpenAPI endpoint for API aggregation
	server.registerOpenAPIRoutes()

	return server, nil
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	klog.InfoS("Starting HTTP server", "address", s.httpServer.Addr)

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		klog.InfoS("Shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	}
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func healthCheck(req *restful.Request, resp *restful.Response) {
	resp.WriteHeader(http.StatusOK)
	resp.WriteEntity(map[string]string{
		"status":    "healthy",
		"service":   "edge-logs-apiserver",
		"version":   getVersion(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// registerOpenAPIRoutes registers OpenAPI endpoint for K8s API aggregation
func (s *Server) registerOpenAPIRoutes() {
	ws := new(restful.WebService)
	ws.Path("/openapi")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON)

	// OpenAPI v2 endpoint for K8s API aggregation
	ws.Route(ws.GET("/v2").To(s.openAPIV2Handler).
		Doc("OpenAPI v2 specification").
		Returns(http.StatusOK, "OK", nil))

	s.container.Add(ws)
	klog.InfoS("OpenAPI routes registered", "path", "/openapi/v2")
}

// openAPIV2Handler returns OpenAPI v2 specification
func (s *Server) openAPIV2Handler(req *restful.Request, resp *restful.Response) {
	openAPISpec := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]string{
			"title":       "Edge Logs API",
			"description": "Edge computing log management API",
			"version":     getVersion(),
		},
		"basePath": "/apis/log.theriseunion.io/v1alpha1",
		"paths": map[string]interface{}{
			"/datasets/{dataset}/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Query logs by dataset",
					"operationId": "queryLogsByDataset",
					"parameters": []map[string]interface{}{
						{"name": "dataset", "in": "path", "required": true, "type": "string"},
						{"name": "start_time", "in": "query", "type": "string"},
						{"name": "end_time", "in": "query", "type": "string"},
						{"name": "namespace", "in": "query", "type": "string"},
						{"name": "pod_name", "in": "query", "type": "string"},
						{"name": "severity", "in": "query", "type": "string"},
						{"name": "filter", "in": "query", "type": "string"},
						{"name": "page", "in": "query", "type": "integer"},
						{"name": "page_size", "in": "query", "type": "integer"},
					},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Successful query"},
						"400": map[string]string{"description": "Bad request"},
						"404": map[string]string{"description": "Dataset not found"},
						"500": map[string]string{"description": "Internal server error"},
					},
				},
			},
			"/datasets/{dataset}/aggregation": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Aggregate logs by dataset",
					"operationId": "aggregateLogsByDataset",
					"parameters": []map[string]interface{}{
						{"name": "dataset", "in": "path", "required": true, "type": "string"},
						{"name": "dimensions", "in": "query", "required": true, "type": "string"},
						{"name": "start_time", "in": "query", "type": "string"},
						{"name": "end_time", "in": "query", "type": "string"},
					},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Successful aggregation"},
						"400": map[string]string{"description": "Bad request"},
						"500": map[string]string{"description": "Internal server error"},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"operationId": "healthCheck",
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Service healthy"},
					},
				},
			},
		},
	}

	resp.WriteEntity(openAPISpec)
}

// Build-time variable (set via ldflags)
var version = "v0.1.0-dev"

// getVersion returns the version from build-time variables or a default
func getVersion() string {
	return version
}