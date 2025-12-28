package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTunnelsHandlerPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tunnels", nil)
	w := httptest.NewRecorder()

	TunnelsHandler(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Parse response body
	var response TunnelResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	// Verify response fields
	if response.PublicURL == "" {
		t.Error("Expected non-empty public_url")
	}
	if response.RelayEndpoint == "" {
		t.Error("Expected non-empty relay_endpoint")
	}
	if response.HybridConnectionName == "" {
		t.Error("Expected non-empty hybrid_connection_name")
	}
	if response.ListenerToken == "" {
		t.Error("Expected non-empty listener_token")
	}
	if response.SessionID == "" {
		t.Error("Expected non-empty session_id")
	}
}

func TestTunnelsHandlerGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tunnels", nil)
	w := httptest.NewRecorder()

	TunnelsHandler(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

func TestTunnelsHandlerPut(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/tunnels", nil)
	w := httptest.NewRecorder()

	TunnelsHandler(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

func TestTunnelsHandlerDelete(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/tunnels", nil)
	w := httptest.NewRecorder()

	TunnelsHandler(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

func TestTunnelsHandlerResponseFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tunnels", nil)
	w := httptest.NewRecorder()

	TunnelsHandler(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	// Verify JSON is valid
	var response TunnelResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	// Verify mock values match expected static data
	expectedURL := "https://63873749.azhexgate.com"
	if response.PublicURL != expectedURL {
		t.Errorf("Expected public_url '%s', got '%s'", expectedURL, response.PublicURL)
	}

	expectedRelay := "https://azhexgate-relay.servicebus.windows.net"
	if response.RelayEndpoint != expectedRelay {
		t.Errorf("Expected relay_endpoint '%s', got '%s'", expectedRelay, response.RelayEndpoint)
	}

	expectedHC := "hc-63873749"
	if response.HybridConnectionName != expectedHC {
		t.Errorf("Expected hybrid_connection_name '%s', got '%s'", expectedHC, response.HybridConnectionName)
	}

	expectedToken := "mock-listener-token"
	if response.ListenerToken != expectedToken {
		t.Errorf("Expected listener_token '%s', got '%s'", expectedToken, response.ListenerToken)
	}

	expectedSessionID := "mock-session-id"
	if response.SessionID != expectedSessionID {
		t.Errorf("Expected session_id '%s', got '%s'", expectedSessionID, response.SessionID)
	}
}
