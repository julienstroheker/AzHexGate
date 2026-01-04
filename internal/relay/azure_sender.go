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

// AzureSender is a real Azure Relay Hybrid Connection sender
type AzureSender struct {
	relayEndpoint        string
	hybridConnectionName string
	token                string
	mu                   sync.Mutex
	closed               bool
}

// AzureSenderOptions contains configuration for Azure Relay Sender
type AzureSenderOptions struct {
	RelayEndpoint        string // e.g., "myrelay.servicebus.windows.net"
	HybridConnectionName string // e.g., "hc-12345"
	Token                string // SAS token or Azure AD token for authentication
}

// NewAzureSender creates a new Azure Relay Hybrid Connection sender
func NewAzureSender(opts *AzureSenderOptions) (*AzureSender, error) {
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

	return &AzureSender{
		relayEndpoint:        opts.RelayEndpoint,
		hybridConnectionName: opts.HybridConnectionName,
		token:                opts.Token,
	}, nil
}

// Dial creates a new connection to the listener
func (s *AzureSender) Dial(ctx context.Context) (Connection, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrSenderClosed
	}
	s.mu.Unlock()

	// Build the WebSocket URL for sender
	// Format: wss://<endpoint>/$hc/<name>?sb-hc-action=connect&sb-hc-token=<token>
	wsURL := fmt.Sprintf("wss://%s/$hc/%s", s.relayEndpoint, s.hybridConnectionName)

	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	q := u.Query()
	q.Set("sb-hc-action", "connect")
	
	// Check if token looks like a SAS token or Azure AD token
	// SAS tokens start with "SharedAccessSignature"
	// Azure AD tokens are just the token value
	if len(s.token) > 0 {
		q.Set("sb-hc-token", s.token)
	}
	
	u.RawQuery = q.Encode()

	// Set up WebSocket dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	// For Azure AD tokens, set as Authorization header
	headers := http.Header{}
	// Azure AD tokens don't start with "SharedAccessSignature"
	// If it's not a SAS token, assume it's Azure AD and use Authorization header
	if len(s.token) > 0 && s.token[:20] != "SharedAccessSignature" {
		headers.Set("Authorization", "Bearer "+s.token)
	}

	// Connect to Azure Relay
	conn, resp, err := dialer.DialContext(ctx, u.String(), headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("failed to connect to relay (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to connect to relay: %w", err)
	}

	return &azureConnection{
		conn: conn,
	}, nil
}

// Close closes the sender
func (s *AzureSender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return nil
}

var _ Sender = (*AzureSender)(nil)
