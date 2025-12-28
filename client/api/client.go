package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/httpclient"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// Client provides methods to interact with the Management API
type Client struct {
	baseURL    string
	httpClient *httpclient.Client
}

// Options contains configuration for the API client
type Options struct {
	// BaseURL is the base URL of the Management API
	BaseURL string

	// Timeout is the HTTP request timeout
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// Logger is used for debug logging (optional)
	Logger *logging.Logger
}

// DefaultOptions returns default options for the API client
func DefaultOptions() *Options {
	return &Options{
		BaseURL:    "http://localhost:8080",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}

// NewClient creates a new Management API client
func NewClient(opts *Options) *Client {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Create HTTP client with policies
	httpOpts := &httpclient.Options{
		Timeout:    opts.Timeout,
		MaxRetries: opts.MaxRetries,
		RetryDelay: time.Second,
		Logger:     opts.Logger,
		UserAgent:  "azhexgate-client/1.0",
	}

	return &Client{
		baseURL:    opts.BaseURL,
		httpClient: httpclient.NewClient(httpOpts),
	}
}

// CreateTunnel requests a new tunnel from the Management API
func (c *Client) CreateTunnel(localPort int) (*api.TunnelResponse, error) {
	// Prepare request body
	requestBody := map[string]interface{}{
		"local_port": localPort,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/api/tunnels", c.baseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
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
