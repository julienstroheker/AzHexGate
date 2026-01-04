package tunnel

import (
	"context"
	"io"
	"net"

	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/julienstroheker/AzHexGate/internal/relay"
)

// Listener handles incoming connections from the relay and forwards them to localhost
type Listener struct {
	relay     relay.Listener
	localAddr string
}

// Options contains configuration for the Listener
type Options struct {
	// Relay is the relay listener to accept connections from
	Relay relay.Listener

	// LocalAddr is the address of the local HTTP server (e.g., "localhost:3000")
	LocalAddr string
}

// NewListener creates a new tunnel listener
func NewListener(opts *Options) *Listener {
	if opts == nil {
		opts = &Options{}
	}

	return &Listener{
		relay:     opts.Relay,
		localAddr: opts.LocalAddr,
	}
}

// Start begins the listener loop, accepting connections and forwarding requests
func (l *Listener) Start(ctx context.Context, logger *logging.Logger) error {
	if logger != nil {
		logger.Info("Starting listener loop", logging.String("local_addr", l.localAddr))
	}

	for {
		select {
		case <-ctx.Done():
			if logger != nil {
				logger.Info("Listener loop stopped")
			}
			return ctx.Err()
		default:
		}

		// Accept incoming connection from relay
		relayConn, err := l.relay.Accept(ctx, logger)
		if err != nil {
			if ctx.Err() != nil {
				// Context cancelled, stop gracefully
				return ctx.Err()
			}
			if logger != nil {
				logger.Error("Failed to accept connection", logging.Error(err))
			}
			continue
		}

		// Handle connection in a separate goroutine
		go l.handleConnection(ctx, relayConn, logger)
	}
}

// handleConnection processes a single relay connection by establishing a TCP connection
// to the local server and bidirectionally copying data between them
func (l *Listener) handleConnection(ctx context.Context, relayConn relay.Connection, logger *logging.Logger) {
	defer func() {
		_ = relayConn.Close()
	}()

	if logger != nil {
		logger.Debug("Handling new connection")
	}

	// Dial the local TCP server
	var dialer net.Dialer
	localConn, err := dialer.DialContext(ctx, "tcp", l.localAddr)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to dial local server", logging.Error(err))
		}
		return
	}
	defer func() {
		_ = localConn.Close()
	}()

	if logger != nil {
		logger.Debug("Connected to local server")
	}

	// Bidirectional copy between relay and local server
	done := make(chan error, 2)

	// Copy from relay to local server
	go func() {
		_, err := io.Copy(localConn, relayConn)
		done <- err
	}()

	// Copy from local server to relay
	go func() {
		_, err := io.Copy(relayConn, localConn)
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
	_ = localConn.Close()

	// Wait for the other goroutine to finish
	<-done
}

// Close closes the listener
func (l *Listener) Close() error {
	if l.relay != nil {
		return l.relay.Close()
	}
	return nil
}
