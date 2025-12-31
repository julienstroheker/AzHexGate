package httpclient

import (
	"context"
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// Client is a production-ready HTTP client with retry, logging, and policy support
type Client struct {
	httpClient *http.Client
	policies   []Policy
}

// Options contains configuration options for the HTTP client
type Options struct {
	// Timeout is the maximum time for the entire request
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// RetryDelay is the initial delay between retries (exponential backoff is applied)
	RetryDelay time.Duration

	// Logger is used for debug logging (optional)
	Logger *logging.Logger

	// UserAgent is the User-Agent header value
	UserAgent string

	// Transport allows customizing the underlying HTTP transport
	Transport http.RoundTripper

	// AdditionalPolicies allows adding custom policies
	AdditionalPolicies []Policy
}

// DefaultOptions returns default options for the HTTP client
func DefaultOptions() *Options {
	return &Options{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
		UserAgent:  defaultUserAgent,
	}
}

// NewClient creates a new HTTP client with the given options
func NewClient(opts *Options) *Client {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Create base HTTP client
	httpClient := &http.Client{
		Timeout: opts.Timeout,
	}

	if opts.Transport != nil {
		httpClient.Transport = opts.Transport
	}

	// Build policy chain in order:
	// 1. Error handling (outermost)
	// 2. Retry logic
	// 3. Logging
	// 4. Request ID
	// 5. User Agent
	// 6. Custom policies
	policies := make([]Policy, 0)

	// Error policy (outermost)
	policies = append(policies, NewErrorPolicy())

	// Retry policy
	if opts.MaxRetries > 0 {
		policies = append(policies, NewRetryPolicy(&RetryOptions{
			MaxRetries: opts.MaxRetries,
			RetryDelay: opts.RetryDelay,
			Logger:     opts.Logger,
		}))
	}

	// Request ID policy (must be before logging to see the ID in logs)
	policies = append(policies, NewDefaultRequestIDPolicy())

	// User Agent policy
	if opts.UserAgent != "" {
		policies = append(policies, NewUserAgentPolicy(opts.UserAgent))
	}

	// Logging policy (only if logger is provided)
	// This should be last so it logs after all other policies have modified the request
	if opts.Logger != nil {
		policies = append(policies, NewLoggingPolicy(opts.Logger, &LoggingOptions{
			LogHeaders: true,
			LogBody:    true,
		}))
	}

	// Add custom policies
	if len(opts.AdditionalPolicies) > 0 {
		policies = append(policies, opts.AdditionalPolicies...)
	}

	return &Client{
		httpClient: httpClient,
		policies:   policies,
	}
}

// Do executes an HTTP request through the policy chain
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Build the policy chain from innermost to outermost
	next := func(r *http.Request) (*http.Response, error) {
		return c.httpClient.Do(r)
	}

	// Apply policies in reverse order to build the chain
	for i := len(c.policies) - 1; i >= 0; i-- {
		policy := c.policies[i]
		currentNext := next
		next = func(r *http.Request) (*http.Response, error) {
			return policy.Do(r, currentNext)
		}
	}

	return next(req)
}

// Get is a convenience method for GET requests
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post is a convenience method for POST requests
func (c *Client) Post(ctx context.Context, url, contentType string, body interface{}) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}
