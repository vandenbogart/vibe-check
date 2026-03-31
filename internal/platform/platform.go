package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ServiceOpts holds configuration for installing the vibe-check service.
type ServiceOpts struct {
	BinaryPath string
	DataDir    string
	PyPIPort   int
	NPMPort    int
	MinAge     string
	LogLevel   string
}

// ServiceManager abstracts platform-specific service management.
type ServiceManager interface {
	Install(opts ServiceOpts) error
	Uninstall() error
	Start() error
	Stop() error
}

// Detect returns the appropriate ServiceManager for the current OS.
// Returns Launchd on darwin, Systemd on linux, nil otherwise.
func Detect() ServiceManager {
	switch runtime.GOOS {
	case "darwin":
		return &Launchd{}
	case "linux":
		return &Systemd{}
	default:
		return nil
	}
}

// FindBinary returns the absolute path of the current executable, resolving symlinks.
func FindBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving symlinks: %w", err)
	}
	return resolved, nil
}
