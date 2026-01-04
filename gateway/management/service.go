package management

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/azure/relay"
)

// Service handles tunnel provisioning and token generation
type Service struct {
	relayNamespace string
	relayKeyName   string
	relayKey       string
	baseDomain     string
}

// Options contains configuration for the Management Service
type Options struct {
	// RelayNamespace is the Azure Relay namespace name (e.g., "myrelay")
	RelayNamespace string

	// RelayKeyName is the name of the shared access key
	RelayKeyName string

	// RelayKey is the shared access key value (base64 encoded)
	RelayKey string

	// BaseDomain is the base domain for public URLs (e.g., "azhexgate.com")
	BaseDomain string
}

// NewService creates a new Management Service
func NewService(opts *Options) (*Service, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if opts.RelayNamespace == "" {
		return nil, fmt.Errorf("relay namespace is required")
	}
	if opts.RelayKeyName == "" {
		return nil, fmt.Errorf("relay key name is required")
	}
	if opts.RelayKey == "" {
		return nil, fmt.Errorf("relay key is required")
	}
	if opts.BaseDomain == "" {
		opts.BaseDomain = "azhexgate.com"
	}

	return &Service{
		relayNamespace: opts.RelayNamespace,
		relayKeyName:   opts.RelayKeyName,
		relayKey:       opts.RelayKey,
		baseDomain:     opts.BaseDomain,
	}, nil
}

// CreateTunnel provisions a new tunnel and generates credentials
func (s *Service) CreateTunnel(localPort int) (*api.TunnelResponse, error) {
	// Generate a unique subdomain ID
	subdomainID := generateSubdomainID()

	// Derive hybrid connection name
	hybridConnectionName := fmt.Sprintf("hc-%s", subdomainID)

	// Generate listener SAS token (valid for 24 hours)
	listenerToken, err := relay.GenerateListenerSASToken(
		s.relayNamespace,
		hybridConnectionName,
		s.relayKeyName,
		s.relayKey,
		24*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate listener token: %w", err)
	}

	// Build public URL
	publicURL := fmt.Sprintf("https://%s.%s", subdomainID, s.baseDomain)

	// Build relay endpoint
	relayEndpoint := fmt.Sprintf("%s.servicebus.windows.net", s.relayNamespace)

	// Generate session ID
	sessionID := uuid.New().String()

	return &api.TunnelResponse{
		PublicURL:            publicURL,
		RelayEndpoint:        relayEndpoint,
		HybridConnectionName: hybridConnectionName,
		ListenerToken:        listenerToken,
		SessionID:            sessionID,
	}, nil
}

// generateSubdomainID generates a random 8-digit subdomain identifier
func generateSubdomainID() string {
	id := uuid.New().String()
	// Take first 8 characters of the UUID (hex format)
	return id[:8]
}
