package sender

import (
	"bufio"
	"context"
	"io"
	"net/http"

	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/julienstroheker/AzHexGate/internal/relay"
)

// Sender handles forwarding HTTP requests through the relay to the local client
type Sender struct {
	relay relay.Sender
}

// Options contains configuration for the Sender
type Options struct {
	// Relay is the relay sender to dial connections through
	Relay relay.Sender
}

// NewSender creates a new sender
func NewSender(opts *Options) *Sender {
	if opts == nil {
		opts = &Options{}
	}

	return &Sender{
		relay: opts.Relay,
	}
}

// Forward forwards an HTTP request through the relay and returns the response
func (s *Sender) Forward(ctx context.Context, req *http.Request, logger *logging.Logger) (*http.Response, error) {
	if logger != nil {
		logger.Debug("Forwarding request through relay",
			logging.String("method", req.Method),
			logging.String("path", req.URL.Path))
	}

	// Dial the relay to establish a connection
	relayConn, err := s.relay.Dial(ctx)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to dial relay", logging.Error(err))
		}
		return nil, err
	}

	if logger != nil {
		logger.Debug("Connected to relay")
	}

	// Write the HTTP request to the relay connection
	if err := req.Write(relayConn); err != nil {
		_ = relayConn.Close()
		if logger != nil {
			logger.Error("Failed to write request to relay", logging.Error(err))
		}
		return nil, err
	}

	if logger != nil {
		logger.Debug("Request written to relay, reading response")
	}

	// Read the HTTP response from the relay connection
	resp, err := http.ReadResponse(bufio.NewReader(relayConn), req)
	if err != nil {
		_ = relayConn.Close()
		if logger != nil {
			logger.Error("Failed to read response from relay", logging.Error(err))
		}
		return nil, err
	}

	if logger != nil {
		logger.Debug("Response received from relay", logging.Int("status", resp.StatusCode))
	}

	// Wrap the response body so that closing it also closes the relay connection
	resp.Body = &responseBodyCloser{
		ReadCloser: resp.Body,
		conn:       relayConn,
	}

	return resp, nil
}

// responseBodyCloser wraps the response body and ensures the relay connection is closed
type responseBodyCloser struct {
	io.ReadCloser
	conn relay.Connection
}

func (r *responseBodyCloser) Close() error {
	// Close the body first
	bodyErr := r.ReadCloser.Close()
	// Then close the connection
	connErr := r.conn.Close()

	// Return the first error encountered
	if bodyErr != nil {
		return bodyErr
	}
	return connErr
}

// Close closes the sender
func (s *Sender) Close() error {
	if s.relay != nil {
		return s.relay.Close()
	}
	return nil
}
