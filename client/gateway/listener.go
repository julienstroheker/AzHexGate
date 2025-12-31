package gateway

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/config"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// StartListening starts listening for connections (both local and remote modes)
func (c *Client) StartListening(ctx context.Context, logger *logging.Logger, localPort int, tunnelResp *api.TunnelResponse) error {
	if c.mode == config.ModeLocal {
		return c.startLocalListening(ctx, logger, localPort, tunnelResp)
	}

	// Remote mode listening
	// TODO: Implement Azure Relay listener
	if logger != nil {
		logger.Info("Remote mode listening not yet implemented")
	}
	return fmt.Errorf("remote mode listening not yet implemented")
}

// startLocalListening listens for local mode connections via gateway HTTP endpoint
func (c *Client) startLocalListening(ctx context.Context, logger *logging.Logger, localPort int, tunnelResp *api.TunnelResponse) error {
	if logger != nil {
		logger.Info("Starting local listener",
			logging.Int("local_port", localPort),
			logging.String("hc_name", tunnelResp.HybridConnectionName))
	}

	// In local mode, we continuously poll the gateway's listener endpoint
	// This mimics Azure Relay's listener behavior where the client accepts connections
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Accept one connection from the gateway
			if err := c.acceptOneConnection(ctx, logger, localPort, tunnelResp.HybridConnectionName); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// Log error but continue listening
				if logger != nil {
					logger.Error("Failed to accept connection", logging.Error(err))
				}
				// Brief pause before retrying
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// acceptOneConnection accepts and handles one connection from the gateway
func (c *Client) acceptOneConnection(ctx context.Context, logger *logging.Logger, localPort int, hcName string) error {
	// Make a request to the gateway's listener endpoint
	url := fmt.Sprintf("%s/internal/listen/%s", c.baseURL, hcName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create listen request: %w", err)
	}

	if logger != nil {
		logger.Debug("Connecting to gateway listener", logging.String("url", url))
	}

	// Use a raw HTTP client without retry policies for listener connections
	client := &http.Client{
		Timeout: 0, // No timeout for long-lived connections
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to gateway listener: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gateway listener returned status %d: %s", resp.StatusCode, string(body))
	}

	if logger != nil {
		logger.Debug("Accepted connection from gateway")
	}

	// Read the HTTP request from the response body
	reader := bufio.NewReader(resp.Body)
	req, err = http.ReadRequest(reader)
	if err != nil {
		return fmt.Errorf("failed to read request from gateway: %w", err)
	}

	// Handle the connection
	c.handleConnection(req, resp.Body, logger, localPort)

	return nil
}

// handleConnection handles an individual connection by forwarding to localhost
func (c *Client) handleConnection(req *http.Request, connReader io.Reader, logger *logging.Logger, localPort int) {
	if logger != nil {
		logger.Info("Handling request",
			logging.String("method", req.Method),
			logging.String("path", req.URL.Path),
			logging.Int("local_port", localPort))
	}

	// Update the request URL to point to localhost
	req.URL.Scheme = "http"
	req.URL.Host = fmt.Sprintf("localhost:%d", localPort)
	req.RequestURI = "" // Must be empty for client requests

	// Forward the request to localhost
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to forward request to localhost", logging.Error(err))
		}
		// Send error response back
		// In this implementation, we can't write back through the connection
		// since the HTTP protocol doesn't support it cleanly
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if logger != nil {
		logger.Info("Received response from localhost",
			logging.Int("status", resp.StatusCode))
	}

	// Note: In this simplified implementation, we can't write the response back
	// through the HTTP connection. A production implementation would need:
	// 1. WebSocket for bidirectional communication
	// 2. HTTP/2 bidirectional streaming
	// 3. Custom protocol over HTTP
	// For now, the response is lost, but this demonstrates the flow
}
