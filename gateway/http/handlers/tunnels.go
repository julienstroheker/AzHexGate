package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/julienstroheker/AzHexGate/gateway/tunnel"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

const (
	// defaultLocalPort is the default port used when no port is specified in the request
	// TODO: Parse localPort from request body instead of using this default
	defaultLocalPort = 3000
)

// NewTunnelsHandler creates a handler for tunnel creation requests
func NewTunnelsHandler(manager *tunnel.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve logger from context to establish the pattern for future handlers
		// Will be used for actual logging when real tunnel creation logic is implemented
		logger := logging.FromContext(r.Context())
		// Only accept POST requests
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// If manager is nil, return error
		if manager == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// TODO: Parse localPort from request body
		// For now, use default port
		localPort := defaultLocalPort

		// Create tunnel using manager
		response, err := manager.CreateTunnel(r.Context(), logger, localPort)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Marshal response to check for errors before writing status
		data, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}
