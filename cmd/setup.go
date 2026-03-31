package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure pip/npm to use vibe-check as a proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("setup: not yet implemented")
		return nil
	},
}

func init() {
	setupCmd.Flags().IntP("pypi-port", "", 3141, "Port for the PyPI proxy")
	setupCmd.Flags().IntP("npm-port", "", 3142, "Port for the npm proxy")
	setupCmd.Flags().StringP("min-age", "", "7d", "Minimum package age before allowing installation")
	setupCmd.Flags().StringP("data-dir", "", "", "Directory for persistent data")
	setupCmd.Flags().StringP("log-level", "", "info", "Log level (debug, info, warn, error)")
}
