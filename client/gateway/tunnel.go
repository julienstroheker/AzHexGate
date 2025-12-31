package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/config"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// CreateTunnelRequest represents the request to create a new tunnel
type CreateTunnelRequest struct {
	LocalPort int `json:"local_port"`
}

// CreateTunnel requests a new tunnel from the Gateway API
func (c *Client) CreateTunnel(ctx context.Context, logger *logging.Logger, localPort int) (*api.TunnelResponse, error) {
	switch c.mode {
	case config.ModeLocal:
		return c.createLocalTunnel(ctx, logger, localPort)
	case config.ModeRemote:
		return c.createRemoteTunnel(ctx, logger, localPort)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", c.mode)
	}
}

// createLocalTunnel creates a local in-memory tunnel
func (c *Client) createLocalTunnel(ctx context.Context, logger *logging.Logger, localPort int) (*api.TunnelResponse, error) {
	// In local mode, we still call the gateway API to create the tunnel
	// This ensures the gateway creates the MockListener and stores it in its registry
	// The client will then connect to that listener via HTTP
	if logger != nil {
		logger.Info("Creating local tunnel via gateway API", logging.Int("local_port", localPort))
	}

	// Call the gateway API (same as remote mode)
	return c.createRemoteTunnel(ctx, logger, localPort)
}

// createRemoteTunnel requests a tunnel from the remote Gateway API
func (c *Client) createRemoteTunnel(ctx context.Context, logger *logging.Logger, localPort int) (*api.TunnelResponse, error) {
	// Log entry
	if logger != nil {
		logger.Info("Creating tunnel", logging.Int("local_port", localPort))
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
