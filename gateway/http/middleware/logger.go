package middleware

import (
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter

	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Logger is a middleware that logs HTTP requests and responses
// It logs when a request is received and when the response is sent
// The logger is stored in the request context for downstream handlers to access
func Logger(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Store logger in context for downstream handlers
			ctx := logging.WithContext(r.Context(), logger)
			r = r.WithContext(ctx)

			// Record start time
			start := time.Now()

			// Retrieve logger from context
			currentLogger := logging.FromContext(ctx)

			// Extract telemetry IDs from context
			requestID := GetRequestID(ctx)
			clientRequestID := GetClientRequestID(ctx)

			// Build request log fields
			fields := []logging.Field{
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.String("remote_addr", r.RemoteAddr),
			}

			if requestID != "" {
				fields = append(fields, logging.String("request_id", requestID))
			}
			if clientRequestID != "" {
				fields = append(fields, logging.String("client_request_id", clientRequestID))
			}

			// Log request received
			currentLogger.Info("Request received", fields...)

			// Wrap response writer to capture status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				written:        false,
			}

			// Call next handler
			next.ServeHTTP(rw, r)

			// Calculate response time
			duration := time.Since(start)

			// Build response log fields
			responseFields := []logging.Field{
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.Int("status", rw.statusCode),
				logging.String("duration", duration.String()),
			}

			if requestID != "" {
				responseFields = append(responseFields, logging.String("request_id", requestID))
			}
			if clientRequestID != "" {
				responseFields = append(responseFields, logging.String("client_request_id", clientRequestID))
			}

			// Log response sent
			currentLogger.Info("Response sent", responseFields...)
		})
	}
}
