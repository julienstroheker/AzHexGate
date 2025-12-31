package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/gateway/http/handlers"
	"github.com/julienstroheker/AzHexGate/gateway/http/middleware"
	"github.com/julienstroheker/AzHexGate/gateway/tunnel"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// Server represents the HTTP server
type Server struct {
	server  *http.Server
	port    int
	manager *tunnel.Manager
	logger  *logging.Logger
}

// Options configures the HTTP server
type Options struct {
	Port    int
	Manager *tunnel.Manager
	Logger  *logging.Logger
}

// NewServer creates a new HTTP server instance
func NewServer(opts *Options) *Server {
	if opts == nil {
		opts = &Options{
			Port: 8080,
		}
	}

	mux := http.NewServeMux()

	// Register health check endpoint
	mux.HandleFunc("/healthz", handlers.HealthHandler)

	// Register management API endpoints
	mux.HandleFunc("/api/tunnels", handlers.NewTunnelsHandler(opts.Manager))

	// Chain middlewares: Telemetry -> Logger -> Metrics -> handlers
	// Telemetry is first to ensure all requests get tracking IDs
	// Logger is second to log requests with telemetry IDs
	// Metrics is third as a placeholder for future metrics collection
	var handler http.Handler = mux
	handler = middleware.Metrics(handler)
	handler = middleware.Logger(opts.Logger)(handler)
	handler = middleware.Telemetry(handler)

	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", opts.Port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
		port:    opts.Port,
		manager: opts.Manager,
		logger:  opts.Logger,
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
