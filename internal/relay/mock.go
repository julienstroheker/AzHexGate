package relay

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/julienstroheker/AzHexGate/internal/logging"
)

var (
	// ErrListenerClosed is returned when trying to accept on a closed listener
	ErrListenerClosed = errors.New("listener is closed")
	// ErrSenderClosed is returned when trying to dial with a closed sender
	ErrSenderClosed = errors.New("sender is closed")
	// ErrConnectionClosed is returned when trying to read/write on a closed connection
	ErrConnectionClosed = errors.New("connection is closed")
)

// memoryConnection represents an in-memory bidirectional pipe
type memoryConnection struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	mu     sync.Mutex
	closed bool
}

// Read reads data from the connection
func (c *memoryConnection) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, ErrConnectionClosed
	}
	c.mu.Unlock()
	return c.reader.Read(p)
}

// Write writes data to the connection
func (c *memoryConnection) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, ErrConnectionClosed
	}
	c.mu.Unlock()
	return c.writer.Write(p)
}

// Close closes the connection
func (c *memoryConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	_ = c.reader.Close()
	_ = c.writer.Close()
	return nil
}

// MemoryListener is an in-memory implementation of Listener for testing
type MemoryListener struct {
	connections chan Connection
	mu          sync.Mutex
	closed      bool
}

// NewMemoryListener creates a new in-memory listener
func NewMemoryListener() *MemoryListener {
	return &MemoryListener{
		connections: make(chan Connection, 10),
	}
}

// Accept waits for and returns the next connection
func (l *MemoryListener) Accept(ctx context.Context, logger *logging.Logger) (Connection, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, ErrListenerClosed
	}
	l.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn, ok := <-l.connections:
		if !ok {
			return nil, ErrListenerClosed
		}
		return conn, nil
	}
}

// Close closes the listener
func (l *MemoryListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	close(l.connections)
	return nil
}

// addConnection adds a connection to the listener (for testing)
func (l *MemoryListener) addConnection(conn Connection) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrListenerClosed
	}
	l.connections <- conn
	return nil
}

// MemorySender is an in-memory implementation of Sender for testing
type MemorySender struct {
	listener *MemoryListener
	mu       sync.Mutex
	closed   bool
}

// NewMemorySender creates a new in-memory sender connected to the given listener
func NewMemorySender(listener *MemoryListener) *MemorySender {
	return &MemorySender{
		listener: listener,
	}
}

// Dial creates a new connection to the listener
func (s *MemorySender) Dial(ctx context.Context) (Connection, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrSenderClosed
	}
	s.mu.Unlock()

	// Create two pipes for bidirectional communication
	// Pipe 1: sender writes -> listener reads
	listenerReader, senderWriter := io.Pipe()
	// Pipe 2: listener writes -> sender reads
	senderReader, listenerWriter := io.Pipe()

	// Sender side connection
	senderConn := &memoryConnection{
		reader: senderReader,
		writer: senderWriter,
	}

	// Listener side connection
	listenerConn := &memoryConnection{
		reader: listenerReader,
		writer: listenerWriter,
	}

	// Add the listener side to the listener's accept queue
	if err := s.listener.addConnection(listenerConn); err != nil {
		_ = senderConn.Close()
		_ = listenerConn.Close()
		return nil, err
	}

	// Return the sender side to the caller
	return senderConn, nil
}

// Close closes the sender
func (s *MemorySender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return nil
}
