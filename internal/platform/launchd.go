package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	launchdLabel = "com.vibe-check"
	plistFile    = "com.vibe-check.plist"
)

// launchdPlistPath returns ~/Library/LaunchAgents/com.vibe-check.plist.
func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", plistFile)
}

// GenerateLaunchdPlist returns the plist XML for the vibe-check launchd service.
func GenerateLaunchdPlist(opts ServiceOpts) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>serve</string>
        <string>--pypi-port</string>
        <string>%d</string>
        <string>--npm-port</string>
        <string>%d</string>
        <string>--min-age</string>
        <string>%s</string>
        <string>--data-dir</string>
        <string>%s</string>
        <string>--log-level</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, launchdLabel, opts.BinaryPath, opts.PyPIPort, opts.NPMPort,
		opts.MinAge, opts.DataDir, opts.LogLevel,
		filepath.Join(opts.DataDir, "vibe-check.log"),
		filepath.Join(opts.DataDir, "vibe-check.log"))
}

// Launchd implements ServiceManager for macOS.
type Launchd struct{}

// Install writes the launchd plist file.
func (l *Launchd) Install(opts ServiceOpts) error {
	plistPath := launchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("creating LaunchAgents dir: %w", err)
	}
	content := GenerateLaunchdPlist(opts)
	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}
	return nil
}

// Uninstall removes the launchd plist file. No-op if missing.
func (l *Launchd) Uninstall() error {
	plistPath := launchdPlistPath()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}
	return nil
}

// Start loads the launchd service.
func (l *Launchd) Start() error {
	return run("launchctl", "load", launchdPlistPath())
}

// Stop unloads the launchd service.
func (l *Launchd) Stop() error {
	return run("launchctl", "unload", launchdPlistPath())
}

// run executes a command with stdout/stderr connected to the process.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
