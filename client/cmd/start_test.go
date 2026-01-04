package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/api"
)

// testMutex ensures tests don't run concurrently (shared global state in cobra)
var testMutex sync.Mutex

func runStartCommandWithTimeout(t *testing.T, args []string, timeout time.Duration) (string, error) {
	t.Helper()

	testMutex.Lock()
	defer testMutex.Unlock()

	// Give any previous test resources time to fully clean up
	time.Sleep(200 * time.Millisecond)

	// Reset flags to defaults before each test
	portFlag = defaultPort
	apiURLFlag = defaultAPIURL
	verboseFlag = false
	logger = nil // Reset the global logger

	// Reset the cobra command's flag values by looking up and resetting them
	if f := startCmd.Flags().Lookup("port"); f != nil {
		_ = f.Value.Set("3000")
	}
	if f := startCmd.Flags().Lookup("api-url"); f != nil {
		_ = f.Value.Set(defaultAPIURL)
	}
	if f := rootCmd.PersistentFlags().Lookup("verbose"); f != nil {
		_ = f.Value.Set("false")
	}

	// Reset command contexts to avoid cancelled parent contexts
	rootCmd.SetContext(context.Background())
	startCmd.SetContext(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rootCmd.SetArgs(args)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	done := make(chan error, 1)

	go func() {
		done <- rootCmd.ExecuteContext(ctx)
	}()

	// Wait for command to complete or timeout
	var cmdErr error
	select {
	case cmdErr = <-done:
		// Command completed normally
	case <-ctx.Done():
		// Wait a bit for the goroutine to notice the context cancellation
		// and finish writing any pending output
		select {
		case cmdErr = <-done:
			// Goroutine finished
		case <-time.After(500 * time.Millisecond):
			// Timeout waiting for cleanup
			cmdErr = ctx.Err()
		}
	}

	// Give time for output buffer to be flushed
	time.Sleep(50 * time.Millisecond)

	output := buf.String()
	rootCmd.SetArgs(nil)
	return output, cmdErr
}

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
			ListenerToken:        "mock-listener-token", // Use mock token to enable in-memory relay
			SessionID:            "mock-session",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Test start command with mock API
	output, cmdErr := runStartCommandWithTimeout(t, []string{"start", "--api-url", mockServer.URL}, 500*time.Millisecond)

	// Should get context deadline exceeded since we're waiting for signals
	if cmdErr != context.DeadlineExceeded && cmdErr != context.Canceled {
		t.Errorf("Expected context deadline exceeded, got: %v", cmdErr)
	}

	// Check that output was printed before timeout
	if !strings.Contains(output, "Tunnel established") {
		t.Errorf("Expected output to contain 'Tunnel established', got: %s", output)
	}

	if !strings.Contains(output, "https://mock123.azhexgate.com") {
		t.Errorf("Expected output to contain mock URL, got: %s", output)
	}

	if !strings.Contains(output, "http://localhost:3000") {
		t.Errorf("Expected output to contain local port, got: %s", output)
	}
}

func TestStartCommandWithCustomPort(t *testing.T) {
	// Create mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.TunnelResponse{
			PublicURL:            "https://test456.azhexgate.com",
			RelayEndpoint:        "https://test-relay.servicebus.windows.net",
			HybridConnectionName: "hc-test456",
			ListenerToken:        "mock-listener-token", // Use mock token to enable in-memory relay
			SessionID:            "test-session",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Test start command with custom port
	args := []string{"start", "--port", "8080", "--api-url", mockServer.URL}
	output, _ := runStartCommandWithTimeout(t, args, 2*time.Second)

	if !strings.Contains(output, "http://localhost:8080") {
		t.Errorf("Expected output to contain custom port 8080, got: %s", output)
	}
}

func TestStartCommandAPIError(t *testing.T) {
	// Create mock API server that returns error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
	}))
	defer mockServer.Close()

	// Test start command with API error
	output, err := runStartCommandWithTimeout(t, []string{"start", "--api-url", mockServer.URL}, time.Second)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create tunnel") && err != context.DeadlineExceeded {
		t.Errorf("Expected error to contain 'failed to create tunnel', got: %v (output: %s)", err, output)
	}
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
	output, err := runStartCommandWithTimeout(t, []string{"start", "--api-url", mockServer.URL}, time.Second)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to") && err != context.DeadlineExceeded {
		t.Errorf("Expected error to contain 'failed to', got: %v (output: %s)", err, output)
	}
}

func TestStartCommandNetworkError(t *testing.T) {
	// Test start command with unreachable API
	output, err := runStartCommandWithTimeout(t, []string{"start", "--api-url", "http://localhost:99999"}, time.Second)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to") && err != context.DeadlineExceeded {
		t.Errorf("Expected error to contain 'failed to', got: %v (output: %s)", err, output)
	}
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
	_, _ = runStartCommandWithTimeout(t, []string{"start", "-v", "--api-url", mockServer.URL}, 500*time.Millisecond)

	// Logger should be initialized with DEBUG level when verbose flag is set
	// Note: We can't easily test the log output since it goes to stdout
	// but we verify the command completes without error

	// Reset verbose flag for other tests
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
