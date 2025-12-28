package management

import (
	"encoding/json"
	"net/http"
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	// Ignore encoding errors as status code is already written
	_ = json.NewEncoder(w).Encode(response)
}
