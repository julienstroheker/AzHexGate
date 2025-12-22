package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julienstroheker/AzHexGate/gateway/http"
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
		return runServer(cmd)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().IntVarP(&portFlag, "port", "p", defaultPort, "Port to listen on")
	startCmd.Flags().IntVar(&shutdownTimeoutFlag, "shutdown-timeout", defaultShutdownTimeout,
		"Graceful shutdown timeout in seconds")
}

func runServer(cmd *cobra.Command) error {
	cmd.Printf("Starting gateway server on port %d...\n", portFlag)

	// Create server
	server := http.NewServer(portFlag)

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	// Start the server
	go func() {
		cmd.Printf("Gateway listening on :%d\n", portFlag)
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
		cmd.Printf("\nReceived signal %v, starting graceful shutdown...\n", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeoutFlag)*time.Second)
		defer cancel()

		// Asking listener to shut down and shed load.
		if err := server.Shutdown(ctx); err != nil {
			if err := server.Close(); err != nil {
				return fmt.Errorf("could not stop server gracefully: %w", err)
			}
			return fmt.Errorf("could not gracefully shutdown the server: %w", err)
		}

		cmd.Println("Server stopped gracefully")
	}

	return nil
}
