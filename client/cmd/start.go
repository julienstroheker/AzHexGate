package cmd

import (
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/spf13/cobra"
)

const defaultPort = 3000

var portFlag int

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the tunnel and forward traffic to localhost",
	Long:  `Start the tunnel and forward traffic to localhost`,
	Run: func(cmd *cobra.Command, args []string) {
		log := GetLogger()
		log.Info("Starting tunnel", logging.Int("port", portFlag))
		log.Debug("Tunnel logic not yet implemented")
		cmd.Println("Tunnel logic not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().IntVarP(&portFlag, "port", "p", defaultPort, "Local port to forward traffic to")
}
