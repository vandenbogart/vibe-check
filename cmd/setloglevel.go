package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var setLogLevelCmd = &cobra.Command{
	Use:   "set-log-level <level>",
	Short: "Update the log level at runtime",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("set-log-level: not yet implemented")
		return nil
	},
}
