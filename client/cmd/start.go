package cmd

import (
	"fmt"
	"time"

	clientapi "github.com/julienstroheker/AzHexGate/client/api"
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

		// Create API client
		apiClient := clientapi.NewClient(&clientapi.Options{
			BaseURL:    apiURLFlag,
			Timeout:    defaultAPITimeout,
			MaxRetries: 3,
			Logger:     log,
		})

		// Call Management API to create tunnel
		tunnelResp, err := apiClient.CreateTunnel(portFlag)
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
