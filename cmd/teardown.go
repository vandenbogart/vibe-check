package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Remove vibe-check proxy configuration from pip/npm",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("teardown: not yet implemented")
		return nil
	},
}

func init() {
	teardownCmd.Flags().StringP("data-dir", "", "", "Directory for persistent data")
}
