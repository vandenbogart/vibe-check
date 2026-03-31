package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vibe-check",
	Short: "Package registry proxy that blocks recently published packages",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(allowCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(teardownCmd)
	rootCmd.AddCommand(setMinAgeCmd)
	rootCmd.AddCommand(setLogLevelCmd)
}
