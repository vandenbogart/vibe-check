package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/vandenbogart/vibe-check/internal/config"
	"github.com/vandenbogart/vibe-check/internal/platform"
	"github.com/vandenbogart/vibe-check/internal/setup"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure pip/npm to use vibe-check as a proxy",
	RunE:  runSetup,
}

func init() {
	setupCmd.Flags().IntP("pypi-port", "", 3141, "Port for the PyPI proxy")
	setupCmd.Flags().IntP("npm-port", "", 3142, "Port for the npm proxy")
	setupCmd.Flags().StringP("min-age", "", "7d", "Minimum package age before allowing installation")
	setupCmd.Flags().StringP("data-dir", "", "", "Directory for persistent data")
	setupCmd.Flags().StringP("log-level", "", "info", "Log level: trace, debug, info")
}

func runSetup(cmd *cobra.Command, args []string) error {
	// 1. Parse flags
	pypiPort, _ := cmd.Flags().GetInt("pypi-port")
	npmPort, _ := cmd.Flags().GetInt("npm-port")
	minAgeStr, _ := cmd.Flags().GetString("min-age")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	logLevel, _ := cmd.Flags().GetString("log-level")

	if dataDir == "" {
		dataDir = config.DefaultDataDir()
	}

	// 2. Validate min-age
	minAge, err := config.ParseMinAge(minAgeStr)
	if err != nil {
		return fmt.Errorf("invalid min-age: %w", err)
	}

	// 3. Create data dir, save config.json
	cfg := config.Config{
		PyPIPort: pypiPort,
		NPMPort:  npmPort,
		MinAge:   minAge,
		MinAgeS:  minAgeStr,
	}
	cfgPath := filepath.Join(dataDir, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("✓ Saved config to %s\n", cfgPath)

	// 4. Find binary path
	binPath, err := platform.FindBinary()
	if err != nil {
		return fmt.Errorf("finding binary: %w", err)
	}

	// 5. Detect platform
	svc := platform.Detect()
	if svc == nil {
		return fmt.Errorf("unsupported platform — cannot install service")
	}

	// 6. Install platform service
	opts := platform.ServiceOpts{
		BinaryPath: binPath,
		DataDir:    dataDir,
		PyPIPort:   pypiPort,
		NPMPort:    npmPort,
		MinAge:     minAgeStr,
		LogLevel:   logLevel,
	}
	if err := svc.Install(opts); err != nil {
		return fmt.Errorf("installing service: %w", err)
	}
	fmt.Println("✓ Installed platform service")

	// 7. Start the service
	if err := svc.Start(); err != nil {
		return fmt.Errorf("starting service: %w", err)
	}
	fmt.Printf("✓ Started vibe-check daemon (PyPI :%d, npm :%d)\n", pypiPort, npmPort)

	// 8. Detect package managers and configure them
	managers := setup.DetectPackageManagers(pypiPort, npmPort)
	backupDir := filepath.Join(dataDir, "backups")
	configured, err := setup.BackupAndConfigure(managers, backupDir)
	if err != nil {
		return fmt.Errorf("configuring package managers: %w", err)
	}
	for _, name := range configured {
		fmt.Printf("✓ Configured %s\n", name)
	}

	fmt.Printf("\nvibe-check is running. Install packages as usual — new packages (< %s) will be blocked.\n", config.FormatMinAge(minAge))
	return nil
}
