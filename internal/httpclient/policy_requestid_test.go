package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRequestIDPolicy(t *testing.T) {
	policy := NewRequestIDPolicy("")
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
	if policy.headerName != "X-Client-Request-Id" {
		t.Errorf("Expected default header name 'X-Client-Request-Id', got '%s'", policy.headerName)
	}
}

func TestNewDefaultRequestIDPolicy(t *testing.T) {
	policy := NewDefaultRequestIDPolicy()
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
	if policy.headerName != "X-Client-Request-Id" {
		t.Errorf("Expected default header name 'X-Client-Request-Id', got '%s'", policy.headerName)
	}
}

func TestNewRequestIDPolicyCustomHeader(t *testing.T) {
	policy := NewRequestIDPolicy("X-Custom-Request-Id")
	if policy.headerName != "X-Custom-Request-Id" {
		t.Errorf("Expected header name 'X-Custom-Request-Id', got '%s'", policy.headerName)
	}
}

func TestRequestIDPolicyDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID header is set
		requestID := r.Header.Get("X-Client-Request-Id")
		if requestID == "" {
			t.Error("Expected X-Client-Request-Id header to be set")
		}
		// Verify it's a valid UUID format (simple check)
		if len(requestID) != 36 {
			t.Errorf("Expected UUID format (36 chars), got length %d", len(requestID))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	policy := NewRequestIDPolicy("X-Client-Request-Id")
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

func TestRequestIDPolicyUniqueIDs(t *testing.T) {
	policy := NewRequestIDPolicy("X-Client-Request-Id")

	req1, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req2, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	next := func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	}

	resp1, _ := policy.Do(req1, next)
	if resp1 != nil {
		defer func() {
			_ = resp1.Body.Close()
		}()
	}

	resp2, _ := policy.Do(req2, next)
	if resp2 != nil {
		defer func() {
			_ = resp2.Body.Close()
		}()
	}

	id1 := req1.Header.Get("X-Client-Request-Id")
	id2 := req2.Header.Get("X-Client-Request-Id")

	if id1 == "" || id2 == "" {
		t.Error("Expected both requests to have IDs")
	}

	if id1 == id2 {
		t.Error("Expected unique IDs for different requests")
	}
}
