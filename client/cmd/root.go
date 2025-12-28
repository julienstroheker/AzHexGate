package cmd

import (
	"fmt"
	"os"

	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/spf13/cobra"
)

var (
	logger      *logging.Logger
	verboseFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "azhexgate",
	Short: "Azure Hybrid Connection reverse tunnel",
	Long:  `azhexgate - Azure Hybrid Connection reverse tunnel`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logger based on verbose flag
		level := logging.InfoLevel
		if verboseFlag {
			level = logging.DebugLevel
		}
		logger = logging.New(level)
		logger.Info("Logger initialized", logging.String("level", level.String()))
	},
}

func init() {
	// Disable default completion and help commands
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// Add persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose logging (debug level)")
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// GetLogger returns the global logger instance
func GetLogger() *logging.Logger {
	return logger
}
