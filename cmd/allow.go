package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var allowCmd = &cobra.Command{
	Use:   "allow <registry> <package> <version>",
	Short: "Allow a specific package version to bypass the age check",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("allow: not yet implemented")
		return nil
	},
}
