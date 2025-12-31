package httpclient

import (
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// RetryPolicy handles retrying failed requests
type RetryPolicy struct {
	maxRetries       int
	retryDelay       time.Duration
	retryStatusCodes []int
	logger           *logging.Logger
}

// RetryOptions contains configuration for RetryPolicy
type RetryOptions struct {
	// MaxRetries is the maximum number of retry attempts (default: 3)
	MaxRetries int

	// RetryDelay is the initial delay between retries (default: 1s)
	RetryDelay time.Duration

	// RetryStatusCodes defines which HTTP status codes should trigger a retry
	// Default: 500-599 (server errors) and 429 (Too Many Requests)
	RetryStatusCodes []int

	// Logger for debug logging (optional)
	Logger *logging.Logger
}

// NewRetryPolicy creates a new RetryPolicy
func NewRetryPolicy(opts *RetryOptions) *RetryPolicy {
	if opts == nil {
		opts = &RetryOptions{}
	}

	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	retryDelay := opts.RetryDelay
	if retryDelay <= 0 {
		retryDelay = time.Second
	}

	// Default retry status codes: 5xx errors and 429
	retryStatusCodes := opts.RetryStatusCodes
	if len(retryStatusCodes) == 0 {
		retryStatusCodes = []int{
			http.StatusTooManyRequests,     // 429
			http.StatusInternalServerError, // 500
			http.StatusBadGateway,          // 502
			http.StatusServiceUnavailable,  // 503
			http.StatusGatewayTimeout,      // 504
		}
	}

	return &RetryPolicy{
		maxRetries:       maxRetries,
		retryDelay:       retryDelay,
		retryStatusCodes: retryStatusCodes,
		logger:           opts.Logger,
	}
}

// Do implements Policy interface
func (p *RetryPolicy) Do(
	req *http.Request,
	next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		// For retries after the first attempt, restore the body if available
		if attempt > 0 && req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
			req.Body = body
		}

		resp, err = next(req)

		// Success or non-retryable error
		if err == nil && !p.shouldRetry(resp) {
			return resp, nil
		}

		// Last attempt - return error
		if attempt == p.maxRetries {
			break
		}

		// Log retry attempt
		if p.logger != nil {
			p.logger.Debug("Retrying request",
				logging.Int("attempt", attempt+1),
				logging.Int("max_retries", p.maxRetries),
				logging.String("url", req.URL.String()))
		}

		// Wait before retry with exponential backoff
		delay := p.retryDelay * time.Duration(1<<attempt)
		time.Sleep(delay)
	}

	return resp, err
}

// shouldRetry determines if a response should be retried
func (p *RetryPolicy) shouldRetry(resp *http.Response) bool {
	if resp == nil {
		return true
	}

	// Check if status code is in the retry list
	for _, code := range p.retryStatusCodes {
		if resp.StatusCode == code {
			return true
		}
	}

	return false
}
