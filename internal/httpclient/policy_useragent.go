package httpclient

import (
	"net/http"
)

// UserAgentPolicy adds a User-Agent header to requests
type UserAgentPolicy struct {
	userAgent string
}

// NewUserAgentPolicy creates a new UserAgentPolicy
func NewUserAgentPolicy(userAgent string) *UserAgentPolicy {
	if userAgent == "" {
		userAgent = "azhexgate-client/1.0"
	}
	return &UserAgentPolicy{userAgent: userAgent}
}

// Do implements Policy interface
func (p *UserAgentPolicy) Do(
	req *http.Request,
	next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
	req.Header.Set("User-Agent", p.userAgent)
	return next(req)
}
