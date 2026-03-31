package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the proxy server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("serve: not yet implemented")
		return nil
	},
}

func init() {
	serveCmd.Flags().IntP("pypi-port", "", 3141, "Port for the PyPI proxy")
	serveCmd.Flags().IntP("npm-port", "", 3142, "Port for the npm proxy")
	serveCmd.Flags().StringP("min-age", "", "7d", "Minimum package age before allowing installation")
	serveCmd.Flags().StringP("data-dir", "", "", "Directory for persistent data")
	serveCmd.Flags().StringP("log-level", "", "info", "Log level (debug, info, warn, error)")
}
