package httpclient

import (
	"net/http"

	"github.com/google/uuid"
)

// RequestIDPolicy adds a unique request ID to each request
type RequestIDPolicy struct {
	headerName string
}

// NewRequestIDPolicy creates a new RequestIDPolicy
func NewRequestIDPolicy(headerName string) *RequestIDPolicy {
	if headerName == "" {
		headerName = "X-Client-Request-Id"
	}
	return &RequestIDPolicy{headerName: headerName}
}

// Do implements Policy interface
func (p *RequestIDPolicy) Do(
	req *http.Request,
	next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
	// Generate a new UUID for this request
	requestID := uuid.New().String()
	req.Header.Set(p.headerName, requestID)
	return next(req)
}
