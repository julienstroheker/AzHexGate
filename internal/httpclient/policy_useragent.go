package httpclient

import (
	"fmt"
	"net/http"
	"runtime"
)

var defaultUserAgent = fmt.Sprintf(
	"azhexgate-client/1.0 (Go/%s; %s/%s)",
	runtime.Version(), runtime.GOOS, runtime.GOARCH,
)

// UserAgentPolicy adds a User-Agent header to requests
type UserAgentPolicy struct {
	userAgent string
}

// NewUserAgentPolicy creates a new UserAgentPolicy
func NewUserAgentPolicy(userAgent string) *UserAgentPolicy {
	if userAgent == "" {
		userAgent = defaultUserAgent
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
