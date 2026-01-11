package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/config"
)

// Server represents the API server
type Server struct {
	config     *config.Config
	container  *restful.Container
	httpServer *http.Server
}

// New creates a new API server instance
func New(cfg *config.Config) (*Server, error) {
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
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:      container,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
	}

	// Register log routes
	server.registerLogRoutes()

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

// Build-time variable (set via ldflags)
var version = "v0.1.0-dev"

// getVersion returns the version from build-time variables or a default
func getVersion() string {
	return version
}