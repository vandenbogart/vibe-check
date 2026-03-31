package cmd

import (
	"fmt"

	"github.com/vandenbogart/vibe-check/internal/client"
	"github.com/vandenbogart/vibe-check/internal/config"
	"github.com/spf13/cobra"
)

var setMinAgeCmd = &cobra.Command{
	Use:   "set-min-age <duration>",
	Short: "Update the minimum package age at runtime",
	Args:  cobra.ExactArgs(1),
	RunE:  runSetMinAge,
}

func init() {
	setMinAgeCmd.Flags().String("data-dir", config.DefaultDataDir(), "path to the vibe-check data directory")
}

func runSetMinAge(cmd *cobra.Command, args []string) error {
	dataDir, err := cmd.Flags().GetString("data-dir")
	if err != nil {
		return err
	}

	c := client.New(dataDir)
	if err := c.SetMinAge(args[0]); err != nil {
		return err
	}

	fmt.Printf("Updated min-age to %s\n", args[0])
	return nil
}
