package relay

import (
	"context"
	"io"
)

// Connection represents a bidirectional stream between sender and listener
type Connection interface {
	io.ReadWriteCloser
}

// Listener represents a relay listener that accepts incoming connections
type Listener interface {
	// Accept waits for and returns the next connection to the listener
	Accept(ctx context.Context) (Connection, error)

	// Close closes the listener
	Close() error
}

// Sender represents a relay sender that creates connections to a listener
type Sender interface {
	// Dial creates a new connection to the listener
	Dial(ctx context.Context) (Connection, error)

	// Close closes the sender
	Close() error
}
