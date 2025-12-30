package cmd

import (
	"context"
	"fmt"

	"github.com/julienstroheker/AzHexGate/client/gateway"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/spf13/cobra"
)

const (
	defaultPort   = 3000
	defaultAPIURL = "http://localhost:8080"
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

		// Create Gateway API client with only overrides
		gatewayClient := gateway.NewClient(&gateway.Options{
			BaseURL: apiURLFlag,
			Logger:  log,
		})

		// Call Gateway API to create tunnel with context
		ctx := context.Background()
		tunnelResp, err := gatewayClient.CreateTunnel(ctx, portFlag)
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
	startCmd.Flags().StringVar(&apiURLFlag, "api-url", defaultAPIURL, "Gateway API base URL")
}
