package middleware

import (
	"net/http"
)

// Metrics is a middleware placeholder for emitting metrics about web server activity
// This middleware is currently a no-op and will be implemented in the future
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement metrics collection
		// This will track request counts, response times, status codes, etc.
		next.ServeHTTP(w, r)
	})
}
