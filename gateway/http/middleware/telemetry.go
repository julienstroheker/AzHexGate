package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// ClientRequestIDKey is the context key for the client request ID
	ClientRequestIDKey contextKey = "X-Client-Request-Id"
	// RequestIDKey is the context key for the internal request ID
	RequestIDKey contextKey = "x-ms-request-id"
)

// Telemetry is a middleware that handles request tracing and telemetry headers
// It extracts X-Client-Request-Id if present and generates a new x-ms-request-id for each request
// Both headers are added to the response and stored in the request context
func Telemetry(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client request ID if present
		clientRequestID := r.Header.Get("X-Client-Request-Id")

		// Generate new request ID
		requestID := uuid.New().String()

		// Add headers to response
		if clientRequestID != "" {
			w.Header().Set("X-Client-Request-Id", clientRequestID)
		}
		w.Header().Set("X-Ms-Request-Id", requestID)

		// Store both IDs in context for downstream handlers
		ctx := r.Context()
		if clientRequestID != "" {
			ctx = context.WithValue(ctx, ClientRequestIDKey, clientRequestID)
		}
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Pass the request with enriched context to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClientRequestID retrieves the client request ID from the context
func GetClientRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(ClientRequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetRequestID retrieves the internal request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
