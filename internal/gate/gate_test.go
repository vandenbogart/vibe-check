package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckAge_OldPackage(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	published := time.Now().Add(-30 * 24 * time.Hour)
	result := g.CheckAge("npm", "react", "19.0.0", published)
	if result.Blocked {
		t.Errorf("expected old package (30 days) to not be blocked")
	}
	if result.Registry != "npm" {
		t.Errorf("expected registry npm, got %s", result.Registry)
	}
	if result.Package != "react" {
		t.Errorf("expected package react, got %s", result.Package)
	}
	if result.Version != "19.0.0" {
		t.Errorf("expected version 19.0.0, got %s", result.Version)
	}
}

func TestCheckAge_NewPackage(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	published := time.Now().Add(-2 * 24 * time.Hour)
	result := g.CheckAge("npm", "react", "19.0.0", published)
	if !result.Blocked {
		t.Errorf("expected new package (2 days) to be blocked")
	}
	if result.Age >= 7*24*time.Hour {
		t.Errorf("expected age < 7 days, got %v", result.Age)
	}
	msg := result.Message()
	if !strings.Contains(msg, "BLOCKED") {
		t.Errorf("expected message to contain BLOCKED, got %s", msg)
	}
	if !strings.Contains(msg, "vibe-check allow") {
		t.Errorf("expected message to contain bypass instructions, got %s", msg)
	}
}

func TestCheckAge_ExactBoundary(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	published := time.Now().Add(-7 * 24 * time.Hour)
	result := g.CheckAge("npm", "react", "19.0.0", published)
	if result.Blocked {
		t.Errorf("expected package at exact boundary (7 days) to not be blocked")
	}
}

func TestAllowlist(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	g.Allow("pypi", "flask", "3.1.0")

	if !g.IsAllowed("pypi", "flask", "3.1.0") {
		t.Errorf("expected pypi/flask/3.1.0 to be allowed")
	}
	if g.IsAllowed("pypi", "flask", "3.2.0") {
		t.Errorf("expected pypi/flask/3.2.0 to not be allowed")
	}
	if g.IsAllowed("npm", "flask", "3.1.0") {
		t.Errorf("expected npm/flask/3.1.0 to not be allowed")
	}
}

func TestCache(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	published := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	g.CachePublishDate("npm", "react", "19.0.0", published)

	cached, ok := g.GetCachedPublishDate("npm", "react", "19.0.0")
	if !ok {
		t.Errorf("expected cache hit for npm/react/19.0.0")
	}
	if !cached.Equal(published) {
		t.Errorf("expected cached date %v, got %v", published, cached)
	}

	_, ok = g.GetCachedPublishDate("npm", "react", "20.0.0")
	if ok {
		t.Errorf("expected cache miss for npm/react/20.0.0")
	}
}

func TestAllowlistPersistence(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	g.Allow("npm", "react", "19.0.0")
	g.Allow("pypi", "flask", "3.1.0")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "allowlist.txt")

	if err := g.SaveAllowlist(path); err != nil {
		t.Fatalf("SaveAllowlist failed: %v", err)
	}

	g2 := New(7 * 24 * time.Hour)
	if err := g2.LoadAllowlist(path); err != nil {
		t.Fatalf("LoadAllowlist failed: %v", err)
	}

	if !g2.IsAllowed("npm", "react", "19.0.0") {
		t.Errorf("expected npm/react/19.0.0 to be allowed after load")
	}
	if !g2.IsAllowed("pypi", "flask", "3.1.0") {
		t.Errorf("expected pypi/flask/3.1.0 to be allowed after load")
	}
}

func TestAllowlistFileFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "allowlist.txt")

	content := `# This is a comment
npm react 19.0.0

# Another comment
pypi flask 3.1.0

`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	g := New(7 * 24 * time.Hour)
	if err := g.LoadAllowlist(path); err != nil {
		t.Fatalf("LoadAllowlist failed: %v", err)
	}

	if !g.IsAllowed("npm", "react", "19.0.0") {
		t.Errorf("expected npm/react/19.0.0 to be allowed")
	}
	if !g.IsAllowed("pypi", "flask", "3.1.0") {
		t.Errorf("expected pypi/flask/3.1.0 to be allowed")
	}
}

func TestAllowlistFileFormat_MissingFile(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	err := g.LoadAllowlist("/nonexistent/path/allowlist.txt")
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
}

func TestSetMinAge(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	published := time.Now().Add(-3 * 24 * time.Hour)

	result := g.CheckAge("npm", "react", "19.0.0", published)
	if !result.Blocked {
		t.Errorf("expected 3-day-old package to be blocked with 7-day threshold")
	}

	g.SetMinAge(2 * 24 * time.Hour)
	if g.MinAge() != 2*24*time.Hour {
		t.Errorf("expected MinAge to be 2 days, got %v", g.MinAge())
	}

	result = g.CheckAge("npm", "react", "19.0.0", published)
	if result.Blocked {
		t.Errorf("expected 3-day-old package to pass with 2-day threshold")
	}
}

func TestAllowlistEntries(t *testing.T) {
	g := New(7 * 24 * time.Hour)
	g.Allow("npm", "react", "19.0.0")
	g.Allow("pypi", "flask", "3.1.0")

	entries := g.AllowlistEntries()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	found := map[string]bool{}
	for _, e := range entries {
		key := e.Registry + " " + e.Package + " " + e.Version
		found[key] = true
	}
	if !found["npm react 19.0.0"] {
		t.Errorf("expected npm react 19.0.0 in entries")
	}
	if !found["pypi flask 3.1.0"] {
		t.Errorf("expected pypi flask 3.1.0 in entries")
	}
}
