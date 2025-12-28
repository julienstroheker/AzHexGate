package api

// TunnelResponse represents the response from the Management API tunnel creation endpoint
type TunnelResponse struct {
	PublicURL            string `json:"public_url"`
	RelayEndpoint        string `json:"relay_endpoint"`
	HybridConnectionName string `json:"hybrid_connection_name"`
	ListenerToken        string `json:"listener_token"`
	SessionID            string `json:"session_id"`
}
