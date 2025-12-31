package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/julienstroheker/AzHexGate/gateway/tunnel"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// NewListenHandler creates a handler for clients to accept relay connections
// This endpoint allows clients to establish a long-lived connection that receives
// incoming relay connections, mimicking Azure Relay's listener behavior
func NewListenHandler(manager *tunnel.Manager, logger *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract hybrid connection name from path
		// Expected format: /internal/listen/hc-abc123
		path := strings.TrimPrefix(r.URL.Path, "/internal/listen/")
		hcName := strings.TrimPrefix(path, "/")

		if hcName == "" {
			http.Error(w, "Missing hybrid connection name", http.StatusBadRequest)
			return
		}

		if logger != nil {
			logger.Info("Client connecting to listener",
				logging.String("hc_name", hcName),
				logging.String("remote_addr", r.RemoteAddr))
		}

		// Get listener for this hybrid connection
		listener, err := manager.GetListener(hcName)
		if err != nil {
			if logger != nil {
				logger.Error("Listener not found",
					logging.String("hc_name", hcName),
					logging.Error(err))
			}
			http.Error(w, fmt.Sprintf("Listener not found: %s", hcName), http.StatusNotFound)
			return
		}

		// Set headers for streaming
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Flush headers
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Create a context that's cancelled when the client disconnects
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		if logger != nil {
			logger.Info("Client listener connected, accepting connections",
				logging.String("hc_name", hcName))
		}

		// Accept connections from the listener
		for {
			// Set a timeout for Accept to check context periodically
			acceptCtx, acceptCancel := context.WithTimeout(ctx, 5*time.Second)
			conn, err := listener.Accept(acceptCtx)
			acceptCancel()

			if err != nil {
				// Check if context was cancelled (client disconnected)
				if ctx.Err() != nil {
					if logger != nil {
						logger.Info("Client listener disconnected",
							logging.String("hc_name", hcName))
					}
					return
				}

				// If it's a timeout, continue waiting
				if err == context.DeadlineExceeded {
					continue
				}

				// Other errors
				if logger != nil {
					logger.Error("Failed to accept connection",
						logging.String("hc_name", hcName),
						logging.Error(err))
				}
				http.Error(w, "Failed to accept connection", http.StatusInternalServerError)
				return
			}

			if logger != nil {
				logger.Debug("Accepted connection from listener",
					logging.String("hc_name", hcName))
			}

			// Handle the connection in a goroutine to process this request
			// and immediately return to continue the listener loop
			go handleListenerConnection(conn, w, logger)

			// In the mock implementation, each Accept() returns one connection
			// After processing one connection, we return to allow the client
			// to make a new /internal/listen request for the next connection
			return
		}
	}
}

// handleListenerConnection processes a single connection from the listener
func handleListenerConnection(conn interface{}, w http.ResponseWriter, logger *logging.Logger) {
	// The connection is passed through the HTTP response body
	// The client will read HTTP request from the response body,
	// process it, and send back the HTTP response

	// This is a simplified implementation - in production, you'd need
	// a proper bidirectional communication protocol (like WebSocket)

	// For now, we just close the connection
	// The full implementation requires a bidirectional channel
	if closer, ok := conn.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}
