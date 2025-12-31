package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienstroheker/AzHexGate/gateway/http/middleware"
	"github.com/julienstroheker/AzHexGate/gateway/tunnel"
	"github.com/julienstroheker/AzHexGate/internal/config"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

func TestServer_MiddlewareIntegration(t *testing.T) {
	logger := logging.New(logging.InfoLevel)
	manager := tunnel.NewManager(&tunnel.Options{
		Mode: config.ModeRemote,
	})
	server := NewServer(&Options{
		Port:    9999,
		Manager: manager,
		Logger:  logger,
	})

	// Test that telemetry headers are added
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Client-Request-Id", "test-123")
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	// Verify response has telemetry headers
	clientReqID := resp.Header.Get("X-Client-Request-Id")
	if clientReqID != "test-123" {
		t.Errorf("Expected X-Client-Request-Id header 'test-123', got '%s'", clientReqID)
	}

	requestID := resp.Header.Get("X-Ms-Request-Id")
	if requestID == "" {
		t.Error("Expected X-Ms-Request-Id header to be present")
	}
}

func TestServer_MiddlewareContextPropagation(t *testing.T) {
	logger := logging.New(logging.InfoLevel)
	manager := tunnel.NewManager(&tunnel.Options{
		Mode: config.ModeRemote,
	})
	server := NewServer(&Options{
		Port:    9999,
		Manager: manager,
		Logger:  logger,
	})

	// Create a custom handler to verify context values
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify telemetry IDs are in context
		requestID := middleware.GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Expected request ID in context")
		}

		clientReqID := middleware.GetClientRequestID(r.Context())
		if clientReqID != "client-456" {
			t.Errorf("Expected client request ID 'client-456' in context, got '%s'", clientReqID)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Wrap test handler with server middleware chain
	handler := server.server.Handler

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Client-Request-Id", "client-456")
	w := httptest.NewRecorder()

	// First, let the middleware process the request
	handler.ServeHTTP(w, req)

	// Create another request to verify the test handler
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Client-Request-Id", "client-456")
	w2 := httptest.NewRecorder()

	// Apply only middleware to test handler
	wrappedTestHandler := middleware.Metrics(testHandler)
	wrappedTestHandler = middleware.Logger(server.logger)(wrappedTestHandler)
	wrappedTestHandler = middleware.Telemetry(wrappedTestHandler)

	wrappedTestHandler.ServeHTTP(w2, req2)
}

func TestServer_MiddlewareWithoutClientRequestID(t *testing.T) {
	logger := logging.New(logging.InfoLevel)
	manager := tunnel.NewManager(&tunnel.Options{
		Mode: config.ModeRemote,
	})
	server := NewServer(&Options{
		Port:    9999,
		Manager: manager,
		Logger:  logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	// Verify X-Client-Request-Id is not in response when not provided
	clientReqID := resp.Header.Get("X-Client-Request-Id")
	if clientReqID != "" {
		t.Errorf("Expected no X-Client-Request-Id header, got '%s'", clientReqID)
	}

	// Verify X-Ms-Request-Id is still present
	requestID := resp.Header.Get("X-Ms-Request-Id")
	if requestID == "" {
		t.Error("Expected X-Ms-Request-Id header to be present")
	}
}
