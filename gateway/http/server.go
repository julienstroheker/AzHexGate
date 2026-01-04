package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/gateway/http/handlers"
	"github.com/julienstroheker/AzHexGate/gateway/http/middleware"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// Server represents the HTTP server
type Server struct {
	server *http.Server
	port   int
	logger *logging.Logger
}

// NewServer creates a new HTTP server instance
func NewServer(port int, logger *logging.Logger) *Server {
	mux := http.NewServeMux()

	// Register health check endpoint
	mux.HandleFunc("/healthz", handlers.HealthHandler)

	// Register management API endpoints
	mux.HandleFunc("/api/tunnels", handlers.TunnelsHandler)

	// Chain middlewares: Proxy -> Telemetry -> Logger -> Metrics -> handlers
	// Proxy is first to intercept subdomain tunnel requests before other processing
	// Telemetry is second to ensure all requests get tracking IDs
	// Logger is third to log requests with telemetry IDs
	// Metrics is fourth as a placeholder for future metrics collection
	var handler http.Handler = mux
	handler = middleware.Metrics(handler)
	handler = middleware.Logger(logger)(handler)
	handler = middleware.Telemetry(handler)
	handler = handlers.ProxyMiddleware(handler)

	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		},
		port:   port,
		logger: logger,
	}
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Close immediately closes the server
func (s *Server) Close() error {
	return s.server.Close()
}

// Port returns the port the server is configured to listen on
func (s *Server) Port() int {
	return s.port
}
