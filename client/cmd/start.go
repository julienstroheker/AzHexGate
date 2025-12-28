package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/spf13/cobra"
)

const (
	defaultPort       = 3000
	defaultAPIURL     = "http://localhost:8080"
	defaultAPITimeout = 30 * time.Second
)

var (
	portFlag   int
	apiURLFlag string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the tunnel and forward traffic to localhost",
	Long:  `Start the tunnel and forward traffic to localhost`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		log.Info("Starting tunnel", logging.Int("port", portFlag))

		// Call Management API to create tunnel
		tunnelResp, err := createTunnel(apiURLFlag, portFlag)
		if err != nil {
			return fmt.Errorf("failed to create tunnel: %w", err)
		}

		// Print the public URL
		cmd.Println("Tunnel established")
		cmd.Println(fmt.Sprintf("Public URL: %s", tunnelResp.PublicURL))
		cmd.Println(fmt.Sprintf("Forwarding to: http://localhost:%d", portFlag))

		log.Info("Tunnel created successfully",
			logging.String("public_url", tunnelResp.PublicURL),
			logging.String("session_id", tunnelResp.SessionID))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().IntVarP(&portFlag, "port", "p", defaultPort, "Local port to forward traffic to")
	startCmd.Flags().StringVar(&apiURLFlag, "api-url", defaultAPIURL, "Management API base URL")
}

// createTunnel calls the Management API to create a new tunnel
func createTunnel(apiURL string, localPort int) (*api.TunnelResponse, error) {
	// Prepare request body
	requestBody := map[string]interface{}{
		"local_port": localPort,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: defaultAPITimeout,
	}

	// Make POST request to /api/tunnels
	url := fmt.Sprintf("%s/api/tunnels", apiURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
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
