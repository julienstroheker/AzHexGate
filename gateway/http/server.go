package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/gateway/http/handlers"
	"github.com/julienstroheker/AzHexGate/gateway/tunnel"
)

// Server represents the HTTP server
type Server struct {
	server  *http.Server
	port    int
	manager *tunnel.Manager
}

// Options configures the HTTP server
type Options struct {
	Port    int
	Manager *tunnel.Manager
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

	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", opts.Port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
		port:    opts.Port,
		manager: opts.Manager,
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
