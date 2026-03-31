package resolve

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Package represents a resolved dependency with its name and version.
type Package struct {
	Name    string
	Version string
}

const resolveTimeout = 120 * time.Second

// NPMDeps resolves the full dependency tree for an npm package by running a
// dry-run install against the upstream registry. It returns every package
// (including the root) that would be installed.
func NPMDeps(pkg, version string) ([]Package, error) {
	tmpDir, err := os.MkdirTemp("", "vibe-check-npm-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
	defer cancel()

	// Initialize a minimal package.json so npm install works.
	initCmd := exec.CommandContext(ctx, "npm", "init", "-y", "--silent")
	initCmd.Dir = tmpDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("npm init failed: %w\n%s", err, out)
	}

	// Dry-run install to discover the full dependency tree.
	spec := fmt.Sprintf("%s@%s", pkg, version)
	installCmd := exec.CommandContext(ctx, "npm", "install", "--dry-run",
		"--registry=https://registry.npmjs.org", spec)
	installCmd.Dir = tmpDir
	out, err := installCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("npm install --dry-run failed: %w\n%s", err, out)
	}

	return parseNPMOutput(string(out)), nil
}

// parseNPMOutput extracts packages from npm dry-run output.
// Lines look like: "add <name> <version>"
func parseNPMOutput(output string) []Package {
	var pkgs []Package
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "add" {
			pkgs = append(pkgs, Package{
				Name:    fields[1],
				Version: fields[2],
			})
		}
	}
	return pkgs
}

// pypiReport is the structure of pip's --report JSON output.
type pypiReport struct {
	Install []pypiInstallItem `json:"install"`
}

type pypiInstallItem struct {
	Metadata pypiMetadata `json:"metadata"`
}

type pypiMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// PyPIDeps resolves the full dependency tree for a PyPI package by running a
// dry-run install with pip's --report flag. It returns every package
// (including the root) that would be installed.
func PyPIDeps(pkg, version string) ([]Package, error) {
	tmpDir, err := os.MkdirTemp("", "vibe-check-pypi-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	reportFile := filepath.Join(tmpDir, "report.json")

	ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
	defer cancel()

	spec := fmt.Sprintf("%s==%s", pkg, version)
	cmd := exec.CommandContext(ctx, "pip", "install", "--dry-run",
		"--report", reportFile,
		"--index-url", "https://pypi.org/simple/",
		spec)
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pip install --dry-run failed: %w\n%s", err, out)
	}

	data, err := os.ReadFile(reportFile)
	if err != nil {
		return nil, fmt.Errorf("reading pip report: %w", err)
	}

	var report pypiReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parsing pip report: %w", err)
	}

	pkgs := make([]Package, 0, len(report.Install))
	for _, item := range report.Install {
		pkgs = append(pkgs, Package{
			Name:    normalizePyPIName(item.Metadata.Name),
			Version: item.Metadata.Version,
		})
	}
	return pkgs, nil
}

// normalizePyPIName lowercases and replaces underscores with hyphens,
// matching the canonical PyPI name format.
func normalizePyPIName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}
