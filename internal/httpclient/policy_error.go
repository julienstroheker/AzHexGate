package httpclient

import (
	"fmt"
	"net/http"
)

// ErrorPolicy wraps errors with additional context
type ErrorPolicy struct{}

// NewErrorPolicy creates a new ErrorPolicy
func NewErrorPolicy() *ErrorPolicy {
	return &ErrorPolicy{}
}

// Do implements Policy interface
func (p *ErrorPolicy) Do(
	req *http.Request,
	next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
	resp, err := next(req)
	if err != nil {
		return resp, fmt.Errorf("request to %s failed: %w", req.URL.String(), err)
	}
	return resp, nil
}
