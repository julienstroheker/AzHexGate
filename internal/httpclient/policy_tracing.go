package httpclient

import (
	"net/http"
)

// TracingPolicy adds tracing information to requests
type TracingPolicy struct {
	spanID string
}

// NewTracingPolicy creates a new TracingPolicy
func NewTracingPolicy(spanID string) *TracingPolicy {
	return &TracingPolicy{spanID: spanID}
}

// Do implements Policy interface
func (p *TracingPolicy) Do(
	req *http.Request,
	next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
	if p.spanID != "" {
		req.Header.Set("X-Trace-Span-Id", p.spanID)
	}
	return next(req)
}
