package relay

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// AzureListener is a real Azure Relay Hybrid Connection listener
type AzureListener struct {
	relayEndpoint        string
	hybridConnectionName string
	token                string
	conn                 *websocket.Conn
	mu                   sync.Mutex
	closed               bool
	acceptQueue          chan Connection
}

// AzureListenerOptions contains configuration for Azure Relay Listener
type AzureListenerOptions struct {
	RelayEndpoint        string // e.g., "myrelay.servicebus.windows.net"
	HybridConnectionName string // e.g., "hc-12345"
	Token                string // SAS token for authentication
}

// NewAzureListener creates a new Azure Relay Hybrid Connection listener
func NewAzureListener(opts *AzureListenerOptions) (*AzureListener, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if opts.RelayEndpoint == "" {
		return nil, fmt.Errorf("relay endpoint is required")
	}
	if opts.HybridConnectionName == "" {
		return nil, fmt.Errorf("hybrid connection name is required")
	}
	if opts.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	return &AzureListener{
		relayEndpoint:        opts.RelayEndpoint,
		hybridConnectionName: opts.HybridConnectionName,
		token:                opts.Token,
		acceptQueue:          make(chan Connection, 10),
	}, nil
}

// connect establishes the WebSocket connection to Azure Relay
func (l *AzureListener) connect(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return ErrListenerClosed
	}

	// Build the WebSocket URL for listener
	// Format: wss://<endpoint>/$hc/<name>?sb-hc-action=listen&sb-hc-token=<token>
	wsURL := fmt.Sprintf("wss://%s/$hc/%s", l.relayEndpoint, l.hybridConnectionName)

	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	q := u.Query()
	q.Set("sb-hc-action", "listen")
	q.Set("sb-hc-token", l.token)
	u.RawQuery = q.Encode()

	// Set up WebSocket dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	// Connect to Azure Relay
	conn, resp, err := dialer.DialContext(ctx, u.String(), http.Header{})
	if err != nil {
		if resp != nil {
			_ = resp.Body.Close()
			return fmt.Errorf("failed to connect to relay (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("failed to connect to relay: %w", err)
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	l.conn = conn
	return nil
}

// Accept waits for and returns the next connection to the listener
func (l *AzureListener) Accept(ctx context.Context) (Connection, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, ErrListenerClosed
	}

	// Connect if not already connected
	if l.conn == nil {
		l.mu.Unlock()
		if err := l.connect(ctx); err != nil {
			return nil, err
		}
		// Start background goroutine to handle incoming connections
		go l.handleIncomingConnections()
		l.mu.Lock()
	}
	l.mu.Unlock()

	// Wait for a connection from the queue
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn, ok := <-l.acceptQueue:
		if !ok {
			return nil, ErrListenerClosed
		}
		return conn, nil
	}
}

// handleIncomingConnections processes incoming connection requests from Azure Relay
func (l *AzureListener) handleIncomingConnections() {
	for {
		l.mu.Lock()
		if l.closed || l.conn == nil {
			l.mu.Unlock()
			return
		}
		conn := l.conn
		l.mu.Unlock()

		// Read message from relay
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			// Connection error, close and return
			_ = l.Close()
			return
		}

		// For hybrid connections, we expect binary messages
		if messageType == websocket.BinaryMessage {
			// Create a new connection wrapper
			azureConn := &azureConnection{
				conn:   conn,
				buffer: data,
			}

			// Add to accept queue
			select {
			case l.acceptQueue <- azureConn:
			default:
				// Queue full, drop connection
			}
		}
	}
}

// Close closes the listener
func (l *AzureListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	close(l.acceptQueue)

	if l.conn != nil {
		return l.conn.Close()
	}

	return nil
}

// azureConnection wraps a WebSocket connection for Azure Relay
type azureConnection struct {
	conn   *websocket.Conn
	buffer []byte
	mu     sync.Mutex
	closed bool
}

// Read reads data from the connection
func (c *azureConnection) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, ErrConnectionClosed
	}

	// If we have buffered data, read from it first
	if len(c.buffer) > 0 {
		n = copy(p, c.buffer)
		c.buffer = c.buffer[n:]
		c.mu.Unlock()
		return n, nil
	}
	c.mu.Unlock()

	// Read next message from WebSocket
	messageType, data, err := c.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	if messageType != websocket.BinaryMessage {
		return 0, fmt.Errorf("unexpected message type: %d", messageType)
	}

	// Copy data to output buffer
	n = copy(p, data)

	// Buffer any remaining data
	if n < len(data) {
		c.mu.Lock()
		c.buffer = data[n:]
		c.mu.Unlock()
	}

	return n, nil
}

// Write writes data to the connection
func (c *azureConnection) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, ErrConnectionClosed
	}
	c.mu.Unlock()

	// Write as binary message
	if err := c.conn.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close closes the connection
func (c *azureConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.conn.Close()
}

var _ Connection = (*azureConnection)(nil)
var _ Listener = (*AzureListener)(nil)
