package relay

import (
	"context"
	"io"
)

// Connection represents a bidirectional stream connection through Azure Relay
type Connection interface {
	// Read reads data from the connection
	io.Reader

	// Write writes data to the connection
	io.Writer

	// Close closes the connection
	io.Closer
}

// Listener represents an Azure Relay Hybrid Connection Listener
// The client uses this to accept incoming connections from the gateway
type Listener interface {
	// Accept waits for and returns the next connection to the listener
	Accept(ctx context.Context) (Connection, error)

	// Close closes the listener
	// Any blocked Accept operations will be unblocked and return errors
	Close() error

	// Addr returns the listener's network address
	Addr() string
}

// Sender represents an Azure Relay Hybrid Connection Sender
// The gateway uses this to establish connections to the client
type Sender interface {
	// Dial establishes a connection to the listener
	Dial(ctx context.Context) (Connection, error)

	// Close closes the sender
	Close() error
}
