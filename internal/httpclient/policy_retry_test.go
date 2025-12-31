package httpclient

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

func TestNewRetryPolicy(t *testing.T) {
	policy := NewRetryPolicy(nil)
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
	if policy.maxRetries != 3 {
		t.Errorf("Expected default maxRetries 3, got %d", policy.maxRetries)
	}
	if policy.retryDelay != time.Second {
		t.Errorf("Expected default retryDelay 1s, got %v", policy.retryDelay)
	}
}

func TestNewRetryPolicyWithOptions(t *testing.T) {
	opts := &RetryOptions{
		MaxRetries:       5,
		RetryDelay:       2 * time.Second,
		RetryStatusCodes: []int{500, 503},
	}
	policy := NewRetryPolicy(opts)

	if policy.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", policy.maxRetries)
	}
	if policy.retryDelay != 2*time.Second {
		t.Errorf("Expected retryDelay 2s, got %v", policy.retryDelay)
	}
	if len(policy.retryStatusCodes) != 2 {
		t.Errorf("Expected 2 retry status codes, got %d", len(policy.retryStatusCodes))
	}
}

func TestRetryPolicySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	policy := NewRetryPolicy(nil)
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

func TestRetryPolicyRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := &RetryOptions{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}
	policy := NewRetryPolicy(opts)
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
		t.Errorf("Expected status 200 after retries, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryPolicyMaxRetriesExceeded(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	opts := &RetryOptions{
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	}
	policy := NewRetryPolicy(opts)
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

	// Should still be 500 after max retries
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after max retries, got %d", resp.StatusCode)
	}

	expectedAttempts := 3 // initial + 2 retries
	if attempts != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attempts)
	}
}

func TestRetryPolicyCustomStatusCodes(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadGateway) // 502
	}))
	defer server.Close()

	opts := &RetryOptions{
		MaxRetries:       2,
		RetryDelay:       10 * time.Millisecond,
		RetryStatusCodes: []int{502}, // Only retry on 502
	}
	policy := NewRetryPolicy(opts)
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	next := func(r *http.Request) (*http.Response, error) {
		return http.DefaultClient.Do(r)
	}

	resp, err := policy.Do(req, next)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedAttempts := 3 // initial + 2 retries
	if attempts != expectedAttempts {
		t.Errorf("Expected %d attempts for 502 error, got %d", expectedAttempts, attempts)
	}
}

func TestRetryPolicyNoRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest) // 400
	}))
	defer server.Close()

	opts := &RetryOptions{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}
	policy := NewRetryPolicy(opts)
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

	// Should not retry on 400
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for 400 error (no retry), got %d", attempts)
	}
}

func TestRetryPolicyWithLogger(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use buffer to capture log output
	var logBuf bytes.Buffer
	logger := logging.NewWithOutput(logging.DebugLevel, &logBuf)
	opts := &RetryOptions{
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
		Logger:     logger,
	}
	policy := NewRetryPolicy(opts)
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

	// Verify log output contains retry message
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Retrying request") {
		t.Error("Expected log output to contain 'Retrying request'")
	}
	if !strings.Contains(logOutput, "attempt=1") {
		t.Error("Expected log output to contain 'attempt=1'")
	}
	if !strings.Contains(logOutput, "max_retries=2") {
		t.Error("Expected log output to contain 'max_retries=2'")
	}
}

func TestShouldRetry(t *testing.T) {
	policy := NewRetryPolicy(nil)

	tests := []struct {
		name        string
		statusCode  int
		shouldRetry bool
	}{
		{"nil response", 0, true},
		{"200 OK", http.StatusOK, false},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"429 Too Many Requests", http.StatusTooManyRequests, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"502 Bad Gateway", http.StatusBadGateway, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout", http.StatusGatewayTimeout, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.statusCode > 0 {
				resp = &http.Response{StatusCode: tt.statusCode}
			}

			result := policy.shouldRetry(resp)
			if result != tt.shouldRetry {
				t.Errorf("shouldRetry(%d) = %v, want %v", tt.statusCode, result, tt.shouldRetry)
			}
		})
	}
}
