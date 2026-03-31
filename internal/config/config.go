package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds the runtime configuration for vibe-check.
type Config struct {
	PyPIPort int           `json:"pypi_port"`
	NPMPort  int           `json:"npm_port"`
	MinAge   time.Duration `json:"-"`
	MinAgeS  string        `json:"min_age"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{
		PyPIPort: 3141,
		NPMPort:  3142,
		MinAge:   7 * 24 * time.Hour,
		MinAgeS:  "7d",
	}
}

// DefaultDataDir returns the default data directory path (~/.vibe-check/).
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".vibe-check")
}

// ParseMinAge parses a string like "7d" into a time.Duration.
// The string must end with "d" and the prefix must be a valid integer.
func ParseMinAge(s string) (time.Duration, error) {
	if s == "" {
		return 0, errors.New("empty min_age string")
	}
	if !strings.HasSuffix(s, "d") {
		return 0, fmt.Errorf("min_age %q must end with \"d\"", s)
	}
	numStr := strings.TrimSuffix(s, "d")
	days, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid min_age %q: %w", s, err)
	}
	return time.Duration(days) * 24 * time.Hour, nil
}

// FormatMinAge converts a duration back to "Nd" format.
func FormatMinAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// Save marshals the config to JSON and writes it to the given path,
// creating parent directories as needed.
func Save(cfg Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// Load reads config from the given path. If the file does not exist,
// it returns the default config without an error. After loading, it
// parses MinAgeS back into MinAge.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshaling config: %w", err)
	}

	if cfg.MinAgeS != "" {
		parsed, err := ParseMinAge(cfg.MinAgeS)
		if err != nil {
			return Config{}, fmt.Errorf("parsing min_age: %w", err)
		}
		cfg.MinAge = parsed
	}

	return cfg, nil
}
