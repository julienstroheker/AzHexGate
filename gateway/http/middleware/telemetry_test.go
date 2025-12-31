package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelemetry_GeneratesRequestID(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Expected request ID to be present in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with telemetry middleware
	middleware := Telemetry(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)

	// Verify response has X-Ms-Request-Id header
	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	requestID := resp.Header.Get("X-Ms-Request-Id")
	if requestID == "" {
		t.Error("Expected X-Ms-Request-Id header in response")
	}
}

func TestTelemetry_PreservesClientRequestID(t *testing.T) {
	clientReqID := "client-test-123"

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify client request ID is in context
		ctxClientID := GetClientRequestID(r.Context())
		if ctxClientID != clientReqID {
			t.Errorf("Expected client request ID %s in context, got %s", clientReqID, ctxClientID)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with telemetry middleware
	middleware := Telemetry(handler)

	// Create test request with client request ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Client-Request-Id", clientReqID)
	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)

	// Verify response has both headers
	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	respClientID := resp.Header.Get("X-Client-Request-Id")
	if respClientID != clientReqID {
		t.Errorf("Expected X-Client-Request-Id header %s in response, got %s", clientReqID, respClientID)
	}

	requestID := resp.Header.Get("X-Ms-Request-Id")
	if requestID == "" {
		t.Error("Expected X-Ms-Request-Id header in response")
	}
}

func TestTelemetry_NoClientRequestID(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify client request ID is not in context
		ctxClientID := GetClientRequestID(r.Context())
		if ctxClientID != "" {
			t.Errorf("Expected no client request ID in context, got %s", ctxClientID)
		}

		// Verify request ID is in context
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Expected request ID to be present in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with telemetry middleware
	middleware := Telemetry(handler)

	// Create test request without client request ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)

	// Verify response has X-Ms-Request-Id but not X-Client-Request-Id
	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	respClientID := resp.Header.Get("X-Client-Request-Id")
	if respClientID != "" {
		t.Errorf("Expected no X-Client-Request-Id header in response, got %s", respClientID)
	}

	requestID := resp.Header.Get("X-Ms-Request-Id")
	if requestID == "" {
		t.Error("Expected X-Ms-Request-Id header in response")
	}
}

func TestTelemetry_UniqueRequestIDs(t *testing.T) {
	var requestID1, requestID2 string

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with telemetry middleware
	middleware := Telemetry(handler)

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)
	requestID1 = w1.Result().Header.Get("X-Ms-Request-Id")
	_ = w1.Result().Body.Close()

	// Second request
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)
	requestID2 = w2.Result().Header.Get("X-Ms-Request-Id")
	_ = w2.Result().Body.Close()

	// Verify request IDs are different
	if requestID1 == "" || requestID2 == "" {
		t.Error("Expected both request IDs to be present")
	}
	if requestID1 == requestID2 {
		t.Error("Expected request IDs to be unique")
	}
}

func TestGetClientRequestID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	clientID := GetClientRequestID(ctx)
	if clientID != "" {
		t.Errorf("Expected empty client ID, got %s", clientID)
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	requestID := GetRequestID(ctx)
	if requestID != "" {
		t.Errorf("Expected empty request ID, got %s", requestID)
	}
}

func TestGetClientRequestID_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClientRequestIDKey, "test-client-id")
	clientID := GetClientRequestID(ctx)
	if clientID != "test-client-id" {
		t.Errorf("Expected 'test-client-id', got %s", clientID)
	}
}

func TestGetRequestID_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-id")
	requestID := GetRequestID(ctx)
	if requestID != "test-request-id" {
		t.Errorf("Expected 'test-request-id', got %s", requestID)
	}
}
