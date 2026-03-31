package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var setMinAgeCmd = &cobra.Command{
	Use:   "set-min-age <duration>",
	Short: "Update the minimum package age at runtime",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("set-min-age: not yet implemented")
		return nil
	},
}
