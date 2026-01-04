package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// AzureListener is a real Azure Relay Hybrid Connection listener
type AzureListener struct {
	relayEndpoint        string
	hybridConnectionName string
	token                string
	listenerID           string
	controlConn          *websocket.Conn
	mu                   sync.Mutex
	closed               bool
	acceptQueue          chan Connection
	logger               *logging.Logger
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
		listenerID:           uuid.New().String(),
		acceptQueue:          make(chan Connection, 10),
	}, nil
}

// connect establishes the control channel WebSocket connection to Azure Relay
func (l *AzureListener) connect(ctx context.Context, logger *logging.Logger) error {
	if logger != nil {
		logger.Debug("Connecting to Azure Relay control channel",
			logging.String("relay_endpoint", l.relayEndpoint),
			logging.String("hybrid_connection_name", l.hybridConnectionName),
			logging.String("listener_id", l.listenerID))
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return ErrListenerClosed
	}

	// Build control channel WebSocket URL
	// Format: wss://<endpoint>/$hc/<name>?sb-hc-action=listen&sb-hc-id=<listener-id>
	wsURL := fmt.Sprintf("wss://%s/$hc/%s?sb-hc-action=listen&sb-hc-id=%s",
		l.relayEndpoint, l.hybridConnectionName, l.listenerID)

	if logger != nil {
		logger.Debug("Control channel URL", logging.String("url", wsURL))
	}

	// Add token in ServiceBusAuthorization header (not query string)
	header := http.Header{}
	header.Add("ServiceBusAuthorization", l.token)

	// Set up WebSocket dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	// Connect to Azure Relay control channel
	conn, resp, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		if resp != nil {
			body := make([]byte, 512)
			n, _ := resp.Body.Read(body)
			_ = resp.Body.Close()
			if logger != nil && n > 0 {
				logger.Error("Azure Relay connection failed",
					logging.Int("status", resp.StatusCode),
					logging.String("body", string(body[:n])))
			}
			return fmt.Errorf("failed to connect to relay control channel (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("failed to connect to relay control channel: %w", err)
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if logger != nil {
		logger.Info("✓ Control channel connected successfully")
	}

	l.controlConn = conn
	l.logger = logger
	return nil
}

// Accept waits for and returns the next connection to the listener
func (l *AzureListener) Accept(ctx context.Context, logger *logging.Logger) (Connection, error) {
	if logger != nil {
		logger.Debug("Waiting to accept new connection", logging.String("relay_endpoint", l.relayEndpoint), logging.String("hybrid_connection_name", l.hybridConnectionName))
	}
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, ErrListenerClosed
	}

	// Connect if not already connected
	if l.controlConn == nil {
		l.mu.Unlock()
		if err := l.connect(ctx, logger); err != nil {
			return nil, err
		}
		// Start background goroutine to handle accept messages from control channel
		go l.handleControlChannel()
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

// acceptMessage represents an accept notification from the control channel
type acceptMessage struct {
	Accept struct {
		Address        string            `json:"address"`
		ID             string            `json:"id"`
		ConnectHeaders map[string]string `json:"connectHeaders"`
	} `json:"accept"`
}

// handleControlChannel processes accept messages from the Azure Relay control channel
func (l *AzureListener) handleControlChannel() {
	if l.logger != nil {
		l.logger.Debug("Control channel handler started")
	}

	for {
		l.mu.Lock()
		if l.closed || l.controlConn == nil {
			l.mu.Unlock()
			return
		}
		conn := l.controlConn
		l.mu.Unlock()

		// Read message from control channel
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if l.logger != nil {
				l.logger.Error("Control channel read error", logging.Error(err))
			}
			// Connection error, close and return
			_ = l.Close()
			return
		}

		// Control channel uses JSON text messages
		if messageType == websocket.TextMessage {
			if l.logger != nil {
				l.logger.Debug("Received control channel message", logging.String("message", string(data)))
			}

			// Parse accept message
			var msg acceptMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				if l.logger != nil {
					l.logger.Error("Failed to parse accept message", logging.Error(err))
				}
				continue
			}

			if msg.Accept.Address != "" {
				if l.logger != nil {
					l.logger.Info("Received accept notification",
						logging.String("rendezvous_address", msg.Accept.Address),
						logging.String("connection_id", msg.Accept.ID))
				}

				// Establish rendezvous connection
				go l.acceptRendezvousConnection(msg.Accept.Address, msg.Accept.ID)
			}
		}
	}
}

// acceptRendezvousConnection establishes the rendezvous connection to handle the actual data transfer
func (l *AzureListener) acceptRendezvousConnection(rendezvousAddress, connectionID string) {
	if l.logger != nil {
		l.logger.Debug("Establishing rendezvous connection",
			logging.String("address", rendezvousAddress),
			logging.String("connection_id", connectionID))
	}

	// Connect to the rendezvous address (no authentication needed for rendezvous)
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, resp, err := dialer.DialContext(context.Background(), rendezvousAddress, http.Header{})
	if err != nil {
		if l.logger != nil {
			body := make([]byte, 512)
			if resp != nil && resp.Body != nil {
				n, _ := resp.Body.Read(body)
				_ = resp.Body.Close()
				if n > 0 {
					l.logger.Error("Rendezvous connection failed",
						logging.Int("status", resp.StatusCode),
						logging.String("body", string(body[:n])),
						logging.Error(err))
				}
			}
		}
		return
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if l.logger != nil {
		l.logger.Info("✓ Rendezvous connection established", logging.String("connection_id", connectionID))
	}

	// Create connection wrapper
	azureConn := &azureConnection{
		conn:   conn,
		buffer: nil,
	}

	// Try to add to accept queue
	select {
	case l.acceptQueue <- azureConn:
		// Successfully queued
	default:
		// Queue full - close the connection to signal backpressure
		if l.logger != nil {
			l.logger.Warn("Accept queue full, dropping connection")
		}
		_ = azureConn.Close()
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

	if l.controlConn != nil {
		return l.controlConn.Close()
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
