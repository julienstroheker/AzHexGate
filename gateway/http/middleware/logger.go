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
			// Record start time
			start := time.Now()

			// Extract telemetry IDs from context
			requestID := GetRequestID(r.Context())
			clientRequestID := GetClientRequestID(r.Context())

			// Create a child logger with request context fields
			requestFields := []logging.Field{
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.String("remote_addr", r.RemoteAddr),
			}

			if requestID != "" {
				requestFields = append(requestFields, logging.String("request_id", requestID))
			}
			if clientRequestID != "" {
				requestFields = append(requestFields, logging.String("client_request_id", clientRequestID))
			}

			// Create child logger with request fields
			requestLogger := logger.With(requestFields...)

			// Store logger in context for downstream handlers
			ctx := logging.WithContext(r.Context(), requestLogger)
			r = r.WithContext(ctx)

			// Log request received
			requestLogger.Info("Request received")

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

			// Log response sent with additional fields
			requestLogger.Info("Response sent",
				logging.Int("status", rw.statusCode),
				logging.String("duration", duration.String()),
			)
		})
	}
}
