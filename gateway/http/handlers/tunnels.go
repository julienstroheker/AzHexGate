package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// TunnelService defines the interface for tunnel creation
type TunnelService interface {
	CreateTunnel(localPort int) (*api.TunnelResponse, error)
}

// tunnelService is a package-level variable that can be set by the gateway
var tunnelService TunnelService

// SetTunnelService sets the tunnel service for the handler
func SetTunnelService(service TunnelService) {
	tunnelService = service
}

// TunnelsHandler handles POST requests to create new tunnels
func TunnelsHandler(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// If no service is configured, return mock data for backward compatibility
	if tunnelService == nil {
		returnMockResponse(w)
		return
	}

	// Parse request body
	var req struct {
		LocalPort int `json:"local_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if logger != nil {
			logger.Error("Failed to decode request", logging.Error(err))
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Create tunnel using service
	response, err := tunnelService.CreateTunnel(req.LocalPort)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to create tunnel", logging.Error(err))
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Marshal response
	data, err := json.Marshal(response)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to marshal response", logging.Error(err))
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// returnMockResponse returns mock tunnel data for backward compatibility
func returnMockResponse(w http.ResponseWriter) {
	response := api.TunnelResponse{
		PublicURL:            "https://63873749.azhexgate.com",
		RelayEndpoint:        "https://azhexgate-relay.servicebus.windows.net",
		HybridConnectionName: "hc-63873749",
		ListenerToken:        "mock-listener-token",
		SessionID:            "mock-session-id",
	}

	data, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
