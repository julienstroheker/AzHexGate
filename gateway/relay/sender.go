package relay

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"

	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/julienstroheker/AzHexGate/internal/relay"
)

// Sender handles outgoing connections to the relay and forwards HTTP requests
type Sender struct {
	relay relay.Sender
}

// Options contains configuration for the Sender
type Options struct {
	// Relay is the relay sender to create connections with
	Relay relay.Sender
}

// NewSender creates a new relay sender
func NewSender(opts *Options) *Sender {
	if opts == nil {
		opts = &Options{}
	}

	return &Sender{
		relay: opts.Relay,
	}
}

// ForwardRequest forwards an HTTP request through the relay and writes the response to the provided writer
// This method handles the complete request/response cycle by:
// 1. Dialing the relay to establish a connection
// 2. Writing the HTTP request to the relay
// 3. Reading and parsing the HTTP response
// 4. Copying the response to the provided ResponseWriter
func (s *Sender) ForwardRequest(ctx context.Context, req *http.Request, w http.ResponseWriter, logger *logging.Logger) error {
	if logger != nil {
		logger.Debug("Forwarding request through relay", logging.String("method", req.Method), logging.String("path", req.URL.Path))
	}

	// Dial the relay to create a connection
	relayConn, err := s.relay.Dial(ctx)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to dial relay", logging.Error(err))
		}
		return err
	}
	defer func() {
		_ = relayConn.Close()
	}()

	if logger != nil {
		logger.Debug("Connected to relay")
	}

	// Write the HTTP request to the relay connection
	if err := req.Write(relayConn); err != nil {
		if logger != nil {
			logger.Error("Failed to write request to relay", logging.Error(err))
		}
		return err
	}

	if logger != nil {
		logger.Debug("Request written to relay")
	}

	// Read the HTTP response from the relay
	resp, err := http.ReadResponse(bufio.NewReader(relayConn), req)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to read response from relay", logging.Error(err))
		}
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if logger != nil {
		logger.Debug("Response received from relay", logging.Int("status", resp.StatusCode))
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil && logger != nil {
		logger.Error("Failed to copy response body", logging.Error(err))
	}

	return err
}

// ForwardRequestRaw forwards an HTTP request through the relay using raw TCP connection
// This method provides bidirectional streaming between the caller and relay
func (s *Sender) ForwardRequestRaw(ctx context.Context, clientConn net.Conn, logger *logging.Logger) error {
	if logger != nil {
		logger.Debug("Forwarding raw request through relay")
	}

	// Dial the relay to create a connection
	relayConn, err := s.relay.Dial(ctx)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to dial relay", logging.Error(err))
		}
		return err
	}
	defer func() {
		_ = relayConn.Close()
	}()

	if logger != nil {
		logger.Debug("Connected to relay for raw forwarding")
	}

	// Bidirectional copy between client and relay
	done := make(chan error, 2)

	// Copy from client to relay
	go func() {
		_, err := io.Copy(relayConn, clientConn)
		done <- err
	}()

	// Copy from relay to client
	go func() {
		_, err := io.Copy(clientConn, relayConn)
		done <- err
	}()

	// Wait for one direction to complete
	err = <-done

	if logger != nil {
		if err != nil && err != io.EOF {
			logger.Debug("Connection closed with error", logging.Error(err))
		} else {
			logger.Debug("Connection completed successfully")
		}
	}

	// Close both connections to terminate the other goroutine
	_ = relayConn.Close()
	_ = clientConn.Close()

	// Wait for the other goroutine to finish
	<-done

	return nil
}

// Close closes the sender
func (s *Sender) Close() error {
	if s.relay != nil {
		return s.relay.Close()
	}
	return nil
}
