package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julienstroheker/AzHexGate/gateway/http"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/spf13/cobra"
)

const (
	defaultPort            = 8080
	defaultShutdownTimeout = 30
)

var (
	portFlag            int
	shutdownTimeoutFlag int
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the gateway HTTP server",
	Long:  `Start the gateway HTTP server with health check endpoint`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().IntVarP(&portFlag, "port", "p", defaultPort, "Port to listen on")
	startCmd.Flags().IntVar(&shutdownTimeoutFlag, "shutdown-timeout", defaultShutdownTimeout,
		"Graceful shutdown timeout in seconds")
}

func runServer() error {
	log := GetLogger()
	log.Info("Starting gateway server", logging.Int("port", portFlag))

	// Create server
	server := http.NewServer(portFlag)

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	// Start the server
	go func() {
		log.Info("Gateway listening", logging.Int("port", portFlag))
		serverErrors <- server.ListenAndServe()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking select
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info("Received shutdown signal", logging.String("signal", sig.String()))

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeoutFlag)*time.Second)
		defer cancel()

		// Asking listener to shut down and shed load.
		if err := server.Shutdown(ctx); err != nil {
			// If graceful shutdown fails, try to force close
			shutdownErr := fmt.Errorf("could not gracefully shutdown the server: %w", err)
			if closeErr := server.Close(); closeErr != nil {
				return fmt.Errorf("%v; also failed to force close: %w", shutdownErr, closeErr)
			}
			return shutdownErr
		}

		log.Info("Server stopped gracefully")
	}

	return nil
}
