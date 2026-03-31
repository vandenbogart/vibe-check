package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/ebo/vibe-check/internal/config"
	"github.com/ebo/vibe-check/internal/platform"
	"github.com/ebo/vibe-check/internal/setup"
	"github.com/spf13/cobra"
)

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Remove vibe-check proxy configuration from pip/npm",
	RunE:  runTeardown,
}

func init() {
	teardownCmd.Flags().StringP("data-dir", "", "", "Directory for persistent data")
}

func runTeardown(cmd *cobra.Command, args []string) error {
	// 1. Get data-dir, load config to know ports
	dataDir, _ := cmd.Flags().GetString("data-dir")
	if dataDir == "" {
		dataDir = config.DefaultDataDir()
	}

	cfgPath := filepath.Join(dataDir, "config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// 2. Detect and restore package manager configs from backups
	managers := setup.DetectPackageManagers(cfg.PyPIPort, cfg.NPMPort)
	backupDir := filepath.Join(dataDir, "backups")
	restored := setup.RestoreConfigs(managers, backupDir)
	for _, name := range restored {
		fmt.Printf("✓ Restored %s config\n", name)
	}

	// 3. Stop and uninstall platform service (handle errors gracefully)
	svc := platform.Detect()
	if svc != nil {
		if err := svc.Stop(); err != nil {
			fmt.Printf("⚠ Warning: stopping service: %v\n", err)
		} else {
			fmt.Println("✓ Stopped vibe-check daemon")
		}

		if err := svc.Uninstall(); err != nil {
			fmt.Printf("⚠ Warning: removing service: %v\n", err)
		} else {
			fmt.Println("✓ Removed platform service")
		}
	}

	// 4. Print summary
	fmt.Println("\nvibe-check has been removed. Package managers restored to original configuration.")
	return nil
}
