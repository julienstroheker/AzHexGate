package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testContextKey is a custom type for test context keys to avoid collisions
type testContextKey string

func TestMetrics_PassesThrough(t *testing.T) {
	// Create test handler
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with metrics middleware
	middleware := Metrics(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)

	// Verify handler was called
	if !called {
		t.Error("Expected handler to be called")
	}

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

func TestMetrics_PreservesResponse(t *testing.T) {
	// Create test handler that writes response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("test response"))
	})

	// Wrap with metrics middleware
	middleware := Metrics(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "test response" {
		t.Errorf("Expected body 'test response', got '%s'", body)
	}
}

func TestMetrics_PreservesContext(t *testing.T) {
	const testKey testContextKey = "test-key"

	// Create test handler that checks context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context is preserved
		val := r.Context().Value(testKey)
		if val == nil {
			t.Error("Expected context value to be preserved")
		} else if val.(string) != "test-value" {
			t.Errorf("Expected context value 'test-value', got '%v'", val)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with metrics middleware
	middleware := Metrics(handler)

	// Create test request with context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), testKey, "test-value")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)
}
