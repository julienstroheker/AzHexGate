package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/config"
	"github.com/julienstroheker/AzHexGate/internal/httpclient"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/julienstroheker/AzHexGate/internal/relay"
)

// Client provides methods to interact with the Gateway API
type Client struct {
	baseURL    string
	httpClient *httpclient.Client
	logger     *logging.Logger
	mode       config.Mode
	listener   relay.Listener
}

// Options contains configuration for the Gateway API client
type Options struct {
	// BaseURL is the base URL of the Gateway API (optional, defaults to http://localhost:8080)
	BaseURL string

	// Timeout is the HTTP request timeout (optional, defaults to 30s)
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts (optional, defaults to 3)
	MaxRetries int

	// Logger is used for debug logging (optional)
	Logger *logging.Logger

	// Mode is the operational mode (optional, defaults to remote)
	Mode config.Mode
}

// NewClient creates a new Gateway API client
func NewClient(opts *Options) *Client {
	if opts == nil {
		opts = &Options{}
	}

	// Set defaults
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxRetries := opts.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	mode := opts.Mode
	if mode == config.Mode("") {
		mode = config.ModeRemote
	}

	// Create HTTP client with policies
	httpOpts := &httpclient.Options{
		Timeout:    timeout,
		MaxRetries: maxRetries,
		RetryDelay: time.Second,
		Logger:     opts.Logger,
		UserAgent:  "azhexgate-client/1.0",
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpclient.NewClient(httpOpts),
		logger:     opts.Logger,
		mode:       mode,
	}
}

// CreateTunnelRequest represents the request to create a new tunnel
type CreateTunnelRequest struct {
	LocalPort int `json:"local_port"`
}

// CreateTunnel requests a new tunnel from the Gateway API
func (c *Client) CreateTunnel(ctx context.Context, localPort int) (*api.TunnelResponse, error) {
	switch c.mode {
	case config.ModeLocal:
		return c.createLocalTunnel(ctx, localPort)
	case config.ModeRemote:
		return c.createRemoteTunnel(ctx, localPort)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", c.mode)
	}
}

// createLocalTunnel creates a local in-memory tunnel
func (c *Client) createLocalTunnel(_ context.Context, localPort int) (*api.TunnelResponse, error) {
	// Log entry
	if c.logger != nil {
		c.logger.Info("Creating local tunnel", logging.Int("local_port", localPort))
	}

	// Create in-memory listener
	hcName := fmt.Sprintf("hc-%s", uuid.New().String()[:8])
	c.listener = relay.NewMockListener(hcName)

	// TODO: Register with shared registry (for gateway to access)

	return &api.TunnelResponse{
		PublicURL:            fmt.Sprintf("http://localhost:8080/tunnel/%s", hcName),
		RelayEndpoint:        "in-memory",
		HybridConnectionName: hcName,
		ListenerToken:        "local-mode-token",
		SessionID:            fmt.Sprintf("session-%s", uuid.New().String()[:8]),
	}, nil
}

// createRemoteTunnel requests a tunnel from the remote Gateway API
func (c *Client) createRemoteTunnel(ctx context.Context, localPort int) (*api.TunnelResponse, error) {
	// Log entry
	if c.logger != nil {
		c.logger.Info("Creating tunnel", logging.Int("local_port", localPort))
	}

	// Prepare request body
	requestBody := CreateTunnelRequest{
		LocalPort: localPort,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request with context and bytes.NewReader to allow retries
	url := fmt.Sprintf("%s/api/tunnels", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set GetBody to allow retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request through HTTP client with policies
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err // Error is already wrapped by ErrorPolicy
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tunnelResp api.TunnelResponse
	if err := json.NewDecoder(resp.Body).Decode(&tunnelResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tunnelResp, nil
}

// StartListening starts listening for connections (both local and remote modes)
func (c *Client) StartListening(ctx context.Context, localPort int) error {
	if c.mode == config.ModeLocal {
		return c.startLocalListening(ctx, localPort)
	}

	// Remote mode listening
	// TODO: Implement Azure Relay listener
	if c.logger != nil {
		c.logger.Info("Remote mode listening not yet implemented")
	}
	return fmt.Errorf("remote mode listening not yet implemented")
}

// startLocalListening listens for local mode connections
func (c *Client) startLocalListening(ctx context.Context, localPort int) error {
	if c.listener == nil {
		return fmt.Errorf("no listener available; call CreateTunnel first")
	}

	if c.logger != nil {
		c.logger.Info("Starting local listener", logging.Int("local_port", localPort))
	}

	for {
		conn, err := c.listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}

		go c.handleConnection(conn, localPort)
	}
}

// handleConnection handles an individual connection
func (c *Client) handleConnection(conn relay.Connection, localPort int) {
	defer func() {
		_ = conn.Close()
	}()

	// TODO: Implement HTTP request parsing and forwarding to localhost:localPort
	if c.logger != nil {
		c.logger.Debug("Handling connection", logging.Int("local_port", localPort))
	}
}
