package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	systemdServiceName = "vibe-check.service"
)

// systemdUnitPath returns ~/.config/systemd/user/vibe-check.service.
func systemdUnitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", systemdServiceName)
}

// GenerateSystemdUnit returns the systemd unit file content for the vibe-check service.
func GenerateSystemdUnit(opts ServiceOpts) string {
	return fmt.Sprintf(`[Unit]
Description=vibe-check package age proxy
After=network.target

[Service]
Type=simple
ExecStart=%s serve --pypi-port %d --npm-port %d --min-age %s --data-dir %s --log-level %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, opts.BinaryPath, opts.PyPIPort, opts.NPMPort, opts.MinAge, opts.DataDir, opts.LogLevel)
}

// Systemd implements ServiceManager for Linux.
type Systemd struct{}

// Install writes the systemd unit file and runs daemon-reload.
func (s *Systemd) Install(opts ServiceOpts) error {
	unitPath := systemdUnitPath()
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return fmt.Errorf("creating systemd user dir: %w", err)
	}
	content := GenerateSystemdUnit(opts)
	if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}
	return run("systemctl", "--user", "daemon-reload")
}

// Uninstall removes the systemd unit file and runs daemon-reload. No-op if missing.
func (s *Systemd) Uninstall() error {
	unitPath := systemdUnitPath()
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit file: %w", err)
	}
	return run("systemctl", "--user", "daemon-reload")
}

// Start enables and starts the systemd service.
func (s *Systemd) Start() error {
	return run("systemctl", "--user", "enable", "--now", "vibe-check")
}

// Stop stops and disables the systemd service.
func (s *Systemd) Stop() error {
	if err := run("systemctl", "--user", "stop", "vibe-check"); err != nil {
		return err
	}
	return run("systemctl", "--user", "disable", "vibe-check")
}
