package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/julienstroheker/AzHexGate/client/gateway"
	"github.com/julienstroheker/AzHexGate/client/tunnel"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/julienstroheker/AzHexGate/internal/relay"
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

		// Get context from command (supports timeout in tests)
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Create Gateway API client with only overrides
		gatewayClient := gateway.NewClient(&gateway.Options{
			BaseURL: apiURLFlag,
			Logger:  log,
		})

		// Call Gateway API to create tunnel with context
		tunnelResp, err := gatewayClient.CreateTunnel(ctx, portFlag)
		if err != nil {
			return fmt.Errorf("failed to create tunnel: %w", err)
		}

		// Print the public URL
		cmd.Println("Tunnel established")
		cmd.Println(fmt.Sprintf("Public URL: %s", tunnelResp.PublicURL))
		cmd.Println(fmt.Sprintf("Forwarding to: http://localhost:%d", portFlag))

		log.Info("Tunnel created, preparing to start listener",
			logging.String("public_url", tunnelResp.PublicURL),
			logging.String("session_id", tunnelResp.SessionID))

		// TODO: In production, create Azure Relay listener using tunnelResp.RelayEndpoint,
		// tunnelResp.HybridConnectionName, and tunnelResp.ListenerToken
		// For now, create an in-memory relay listener for testing
		relayListener := relay.NewMemoryListener()
		defer func() { _ = relayListener.Close() }()

		// Create tunnel listener
		localAddr := fmt.Sprintf("localhost:%d", portFlag)
		tunnelListener := tunnel.NewListener(&tunnel.Options{
			Relay:     relayListener,
			LocalAddr: localAddr,
			Logger:    log,
		})
		defer func() { _ = tunnelListener.Close() }()

		// Start the listener loop in a goroutine
		errChan := make(chan error, 1)
		go func() {
			if err := tunnelListener.Start(ctx); err != nil && err != context.Canceled {
				errChan <- err
			}
		}()

		log.Info("Listener loop started, waiting for connections...")

		// Wait for interrupt signal or context cancellation
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)

		select {
		case <-ctx.Done():
			log.Info("Context cancelled, shutting down...")
			return ctx.Err()
		case <-sigChan:
			log.Info("Received interrupt signal, shutting down...")
			cancel()
		case err := <-errChan:
			log.Error("Listener error", logging.Error(err))
			cancel()
			return err
		}

		log.Info("Tunnel closed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().IntVarP(&portFlag, "port", "p", defaultPort, "Local port to forward traffic to")
	startCmd.Flags().StringVar(&apiURLFlag, "api-url", defaultAPIURL, "Gateway API base URL")
}
