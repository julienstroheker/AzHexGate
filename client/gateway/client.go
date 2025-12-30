package gateway

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

// Client provides methods to interact with the Gateway API
type Client struct {
	baseURL    string
	httpClient *httpclient.Client
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
	}
}

// CreateTunnel requests a new tunnel from the Gateway API
func (c *Client) CreateTunnel(localPort int) (*api.TunnelResponse, error) {
	// Prepare request body
	requestBody := map[string]interface{}{
		"local_port": localPort,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request with bytes.NewReader to allow retries
	url := fmt.Sprintf("%s/api/tunnels", c.baseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
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
