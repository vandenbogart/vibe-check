package setup

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PackageManager describes a detected package manager and its proxy config.
type PackageManager struct {
	Name       string
	ConfigPath string
	Content    string
}

// DetectPackageManagers checks which package managers are installed and returns
// the config path and content needed to redirect each one to vibe-check.
func DetectPackageManagers(pypiPort, npmPort int) []PackageManager {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var managers []PackageManager

	hasPip := false
	if _, err := exec.LookPath("pip"); err == nil {
		hasPip = true
		managers = append(managers, PackageManager{
			Name:       "pip",
			ConfigPath: filepath.Join(home, ".config", "pip", "pip.conf"),
			Content:    fmt.Sprintf("[global]\nindex-url = http://localhost:%d/simple/\n", pypiPort),
		})
	}

	if _, err := exec.LookPath("pip3"); err == nil && !hasPip {
		managers = append(managers, PackageManager{
			Name:       "pip",
			ConfigPath: filepath.Join(home, ".config", "pip", "pip.conf"),
			Content:    fmt.Sprintf("[global]\nindex-url = http://localhost:%d/simple/\n", pypiPort),
		})
	}

	if _, err := exec.LookPath("uv"); err == nil {
		managers = append(managers, PackageManager{
			Name:       "uv",
			ConfigPath: filepath.Join(home, ".config", "uv", "uv.toml"),
			Content:    fmt.Sprintf("index-url = \"http://localhost:%d/simple/\"\n", pypiPort),
		})
	}

	if _, err := exec.LookPath("npm"); err == nil {
		managers = append(managers, PackageManager{
			Name:       "npm",
			ConfigPath: filepath.Join(home, ".npmrc"),
			Content:    fmt.Sprintf("registry=http://localhost:%d/\n", npmPort),
		})
	}

	return managers
}

// BackupAndConfigure backs up existing config files and writes proxy configs.
// Returns the list of configured manager names.
func BackupAndConfigure(managers []PackageManager, backupDir string) ([]string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating backup dir: %w", err)
	}

	var configured []string
	for _, m := range managers {
		// Backup existing config if present
		if _, err := os.Stat(m.ConfigPath); err == nil {
			backupPath := filepath.Join(backupDir, m.Name+".bak")
			data, err := os.ReadFile(m.ConfigPath)
			if err != nil {
				return configured, fmt.Errorf("reading %s config for backup: %w", m.Name, err)
			}
			if err := os.WriteFile(backupPath, data, 0o644); err != nil {
				return configured, fmt.Errorf("writing %s backup: %w", m.Name, err)
			}
		}

		// Create parent directories for config
		if err := os.MkdirAll(filepath.Dir(m.ConfigPath), 0o755); err != nil {
			return configured, fmt.Errorf("creating config dir for %s: %w", m.Name, err)
		}

		// Write config
		if m.Name == "npm" {
			if err := updateNPMRC(m.ConfigPath, m.Content); err != nil {
				return configured, fmt.Errorf("updating npmrc: %w", err)
			}
		} else {
			if err := os.WriteFile(m.ConfigPath, []byte(m.Content), 0o644); err != nil {
				return configured, fmt.Errorf("writing %s config: %w", m.Name, err)
			}
		}

		configured = append(configured, m.Name)
	}

	return configured, nil
}

// updateNPMRC reads the existing .npmrc, removes any existing registry= lines,
// prepends the new registry line, and writes it back.
func updateNPMRC(path, registryLine string) error {
	registryLine = strings.TrimRight(registryLine, "\n")

	var existingLines []string
	if data, err := os.ReadFile(path); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "registry=") {
				existingLines = append(existingLines, line)
			}
		}
	}

	var lines []string
	lines = append(lines, registryLine)
	lines = append(lines, existingLines...)

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// RestoreConfigs restores backed-up config files and cleans up.
// Returns the list of restored manager names.
func RestoreConfigs(managers []PackageManager, backupDir string) []string {
	var restored []string
	for _, m := range managers {
		backupPath := filepath.Join(backupDir, m.Name+".bak")
		if _, err := os.Stat(backupPath); err == nil {
			// Backup exists — restore it
			data, err := os.ReadFile(backupPath)
			if err != nil {
				continue
			}
			if err := os.WriteFile(m.ConfigPath, data, 0o644); err != nil {
				continue
			}
			os.Remove(backupPath)
			restored = append(restored, m.Name)
		} else {
			// No backup — remove the config we wrote
			if _, err := os.Stat(m.ConfigPath); err == nil {
				os.Remove(m.ConfigPath)
				restored = append(restored, m.Name)
			}
		}
	}
	return restored
}
