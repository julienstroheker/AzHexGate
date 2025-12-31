package relay

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
)

// MockConnection is an in-memory implementation of Connection for testing
type MockConnection struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

// NewMockConnection creates a new mock connection
func NewMockConnection() *MockConnection {
	return &MockConnection{
		readBuf:  &bytes.Buffer{},
		writeBuf: &bytes.Buffer{},
	}
}

// Read reads data from the connection's read buffer
func (c *MockConnection) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.EOF
	}

	return c.readBuf.Read(p)
}

// Write writes data to the connection's write buffer
func (c *MockConnection) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, errors.New("connection closed")
	}

	return c.writeBuf.Write(p)
}

// Close closes the connection
func (c *MockConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	return nil
}

// WriteToReadBuffer writes data to the read buffer (simulates incoming data)
func (c *MockConnection) WriteToReadBuffer(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readBuf.Write(data)
}

// GetWrittenData returns data written to the connection
func (c *MockConnection) GetWrittenData() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeBuf.Bytes()
}

// MockListener is an in-memory implementation of Listener for testing
type MockListener struct {
	addr        string
	connections chan Connection
	closed      bool
	mu          sync.Mutex
}

// NewMockListener creates a new mock listener
func NewMockListener(addr string) *MockListener {
	return &MockListener{
		addr:        addr,
		connections: make(chan Connection, 10),
	}
}

// Accept waits for and returns the next connection
func (l *MockListener) Accept(ctx context.Context) (Connection, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn, ok := <-l.connections:
		if !ok {
			return nil, errors.New("listener closed")
		}
		return conn, nil
	}
}

// Close closes the listener
func (l *MockListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	close(l.connections)
	return nil
}

// Addr returns the listener's address
func (l *MockListener) Addr() string {
	return l.addr
}

// SendConnection sends a connection to the listener (simulates incoming connection)
func (l *MockListener) SendConnection(conn Connection) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return errors.New("listener closed")
	}

	l.connections <- conn
	return nil
}

// MockSender is an in-memory implementation of Sender for testing
type MockSender struct {
	listener *MockListener
	closed   bool
	mu       sync.Mutex
}

// NewMockSender creates a new mock sender that connects to the given listener
func NewMockSender(listener *MockListener) *MockSender {
	return &MockSender{
		listener: listener,
	}
}

// Dial establishes a connection to the listener
func (s *MockSender) Dial(ctx context.Context) (Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, errors.New("sender closed")
	}

	// Create a connection pair
	conn := NewMockConnection()

	// Send the connection to the listener
	if err := s.listener.SendConnection(conn); err != nil {
		return nil, err
	}

	return conn, nil
}

// Close closes the sender
func (s *MockSender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	return nil
}
