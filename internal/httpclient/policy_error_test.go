package httpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewErrorPolicy(t *testing.T) {
	policy := NewErrorPolicy()
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
}

func TestErrorPolicyDoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	policy := NewErrorPolicy()
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

func TestErrorPolicyDoWithError(t *testing.T) {
	policy := NewErrorPolicy()
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	expectedErr := errors.New("connection failed")
	next := func(r *http.Request) (*http.Response, error) {
		return nil, expectedErr
	}

	resp, err := policy.Do(req, next)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify error is wrapped with URL context
	if !strings.Contains(err.Error(), "request to http://example.com failed") {
		t.Errorf("Expected error to contain URL context, got: %v", err)
	}

	// Verify original error is preserved
	if !strings.Contains(err.Error(), "connection failed") {
		t.Errorf("Expected error to contain original error, got: %v", err)
	}
}

func TestErrorPolicyDoWrapsError(t *testing.T) {
	policy := NewErrorPolicy()
	testURL := "http://test.example.com/api/endpoint"
	req, _ := http.NewRequest(http.MethodPost, testURL, nil)

	next := func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("network timeout")
	}

	resp, err := policy.Do(req, next)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, testURL) {
		t.Errorf("Expected error message to contain '%s', got: %v", testURL, errorMsg)
	}

	if !strings.Contains(errorMsg, "network timeout") {
		t.Errorf("Expected error message to contain 'network timeout', got: %v", errorMsg)
	}
}

func TestErrorPolicyPreservesErrorChain(t *testing.T) {
	policy := NewErrorPolicy()
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	baseErr := errors.New("base error")
	next := func(r *http.Request) (*http.Response, error) {
		return nil, baseErr
	}

	resp, err := policy.Do(req, next)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify the error chain is preserved (using errors.Is)
	if !errors.Is(err, baseErr) {
		t.Error("Expected error chain to be preserved")
	}
}
