package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/httpclient"
)

func TestNewClient(t *testing.T) {
	client := NewClient(nil)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.baseURL == "" {
		t.Error("Expected baseURL to be set")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	opts := &Options{
		BaseURL: "http://test.example.com",
	}

	client := NewClient(opts)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.baseURL != opts.BaseURL {
		t.Errorf("Expected baseURL '%s', got '%s'", opts.BaseURL, client.baseURL)
	}
}

func TestCreateTunnelSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/tunnels" {
			t.Errorf("Expected path /api/tunnels, got %s", r.URL.Path)
		}

		// Return mock response
		response := api.TunnelResponse{
			PublicURL:            "https://test123.azhexgate.com",
			RelayEndpoint:        "https://test-relay.servicebus.windows.net",
			HybridConnectionName: "hc-test123",
			ListenerToken:        "test-token",
			SessionID:            "test-session",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	opts := &Options{
		BaseURL: server.URL,
	}
	client := NewClient(opts)

	ctx := context.Background()
	resp, err := client.CreateTunnel(ctx, 3000)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp.PublicURL != "https://test123.azhexgate.com" {
		t.Errorf("Expected PublicURL 'https://test123.azhexgate.com', got '%s'", resp.PublicURL)
	}
}

func TestCreateTunnelHTTPError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	opts := &Options{
		BaseURL:    server.URL,
		MaxRetries: 0, // Disable retries for this test
	}
	client := NewClient(opts)

	ctx := context.Background()
	_, err := client.CreateTunnel(ctx, 3000)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "API returned status 500") {
		t.Errorf("Expected error about status 500, got: %v", err)
	}
}

func TestCreateTunnelInvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	opts := &Options{
		BaseURL: server.URL,
	}
	client := NewClient(opts)

	ctx := context.Background()
	_, err := client.CreateTunnel(ctx, 3000)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("Expected error about decoding response, got: %v", err)
	}
}

func TestCreateTunnelWithMockTransport(t *testing.T) {
	// Create a mock transport
	mockTransport := &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			response := api.TunnelResponse{
				PublicURL:            "https://mocked.azhexgate.com",
				RelayEndpoint:        "https://mocked-relay.servicebus.windows.net",
				HybridConnectionName: "hc-mocked",
				ListenerToken:        "mocked-token",
				SessionID:            "mocked-session",
			}

			body, _ := json.Marshal(response)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}

	// Create HTTP client with mock transport
	httpOpts := &httpclient.Options{
		Transport: mockTransport,
	}
	httpClient := httpclient.NewClient(httpOpts)

	// Create API client with custom HTTP client
	apiClient := &Client{
		baseURL:    "http://example.com",
		httpClient: httpClient,
	}

	ctx := context.Background()
	resp, err := apiClient.CreateTunnel(ctx, 3000)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp.PublicURL != "https://mocked.azhexgate.com" {
		t.Errorf("Expected PublicURL 'https://mocked.azhexgate.com', got '%s'", resp.PublicURL)
	}
}

// MockTransport is a mock HTTP transport for testing
type MockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

// RoundTrip implements http.RoundTripper
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}
