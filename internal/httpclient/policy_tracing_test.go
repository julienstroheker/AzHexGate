package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewTracingPolicy(t *testing.T) {
	policy := NewTracingPolicy("")
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
	if policy.spanID != "" {
		t.Errorf("Expected empty spanID, got '%s'", policy.spanID)
	}
}

func TestNewTracingPolicyWithSpanID(t *testing.T) {
	policy := NewTracingPolicy("test-span-123")
	if policy.spanID != "test-span-123" {
		t.Errorf("Expected spanID 'test-span-123', got '%s'", policy.spanID)
	}
}

func TestTracingPolicyDoWithSpanID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify tracing header is set
		spanID := r.Header.Get("X-Trace-Span-Id")
		if spanID != "test-span-456" {
			t.Errorf("Expected X-Trace-Span-Id 'test-span-456', got '%s'", spanID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	policy := NewTracingPolicy("test-span-456")
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	next := func(r *http.Request) (*http.Response, error) {
		return http.DefaultClient.Do(r)
	}

	resp, err := policy.Do(req, next)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestTracingPolicyDoWithoutSpanID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify tracing header is NOT set
		spanID := r.Header.Get("X-Trace-Span-Id")
		if spanID != "" {
			t.Errorf("Expected no X-Trace-Span-Id header, got '%s'", spanID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	policy := NewTracingPolicy("")
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	next := func(r *http.Request) (*http.Response, error) {
		return http.DefaultClient.Do(r)
	}

	resp, err := policy.Do(req, next)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
