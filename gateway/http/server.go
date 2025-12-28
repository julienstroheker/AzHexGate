package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/gateway/http/handlers"
)

// Server represents the HTTP server
type Server struct {
	server *http.Server
	port   int
}

// NewServer creates a new HTTP server instance
func NewServer(port int) *Server {
	mux := http.NewServeMux()

	// Register health check endpoint
	mux.HandleFunc("/healthz", handlers.HealthHandler)

	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
		port: port,
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
