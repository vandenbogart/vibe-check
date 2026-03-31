package cmd

import (
	"fmt"

	"github.com/ebo/vibe-check/internal/client"
	"github.com/ebo/vibe-check/internal/config"
	"github.com/spf13/cobra"
)

var allowCmd = &cobra.Command{
	Use:   "allow <registry> <package> <version>",
	Short: "Allow a specific package version to bypass the age check",
	Args:  cobra.ExactArgs(3),
	RunE:  runAllow,
}

func init() {
	allowCmd.Flags().String("data-dir", config.DefaultDataDir(), "path to the vibe-check data directory")
}

func runAllow(cmd *cobra.Command, args []string) error {
	registry, pkg, version := args[0], args[1], args[2]

	if registry != "pypi" && registry != "npm" {
		return fmt.Errorf("unsupported registry %q (must be \"pypi\" or \"npm\")", registry)
	}

	dataDir, err := cmd.Flags().GetString("data-dir")
	if err != nil {
		return err
	}

	c := client.New(dataDir)
	if err := c.Allow(registry, pkg, version); err != nil {
		return err
	}

	fmt.Printf("Allowed: %s %s %s\n", registry, pkg, version)
	return nil
}
