package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// LoggingPolicy logs requests and responses in debug mode
type LoggingPolicy struct {
	logger        *logging.Logger
	logHeaders    bool
	logBody       bool
	redactBody    bool
	headerFilters []string // Headers to redact
}

// LoggingOptions contains configuration for LoggingPolicy
type LoggingOptions struct {
	// LogHeaders enables logging of all request/response headers
	LogHeaders bool

	// LogBody enables logging of request/response body
	LogBody bool

	// RedactBody redacts the body content (shows only size)
	RedactBody bool

	// HeaderFilters is a list of header names to redact values for (e.g., "Authorization")
	HeaderFilters []string
}

// NewLoggingPolicy creates a new LoggingPolicy
func NewLoggingPolicy(logger *logging.Logger, opts *LoggingOptions) *LoggingPolicy {
	if opts == nil {
		opts = &LoggingOptions{}
	}

	return &LoggingPolicy{
		logger:        logger,
		logHeaders:    opts.LogHeaders,
		logBody:       opts.LogBody,
		redactBody:    opts.RedactBody,
		headerFilters: opts.HeaderFilters,
	}
}

// Do implements Policy interface
func (p *LoggingPolicy) Do(
	req *http.Request,
	next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
	// Log request
	p.logRequest(req)

	start := time.Now()
	resp, err := next(req)
	duration := time.Since(start)

	// Log response or error
	if err != nil {
		p.logRequestFailure(req, err, duration)
		return resp, err
	}

	p.logResponse(req, resp, duration)
	return resp, nil
}

// logRequest logs the outgoing request
func (p *LoggingPolicy) logRequest(req *http.Request) {
	fields := []logging.Field{
		logging.String("method", req.Method),
		logging.String("url", req.URL.String()),
	}

	if p.logHeaders && len(req.Header) > 0 {
		fields = append(fields, p.formatHeaders("request_headers", req.Header))
	}

	if p.logBody && req.Body != nil {
		bodyContent, err := p.readAndRestoreBody(req)
		if err == nil {
			fields = append(fields, p.formatBody("request_body", bodyContent))
		}
	}

	p.logger.Debug("HTTP Request", fields...)
}

// logRequestFailure logs a failed request
func (p *LoggingPolicy) logRequestFailure(req *http.Request, err error, duration time.Duration) {
	p.logger.Debug("HTTP Request failed",
		logging.String("method", req.Method),
		logging.String("url", req.URL.String()),
		logging.String("error", err.Error()),
		logging.Int("duration_ms", int(duration.Milliseconds())))
}

// logResponse logs the response
func (p *LoggingPolicy) logResponse(req *http.Request, resp *http.Response, duration time.Duration) {
	respFields := []logging.Field{
		logging.String("method", req.Method),
		logging.String("url", req.URL.String()),
		logging.Int("status", resp.StatusCode),
		logging.Int("duration_ms", int(duration.Milliseconds())),
	}

	if p.logHeaders && len(resp.Header) > 0 {
		respFields = append(respFields, p.formatHeaders("response_headers", resp.Header))
	}

	if p.logBody && resp.Body != nil {
		bodyContent, err := p.readAndRestoreResponseBody(resp)
		if err == nil {
			respFields = append(respFields, p.formatBody("response_body", bodyContent))
		}
	}

	p.logger.Debug("HTTP Response", respFields...)
}

// formatBody formats body content for logging
func (p *LoggingPolicy) formatBody(key string, bodyContent []byte) logging.Field {
	if p.redactBody {
		return logging.String(key+"_size", fmt.Sprintf("%d bytes", len(bodyContent)))
	}
	return logging.String(key, string(bodyContent))
}

// formatHeaders formats headers for logging, applying redaction filters
func (p *LoggingPolicy) formatHeaders(key string, headers http.Header) logging.Field {
	var headerStrings []string
	for name, values := range headers {
		value := strings.Join(values, ", ")

		// Check if this header should be redacted
		shouldRedact := false
		for _, filter := range p.headerFilters {
			if strings.EqualFold(name, filter) {
				shouldRedact = true
				break
			}
		}

		if shouldRedact {
			value = "[REDACTED]"
		}

		headerStrings = append(headerStrings, fmt.Sprintf("%s: %s", name, value))
	}
	return logging.String(key, strings.Join(headerStrings, "; "))
}

// readAndRestoreBody reads the request body and restores it
func (p *LoggingPolicy) readAndRestoreBody(req *http.Request) ([]byte, error) {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	// Restore the body for the next policy
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}

// readAndRestoreResponseBody reads the response body and restores it
func (p *LoggingPolicy) readAndRestoreResponseBody(resp *http.Response) ([]byte, error) {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Restore the body for the caller
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}
