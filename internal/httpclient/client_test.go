package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

func TestNewClient(t *testing.T) {
	client := NewClient(nil)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if len(client.policies) == 0 {
		t.Error("Expected client to have policies")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	opts := &Options{
		Timeout:    10 * time.Second,
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
		UserAgent:  "test-agent/1.0",
	}

	client := NewClient(opts)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

func TestClientDo(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that request ID header is set
		if r.Header.Get("X-Client-Request-Id") == "" {
			t.Error("Expected X-Client-Request-Id header to be set")
		}

		// Check that User-Agent header is set
		if r.Header.Get("User-Agent") == "" {
			t.Error("Expected User-Agent header to be set")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewClient(DefaultOptions())

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "success" {
		t.Errorf("Expected body 'success', got '%s'", string(body))
	}
}

func TestClientWithLogging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("logged"))
	}))
	defer server.Close()

	logger := logging.New(logging.DebugLevel)
	opts := &Options{
		Timeout: 10 * time.Second,
		Logger:  logger,
	}

	client := NewClient(opts)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRetryPolicy(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success after retry"))
	}))
	defer server.Close()

	opts := &Options{
		Timeout:    10 * time.Second,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond, // Short delay for testing
	}

	client := NewClient(opts)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 after retries, got %d", resp.StatusCode)
	}

	if attempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", attempts)
	}
}

func TestCustomPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check custom header
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Error("Expected X-Custom-Header to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a custom policy
	customPolicy := PolicyFunc(func(
		req *http.Request,
		next func(*http.Request) (*http.Response, error),
	) (*http.Response, error) {
		req.Header.Set("X-Custom-Header", "custom-value")
		return next(req)
	})

	opts := &Options{
		Timeout:            10 * time.Second,
		AdditionalPolicies: []Policy{customPolicy},
	}

	client := NewClient(opts)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestMockTransport(t *testing.T) {
	// Create a mock transport for testing
	mockTransport := &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("mocked response")),
				Header:     make(http.Header),
			}, nil
		},
	}

	opts := &Options{
		Timeout:   10 * time.Second,
		Transport: mockTransport,
	}

	client := NewClient(opts)

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "mocked response" {
		t.Errorf("Expected 'mocked response', got '%s'", string(body))
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
