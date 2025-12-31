package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

func TestLogger_LogsRequestAndResponse(t *testing.T) {
	// Create logger with buffer
	buf := &bytes.Buffer{}
	logger := logging.NewWithOutput(logging.InfoLevel, buf)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	})

	// Wrap with logger middleware
	middleware := Logger(logger)(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)

	// Check output
	output := buf.String()

	// Should have two log lines: request received and response sent
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(lines))
	}

	// Check first log line (request received)
	if !strings.Contains(lines[0], "Request received") {
		t.Errorf("Expected 'Request received' in first log line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "method=GET") {
		t.Errorf("Expected 'method=GET' in first log line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "path=/test") {
		t.Errorf("Expected 'path=/test' in first log line, got: %s", lines[0])
	}

	// Check second log line (response sent)
	if !strings.Contains(lines[1], "Response sent") {
		t.Errorf("Expected 'Response sent' in second log line, got: %s", lines[1])
	}
	if !strings.Contains(lines[1], "method=GET") {
		t.Errorf("Expected 'method=GET' in second log line, got: %s", lines[1])
	}
	if !strings.Contains(lines[1], "path=/test") {
		t.Errorf("Expected 'path=/test' in second log line, got: %s", lines[1])
	}
	if !strings.Contains(lines[1], "status=200") {
		t.Errorf("Expected 'status=200' in second log line, got: %s", lines[1])
	}
	if !strings.Contains(lines[1], "duration=") {
		t.Errorf("Expected 'duration=' in second log line, got: %s", lines[1])
	}
}

func TestLogger_LogsWithTelemetryIDs(t *testing.T) {
	// Create logger with buffer
	buf := &bytes.Buffer{}
	logger := logging.NewWithOutput(logging.InfoLevel, buf)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with logger middleware
	middleware := Logger(logger)(handler)

	// Create test request with context containing telemetry IDs
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "test-request-id")
	ctx = context.WithValue(ctx, ClientRequestIDKey, "test-client-id")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)

	// Check output
	output := buf.String()

	// Both log lines should contain telemetry IDs
	if !strings.Contains(output, "request_id=test-request-id") {
		t.Errorf("Expected 'request_id=test-request-id' in output, got: %s", output)
	}
	if !strings.Contains(output, "client_request_id=test-client-id") {
		t.Errorf("Expected 'client_request_id=test-client-id' in output, got: %s", output)
	}
}

func TestLogger_LogsDifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"BadRequest", http.StatusBadRequest},
		{"NotFound", http.StatusNotFound},
		{"InternalServerError", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create logger with buffer
			buf := &bytes.Buffer{}
			logger := logging.NewWithOutput(logging.InfoLevel, buf)

			// Create test handler that returns specific status
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			// Wrap with logger middleware
			middleware := Logger(logger)(handler)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Execute
			middleware.ServeHTTP(w, req)

			// Check output
			output := buf.String()
			if !strings.Contains(output, "status=") {
				t.Errorf("Expected 'status=' in output, got: %s", output)
			}
		})
	}
}

func TestLogger_LogsWithoutTelemetryIDs(t *testing.T) {
	// Create logger with buffer
	buf := &bytes.Buffer{}
	logger := logging.NewWithOutput(logging.InfoLevel, buf)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with logger middleware
	middleware := Logger(logger)(handler)

	// Create test request without telemetry context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)

	// Check output - should still log without telemetry IDs
	output := buf.String()
	if !strings.Contains(output, "Request received") {
		t.Errorf("Expected 'Request received' in output even without telemetry IDs, got: %s", output)
	}
	if !strings.Contains(output, "Response sent") {
		t.Errorf("Expected 'Response sent' in output even without telemetry IDs, got: %s", output)
	}
}

func TestLogger_LogsDifferentMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Create logger with buffer
			buf := &bytes.Buffer{}
			logger := logging.NewWithOutput(logging.InfoLevel, buf)

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with logger middleware
			middleware := Logger(logger)(handler)

			// Create test request
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			// Execute
			middleware.ServeHTTP(w, req)

			// Check output
			output := buf.String()
			expectedMethod := "method=" + method
			if !strings.Contains(output, expectedMethod) {
				t.Errorf("Expected '%s' in output, got: %s", expectedMethod, output)
			}
		})
	}
}

func TestResponseWriter_WriteWithoutWriteHeader(t *testing.T) {
	// Test that writing without calling WriteHeader explicitly sets status to 200
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        false,
	}

	_, err := rw.Write([]byte("test"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", rw.statusCode)
	}

	if !rw.written {
		t.Error("Expected written flag to be true")
	}
}

func TestResponseWriter_MultipleWriteHeader(t *testing.T) {
	// Test that calling WriteHeader multiple times only sets status once
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        false,
	}

	rw.WriteHeader(http.StatusCreated)
	rw.WriteHeader(http.StatusInternalServerError)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", rw.statusCode)
	}
}

func TestLogger_StoresLoggerInContext(t *testing.T) {
	// Create logger with buffer
	buf := &bytes.Buffer{}
	logger := logging.NewWithOutput(logging.InfoLevel, buf)

	// Create test handler that retrieves logger from context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retrievedLogger := logging.FromContext(r.Context())
		if retrievedLogger == nil {
			t.Error("Expected logger in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with logger middleware
	middleware := Logger(logger)(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	middleware.ServeHTTP(w, req)
}
