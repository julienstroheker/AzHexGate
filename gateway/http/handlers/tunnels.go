package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// TunnelResponse represents the response from the tunnel creation endpoint
type TunnelResponse struct {
	PublicURL            string `json:"public_url"`
	RelayEndpoint        string `json:"relay_endpoint"`
	HybridConnectionName string `json:"hybrid_connection_name"`
	ListenerToken        string `json:"listener_token"`
	SessionID            string `json:"session_id"`
}

// TunnelsHandler handles POST requests to create new tunnels
// This is a mock implementation that returns static data
func TunnelsHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve logger from context to establish the pattern
	_ = logging.FromContext(r.Context())

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Return mock tunnel data
	response := TunnelResponse{
		PublicURL:            "https://63873749.azhexgate.com",
		RelayEndpoint:        "https://azhexgate-relay.servicebus.windows.net",
		HybridConnectionName: "hc-63873749",
		ListenerToken:        "mock-listener-token",
		SessionID:            "mock-session-id",
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
