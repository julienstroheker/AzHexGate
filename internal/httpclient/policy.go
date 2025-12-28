package httpclient

import (
	"net/http"
)

// Policy represents a middleware that can modify requests and responses
type Policy interface {
	// Do executes the policy and calls the next policy in the chain
	Do(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error)
}

// PolicyFunc is a function adapter for Policy interface
type PolicyFunc func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error)

// Do implements Policy interface
func (f PolicyFunc) Do(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	return f(req, next)
}
