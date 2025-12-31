package tunnel

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/config"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/julienstroheker/AzHexGate/internal/relay"
)

// Manager manages tunnel creation and lifecycle
type Manager struct {
	mode config.Mode
	// Local mode: in-memory registry
	mu        sync.RWMutex
	listeners map[string]relay.Listener

	// Remote mode: Azure Relay credentials
	// TODO: Add Azure Relay client fields when implementing remote mode
}

// Options configures the tunnel manager
type Options struct {
	Mode config.Mode
}

// NewManager creates a new tunnel manager
func NewManager(opts *Options) *Manager {
	if opts == nil {
		opts = &Options{
			Mode: config.ModeRemote,
		}
	}

	return &Manager{
		mode:      opts.Mode,
		listeners: make(map[string]relay.Listener),
	}
}

// CreateTunnel creates a new tunnel based on the configured mode
func (m *Manager) CreateTunnel(ctx context.Context, logger *logging.Logger, localPort int) (*api.TunnelResponse, error) {
	switch m.mode {
	case config.ModeLocal:
		return m.createLocalTunnel(ctx, logger, localPort)
	case config.ModeRemote:
		return m.createRemoteTunnel(ctx, logger, localPort)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", m.mode)
	}
}

// createLocalTunnel creates an in-memory tunnel for local development
func (m *Manager) createLocalTunnel(_ context.Context, logger *logging.Logger, localPort int) (*api.TunnelResponse, error) {
	// Generate a unique hybrid connection name
	hcName := fmt.Sprintf("hc-%s", uuid.New().String()[:8])

	if logger != nil {
		logger.Info("Creating local tunnel",
			logging.String("hc_name", hcName),
			logging.Int("local_port", localPort))
	}

	// Create mock listener
	listener := relay.NewMockListener(hcName)

	// Store in registry
	m.mu.Lock()
	m.listeners[hcName] = listener
	m.mu.Unlock()

	// Return tunnel metadata
	return &api.TunnelResponse{
		PublicURL:            fmt.Sprintf("http://localhost:8080/tunnel/%s", hcName),
		RelayEndpoint:        "in-memory",
		HybridConnectionName: hcName,
		ListenerToken:        "local-mode-token",
		SessionID:            fmt.Sprintf("session-%s", uuid.New().String()[:8]),
	}, nil
}

// createRemoteTunnel creates a tunnel using Azure Relay (placeholder)
func (m *Manager) createRemoteTunnel(_ context.Context, logger *logging.Logger, localPort int) (*api.TunnelResponse, error) {
	// TODO: Implement Azure Relay integration
	// This is a placeholder that returns mock data for now
	if logger != nil {
		logger.Info("Creating remote tunnel (placeholder)",
			logging.Int("local_port", localPort))
	}

	return &api.TunnelResponse{
		PublicURL:            "https://63873749.azhexgate.com",
		RelayEndpoint:        "https://azhexgate-relay.servicebus.windows.net",
		HybridConnectionName: "hc-63873749",
		ListenerToken:        "mock-listener-token",
		SessionID:            "mock-session-id",
	}, nil
}

// GetListener retrieves a listener by hybrid connection name (local mode only)
func (m *Manager) GetListener(hcName string) (relay.Listener, error) {
	if m.mode != config.ModeLocal {
		return nil, fmt.Errorf("GetListener only available in local mode")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	listener, ok := m.listeners[hcName]
	if !ok {
		return nil, fmt.Errorf("listener not found: %s", hcName)
	}

	return listener, nil
}

// GetSender creates a sender for the given hybrid connection (local mode only)
func (m *Manager) GetSender(hcName string) (relay.Sender, error) {
	if m.mode != config.ModeLocal {
		return nil, fmt.Errorf("GetSender only available in local mode")
	}

	listener, err := m.GetListener(hcName)
	if err != nil {
		return nil, err
	}

	mockListener, ok := listener.(*relay.MockListener)
	if !ok {
		return nil, fmt.Errorf("listener is not a MockListener")
	}

	return relay.NewMockSender(mockListener), nil
}
