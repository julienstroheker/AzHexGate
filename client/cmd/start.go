package cmd

import (
	"context"
	"fmt"

	"github.com/julienstroheker/AzHexGate/client/gateway"
	"github.com/julienstroheker/AzHexGate/internal/config"
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
	modeFlag   string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the tunnel and forward traffic to localhost",
	Long:  `Start the tunnel and forward traffic to localhost`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()

		// Parse and validate mode
		mode := config.Mode(modeFlag)
		if !mode.IsValid() {
			return fmt.Errorf("invalid mode: %s (must be 'local' or 'remote')", modeFlag)
		}

		log.Info("Starting tunnel",
			logging.Int("port", portFlag),
			logging.String("mode", mode.String()))

		// Create Gateway API client with mode
		gatewayClient := gateway.NewClient(&gateway.Options{
			BaseURL: apiURLFlag,
			Logger:  log,
			Mode:    mode,
		})

		// Call Gateway API to create tunnel
		ctx := context.Background()
		tunnelResp, err := gatewayClient.CreateTunnel(ctx, log, portFlag)
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

		// Start listening for connections (both local and remote modes)
		if mode == config.ModeLocal {
			log.Info("Starting listener in local mode")
			if err := gatewayClient.StartListening(ctx, log, portFlag, tunnelResp); err != nil {
				return fmt.Errorf("listener error: %w", err)
			}
		} else {
			log.Info("Remote mode - listener not yet implemented")
			// In remote mode, we would start the Azure Relay listener here
			// For now, just return after printing the URL
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().IntVarP(&portFlag, "port", "p", defaultPort, "Local port to forward traffic to")
	startCmd.Flags().StringVar(&apiURLFlag, "api-url", defaultAPIURL, "Gateway API base URL")
	startCmd.Flags().StringVar(&modeFlag, "mode", string(config.ModeRemote),
		"Operation mode: local or remote")
}
