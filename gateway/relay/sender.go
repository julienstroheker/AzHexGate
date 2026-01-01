package relay

import (
	"context"
	"io"
	"net"

	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/julienstroheker/AzHexGate/internal/relay"
)

// Sender handles outgoing connections to the relay and forwards traffic
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

// ForwardRequestRaw forwards traffic through the relay using raw TCP connection
// This method provides bidirectional streaming between the client and relay,
// maintaining a transparent tunnel that mirrors the behavior of client/tunnel/listener.go
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

	// Return the error unless it's EOF (which is normal termination)
	if err == io.EOF {
		return nil
	}
	return err
}

// Close closes the sender
func (s *Sender) Close() error {
	if s.relay != nil {
		return s.relay.Close()
	}
	return nil
}
