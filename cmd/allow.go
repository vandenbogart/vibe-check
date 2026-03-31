package cmd

import (
	"fmt"

	"github.com/vandenbogart/vibe-check/internal/client"
	"github.com/vandenbogart/vibe-check/internal/config"
	"github.com/vandenbogart/vibe-check/internal/resolve"
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
	allowCmd.Flags().Bool("exact", false, "allow only this exact package without resolving dependencies")
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

	exact, err := cmd.Flags().GetBool("exact")
	if err != nil {
		return err
	}

	c := client.New(dataDir)

	if exact {
		if err := c.Allow(registry, pkg, version); err != nil {
			return err
		}
		fmt.Printf("Allowed: %s %s %s\n", registry, pkg, version)
		return nil
	}

	// Resolve full dependency tree.
	fmt.Println("Resolving dependency tree...")

	var packages []resolve.Package
	switch registry {
	case "npm":
		packages, err = resolve.NPMDeps(pkg, version)
	case "pypi":
		packages, err = resolve.PyPIDeps(pkg, version)
	}
	if err != nil {
		return fmt.Errorf("resolving dependencies: %w", err)
	}

	if len(packages) == 0 {
		return fmt.Errorf("no packages resolved for %s %s@%s", registry, pkg, version)
	}

	// Allow each resolved package.
	for _, p := range packages {
		if err := c.Allow(registry, p.Name, p.Version); err != nil {
			return fmt.Errorf("allowing %s@%s: %w", p.Name, p.Version, err)
		}
	}

	fmt.Printf("Allowed %d packages:\n", len(packages))
	for _, p := range packages {
		fmt.Printf("  %s@%s\n", p.Name, p.Version)
	}
	return nil
}
