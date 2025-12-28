package cmd

import (
	"fmt"
	"os"

	"github.com/julienstroheker/AzHexGate/internal/config"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/spf13/cobra"
)

var (
	cfg         *config.Config
	logger      *logging.Logger
	verboseFlag bool
	jsonFlag    bool
)

var rootCmd = &cobra.Command{
	Use:   "gateway",
	Short: "AzHexGate Cloud Gateway server",
	Long:  `gateway - AzHexGate Cloud Gateway server for handling tunnel traffic`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg = config.Load()

		// Determine log level
		level := logging.ParseLevel(cfg.LogLevel)
		if verboseFlag {
			level = logging.DebugLevel
		}

		// Determine format
		format := logging.FormatConsole
		if jsonFlag {
			format = logging.FormatJSON
		}

		// Initialize logger
		logger = logging.NewWithFormat(level, format)
		logger.Info("Logger initialized",
			logging.String("level", level.String()),
			logging.String("format", map[logging.Format]string{
				logging.FormatConsole: "console",
				logging.FormatJSON:    "json",
			}[format]),
		)
	},
}

func init() {
	// Disable default completion and help commands
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// Add persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose logging (debug level)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output logs in JSON format")
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

// GetConfig returns the global config instance
func GetConfig() *config.Config {
	return cfg
}
