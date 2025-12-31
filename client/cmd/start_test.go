package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienstroheker/AzHexGate/internal/api"
)

func TestStartCommandWithMockAPI(t *testing.T) {
	// Create mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify request path
		if r.URL.Path != "/api/tunnels" {
			t.Errorf("Expected path /api/tunnels, got %s", r.URL.Path)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request ID header is present
		if r.Header.Get("X-Client-Request-Id") == "" {
			t.Error("Expected X-Client-Request-Id header to be set")
		}

		// Return mock response
		response := api.TunnelResponse{
			PublicURL:            "https://mock123.azhexgate.com",
			RelayEndpoint:        "https://mock-relay.servicebus.windows.net",
			HybridConnectionName: "hc-mock123",
			ListenerToken:        "mock-token",
			SessionID:            "mock-session",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Test start command with mock API
	rootCmd.SetArgs([]string{"start", "--api-url", mockServer.URL})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Tunnel established") {
		t.Errorf("Expected output to contain 'Tunnel established', got: %s", output)
	}

	if !strings.Contains(output, "https://mock123.azhexgate.com") {
		t.Errorf("Expected output to contain mock URL, got: %s", output)
	}

	if !strings.Contains(output, "http://localhost:3000") {
		t.Errorf("Expected output to contain local port, got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestStartCommandWithCustomPort(t *testing.T) {
	// Create mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.TunnelResponse{
			PublicURL:            "https://test456.azhexgate.com",
			RelayEndpoint:        "https://test-relay.servicebus.windows.net",
			HybridConnectionName: "hc-test456",
			ListenerToken:        "test-token",
			SessionID:            "test-session",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Test start command with custom port
	rootCmd.SetArgs([]string{"start", "--port", "8080", "--api-url", mockServer.URL})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "http://localhost:8080") {
		t.Errorf("Expected output to contain custom port 8080, got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestStartCommandAPIError(t *testing.T) {
	// Create mock API server that returns error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
	}))
	defer mockServer.Close()

	// Test start command with API error
	rootCmd.SetArgs([]string{"start", "--api-url", mockServer.URL})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create tunnel") {
		t.Errorf("Expected error to contain 'failed to create tunnel', got: %v", err)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestStartCommandInvalidJSON(t *testing.T) {
	// Create mock API server that returns invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer mockServer.Close()

	// Test start command with invalid JSON
	rootCmd.SetArgs([]string{"start", "--api-url", mockServer.URL})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to") {
		t.Errorf("Expected error to contain 'failed to', got: %v", err)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestStartCommandNetworkError(t *testing.T) {
	// Test start command with unreachable API
	rootCmd.SetArgs([]string{"start", "--api-url", "http://localhost:99999"})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to") {
		t.Errorf("Expected error to contain 'failed to', got: %v", err)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestVerboseFlag(t *testing.T) {
	// Create mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.TunnelResponse{
			PublicURL:            "https://verbose.azhexgate.com",
			RelayEndpoint:        "https://relay.servicebus.windows.net",
			HybridConnectionName: "hc-verbose",
			ListenerToken:        "verbose-token",
			SessionID:            "verbose-session",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Test verbose flag
	rootCmd.SetArgs([]string{"start", "-v", "--api-url", mockServer.URL})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Logger should be initialized with DEBUG level when verbose flag is set
	// Note: We can't easily test the log output since it goes to stdout
	// but we verify the command completes without error

	// Reset for next test
	rootCmd.SetArgs(nil)
	verboseFlag = false
}

func TestRootCommandHelp(t *testing.T) {
	// Test help command
	rootCmd.SetArgs([]string{"--help"})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "azhexgate") {
		t.Errorf("Expected output to contain 'azhexgate', got: %s", output)
	}

	if !strings.Contains(output, "start") {
		t.Errorf("Expected output to contain 'start' command, got: %s", output)
	}

	if !strings.Contains(output, "verbose") {
		t.Errorf("Expected output to contain 'verbose' flag, got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}
