package httpclient

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/julienstroheker/AzHexGate/internal/logging"
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

// LoggingPolicy logs requests and responses in debug mode
type LoggingPolicy struct {
	logger *logging.Logger
}

// NewLoggingPolicy creates a new LoggingPolicy
func NewLoggingPolicy(logger *logging.Logger) *LoggingPolicy {
	return &LoggingPolicy{logger: logger}
}

// Do implements Policy interface
func (p *LoggingPolicy) Do(
	req *http.Request,
	next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
	if p.logger == nil {
		return next(req)
	}

	// Log request
	p.logger.Debug("HTTP Request",
		logging.String("method", req.Method),
		logging.String("url", req.URL.String()),
		logging.String("request_id", req.Header.Get("X-Client-Request-Id")))

	start := time.Now()
	resp, err := next(req)
	duration := time.Since(start)

	if err != nil {
		p.logger.Debug("HTTP Request failed",
			logging.String("method", req.Method),
			logging.String("url", req.URL.String()),
			logging.String("error", err.Error()),
			logging.Int("duration_ms", int(duration.Milliseconds())))
		return resp, err
	}

	// Log response
	p.logger.Debug("HTTP Response",
		logging.String("method", req.Method),
		logging.String("url", req.URL.String()),
		logging.Int("status", resp.StatusCode),
		logging.Int("duration_ms", int(duration.Milliseconds())))

	return resp, nil
}

// RetryPolicy handles retrying failed requests
type RetryPolicy struct {
	maxRetries int
	retryDelay time.Duration
}

// NewRetryPolicy creates a new RetryPolicy
func NewRetryPolicy(maxRetries int, retryDelay time.Duration) *RetryPolicy {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if retryDelay <= 0 {
		retryDelay = time.Second
	}
	return &RetryPolicy{
		maxRetries: maxRetries,
		retryDelay: retryDelay,
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
		resp, err = next(req)

		// Success or non-retryable error
		if err == nil && !shouldRetry(resp) {
			return resp, nil
		}

		// Last attempt - return error
		if attempt == p.maxRetries {
			break
		}

		// Wait before retry with exponential backoff
		delay := p.retryDelay * time.Duration(1<<attempt)
		time.Sleep(delay)
	}

	return resp, err
}

// shouldRetry determines if a response should be retried
func shouldRetry(resp *http.Response) bool {
	if resp == nil {
		return true
	}

	// Retry on 5xx errors and 429 (Too Many Requests)
	return resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests
}

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
