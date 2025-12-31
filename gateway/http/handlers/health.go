package handlers

import (
	"net/http"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve logger from context to establish the pattern for future handlers
	// Currently not used for logging as this is a simple health check
	_ = logging.FromContext(r.Context())

	// Only accept GET requests
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	// Ignore write error for health check as status is already set
	_, _ = w.Write([]byte("OK"))
}
