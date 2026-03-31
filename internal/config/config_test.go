package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.PyPIPort != 3141 {
		t.Errorf("PyPIPort = %d, want 3141", cfg.PyPIPort)
	}
	if cfg.NPMPort != 3142 {
		t.Errorf("NPMPort = %d, want 3142", cfg.NPMPort)
	}
	if cfg.MinAge != 7*24*time.Hour {
		t.Errorf("MinAge = %v, want %v", cfg.MinAge, 7*24*time.Hour)
	}
	if cfg.MinAgeS != "7d" {
		t.Errorf("MinAgeS = %q, want %q", cfg.MinAgeS, "7d")
	}
}

func TestParseMinAge(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"7d", 7 * 24 * time.Hour, false},
		{"14d", 14 * 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"0d", 0, false},
		{"", 0, true},
		{"abc", 0, true},
		{"7", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseMinAge(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMinAge(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseMinAge(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatMinAge(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{7 * 24 * time.Hour, "7d"},
		{14 * 24 * time.Hour, "14d"},
		{24 * time.Hour, "1d"},
		{0, "0d"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatMinAge(tt.input)
			if got != tt.want {
				t.Errorf("FormatMinAge(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.json")

	cfg := Config{
		PyPIPort: 4000,
		NPMPort:  4001,
		MinAge:   14 * 24 * time.Hour,
		MinAgeS:  "14d",
	}

	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.PyPIPort != cfg.PyPIPort {
		t.Errorf("PyPIPort = %d, want %d", loaded.PyPIPort, cfg.PyPIPort)
	}
	if loaded.NPMPort != cfg.NPMPort {
		t.Errorf("NPMPort = %d, want %d", loaded.NPMPort, cfg.NPMPort)
	}
	if loaded.MinAge != cfg.MinAge {
		t.Errorf("MinAge = %v, want %v", loaded.MinAge, cfg.MinAge)
	}
	if loaded.MinAgeS != cfg.MinAgeS {
		t.Errorf("MinAgeS = %q, want %q", loaded.MinAgeS, cfg.MinAgeS)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "config.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() for missing file should not error, got %v", err)
	}

	def := Default()
	if cfg.PyPIPort != def.PyPIPort {
		t.Errorf("PyPIPort = %d, want default %d", cfg.PyPIPort, def.PyPIPort)
	}
	if cfg.NPMPort != def.NPMPort {
		t.Errorf("NPMPort = %d, want default %d", cfg.NPMPort, def.NPMPort)
	}
	if cfg.MinAge != def.MinAge {
		t.Errorf("MinAge = %v, want default %v", cfg.MinAge, def.MinAge)
	}
}

func TestDataDir(t *testing.T) {
	dir := DefaultDataDir()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	want := filepath.Join(home, ".vibe-check")
	if dir != want {
		t.Errorf("DefaultDataDir() = %q, want %q", dir, want)
	}
}
