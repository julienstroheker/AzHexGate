package httpclient

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

func TestNewLoggingPolicy(t *testing.T) {
	logger := logging.New(logging.DebugLevel)
	policy := NewLoggingPolicy(logger, nil)

	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
	if policy.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestLoggingPolicyWithOptions(t *testing.T) {
	logger := logging.New(logging.DebugLevel)
	opts := &LoggingOptions{
		LogHeaders:    true,
		LogBody:       true,
		RedactBody:    false,
		HeaderFilters: []string{"Authorization"},
	}
	policy := NewLoggingPolicy(logger, opts)

	if !policy.logHeaders {
		t.Error("Expected logHeaders to be true")
	}
	if !policy.logBody {
		t.Error("Expected logBody to be true")
	}
	if policy.redactBody {
		t.Error("Expected redactBody to be false")
	}
	if len(policy.headerFilters) != 1 {
		t.Errorf("Expected 1 header filter, got %d", len(policy.headerFilters))
	}
}

func TestLoggingPolicyDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body"))
	}))
	defer server.Close()

	// Use buffer to capture log output
	var logBuf bytes.Buffer
	logger := logging.NewWithOutput(logging.DebugLevel, &logBuf)
	policy := NewLoggingPolicy(logger, nil)

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

	// Verify log output contains expected messages
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "HTTP Request") {
		t.Error("Expected log output to contain 'HTTP Request'")
	}
	if !strings.Contains(logOutput, "HTTP Response") {
		t.Error("Expected log output to contain 'HTTP Response'")
	}
	if !strings.Contains(logOutput, "method=GET") {
		t.Error("Expected log output to contain 'method=GET'")
	}
	if !strings.Contains(logOutput, "status=200") {
		t.Error("Expected log output to contain 'status=200'")
	}
}

func TestLoggingPolicyWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use buffer to capture log output
	var logBuf bytes.Buffer
	logger := logging.NewWithOutput(logging.DebugLevel, &logBuf)
	opts := &LoggingOptions{
		LogHeaders: true,
	}
	policy := NewLoggingPolicy(logger, opts)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	req.Header.Set("X-Request-Header", "request-value")

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

	// Verify log output contains headers
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "request_headers=") {
		t.Error("Expected log output to contain request_headers")
	}
	if !strings.Contains(logOutput, "response_headers=") {
		t.Error("Expected log output to contain response_headers")
	}
}

func TestLoggingPolicyWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body content"))
	}))
	defer server.Close()

	// Use buffer to capture log output
	var logBuf bytes.Buffer
	logger := logging.NewWithOutput(logging.DebugLevel, &logBuf)
	opts := &LoggingOptions{
		LogBody: true,
	}
	policy := NewLoggingPolicy(logger, opts)

	bodyContent := "request body content"
	req, _ := http.NewRequest(http.MethodPost, server.URL, bytes.NewBufferString(bodyContent))

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

	// Verify response body is still readable after logging
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "response body content" {
		t.Errorf("Expected response body to be readable, got: %s", string(body))
	}

	// Verify log output contains body information
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "request_body=") {
		t.Error("Expected log output to contain request_body")
	}
	if !strings.Contains(logOutput, "response_body=") {
		t.Error("Expected log output to contain response_body")
	}
	if !strings.Contains(logOutput, "request body content") {
		t.Error("Expected log output to contain request body content")
	}
	if !strings.Contains(logOutput, "response body content") {
		t.Error("Expected log output to contain response body content")
	}
}

func TestLoggingPolicyRedactBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("sensitive data"))
	}))
	defer server.Close()

	// Use buffer to capture log output
	var logBuf bytes.Buffer
	logger := logging.NewWithOutput(logging.DebugLevel, &logBuf)
	opts := &LoggingOptions{
		LogBody:    true,
		RedactBody: true,
	}
	policy := NewLoggingPolicy(logger, opts)

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

	// Verify log output contains redacted body (size only)
	logOutput := logBuf.String()
	if strings.Contains(logOutput, "sensitive data") {
		t.Error("Expected body content to be redacted, but found 'sensitive data'")
	}
	if !strings.Contains(logOutput, "response_body_size=") {
		t.Error("Expected log output to contain 'response_body_size='")
	}
	if !strings.Contains(logOutput, "bytes") {
		t.Error("Expected log output to contain 'bytes'")
	}
}

func TestLoggingPolicyHeaderFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use buffer to capture log output
	var logBuf bytes.Buffer
	logger := logging.NewWithOutput(logging.DebugLevel, &logBuf)
	opts := &LoggingOptions{
		LogHeaders:    true,
		HeaderFilters: []string{"Authorization", "X-Api-Key"},
	}
	policy := NewLoggingPolicy(logger, opts)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("X-Api-Key", "secret-key")
	req.Header.Set("X-Public-Header", "public-value")

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

	// Verify log output contains redacted headers
	logOutput := logBuf.String()
	if strings.Contains(logOutput, "secret-token") {
		t.Error("Expected Authorization header to be redacted, but found secret-token")
	}
	if strings.Contains(logOutput, "secret-key") {
		t.Error("Expected X-Api-Key header to be redacted, but found secret-key")
	}
	if !strings.Contains(logOutput, "[REDACTED]") {
		t.Error("Expected log output to contain '[REDACTED]'")
	}
	if !strings.Contains(logOutput, "public-value") {
		t.Error("Expected public header value to be visible")
	}
}

func TestLoggingPolicyFormatHeaders(t *testing.T) {
	var logBuf bytes.Buffer
	logger := logging.NewWithOutput(logging.DebugLevel, &logBuf)
	opts := &LoggingOptions{
		HeaderFilters: []string{"Authorization"},
	}
	policy := NewLoggingPolicy(logger, opts)

	headers := http.Header{}
	headers.Set("Authorization", "Bearer token")
	headers.Set("Content-Type", "application/json")

	field := policy.formatHeaders("test_headers", headers)

	if field.Key != "test_headers" {
		t.Errorf("Expected key 'test_headers', got '%s'", field.Key)
	}

	valueStr, ok := field.Value.(string)
	if !ok {
		t.Fatal("Expected value to be a string")
	}

	if !strings.Contains(valueStr, "[REDACTED]") {
		t.Error("Expected Authorization header to be redacted")
	}
	if !strings.Contains(valueStr, "application/json") {
		t.Error("Expected Content-Type header to be visible")
	}
}
