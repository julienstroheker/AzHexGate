package httpclient

import (
	"net/http"

	"github.com/google/uuid"
)

const defaultRequestIDHeader = "X-Client-Request-Id"

// RequestIDPolicy adds a unique request ID to each request
type RequestIDPolicy struct {
	headerName string
}

// NewRequestIDPolicy creates a new RequestIDPolicy with a custom header name
func NewRequestIDPolicy(headerName string) *RequestIDPolicy {
	if headerName == "" {
		headerName = defaultRequestIDHeader
	}
	return &RequestIDPolicy{headerName: headerName}
}

// NewDefaultRequestIDPolicy creates a new RequestIDPolicy with the default header name
func NewDefaultRequestIDPolicy() *RequestIDPolicy {
	return &RequestIDPolicy{headerName: defaultRequestIDHeader}
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
