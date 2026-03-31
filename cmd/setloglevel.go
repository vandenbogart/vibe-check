package cmd

import (
	"fmt"

	"github.com/vandenbogart/vibe-check/internal/client"
	"github.com/vandenbogart/vibe-check/internal/config"
	"github.com/spf13/cobra"
)

var setLogLevelCmd = &cobra.Command{
	Use:   "set-log-level <level>",
	Short: "Update the log level at runtime",
	Args:  cobra.ExactArgs(1),
	RunE:  runSetLogLevel,
}

func init() {
	setLogLevelCmd.Flags().String("data-dir", config.DefaultDataDir(), "path to the vibe-check data directory")
}

func runSetLogLevel(cmd *cobra.Command, args []string) error {
	dataDir, err := cmd.Flags().GetString("data-dir")
	if err != nil {
		return err
	}

	c := client.New(dataDir)
	if err := c.SetLogLevel(args[0]); err != nil {
		return err
	}

	fmt.Printf("Updated log level to %s\n", args[0])
	return nil
}
