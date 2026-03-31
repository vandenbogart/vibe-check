package gate

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// CheckResult holds the outcome of an age check on a package version.
type CheckResult struct {
	Blocked  bool
	Age      time.Duration
	MinAge   time.Duration
	Registry string
	Package  string
	Version  string
}

// Message returns a human-readable block message with bypass instructions.
func (r CheckResult) Message() string {
	if !r.Blocked {
		return ""
	}
	ageDays := int(r.Age.Hours() / 24)
	minAgeDays := int(r.MinAge.Hours() / 24)
	return fmt.Sprintf(
		"BLOCKED: %s@%s published %d days ago (minimum age: %d days)\nTo override: vibe-check allow %s %s %s\nThen retry your install.",
		r.Package, r.Version, ageDays, minAgeDays,
		r.Registry, r.Package, r.Version,
	)
}

// AllowlistEntry represents a single entry in the allowlist.
type AllowlistEntry struct {
	Registry string
	Package  string
	Version  string
}

// Gate provides age-based gating for package downloads, with an allowlist and
// publish-date cache.
type Gate struct {
	mu        sync.RWMutex
	minAge    time.Duration
	allowlist map[string]struct{}
	cache     map[string]time.Time
}

// New creates a Gate with the given minimum age threshold.
func New(minAge time.Duration) *Gate {
	return &Gate{
		minAge:    minAge,
		allowlist: make(map[string]struct{}),
		cache:     make(map[string]time.Time),
	}
}

func allowlistKey(registry, pkg, version string) string {
	return registry + " " + pkg + " " + version
}

func cacheKey(registry, pkg, version string) string {
	return registry + "/" + pkg + "/" + version
}

// CheckAge checks whether a package version is old enough to pass the gate.
// At exact boundary (age == minAge) the package is NOT blocked.
func (g *Gate) CheckAge(registry, pkg, version string, published time.Time) CheckResult {
	g.mu.RLock()
	minAge := g.minAge
	g.mu.RUnlock()

	age := time.Since(published)
	blocked := age < minAge

	return CheckResult{
		Blocked:  blocked,
		Age:      age,
		MinAge:   minAge,
		Registry: registry,
		Package:  pkg,
		Version:  version,
	}
}

// IsAllowed returns true if the given package version is in the allowlist.
func (g *Gate) IsAllowed(registry, pkg, version string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.allowlist[allowlistKey(registry, pkg, version)]
	return ok
}

// Allow adds a package version to the allowlist.
func (g *Gate) Allow(registry, pkg, version string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.allowlist[allowlistKey(registry, pkg, version)] = struct{}{}
}

// AllowlistEntries returns all entries in the allowlist.
func (g *Gate) AllowlistEntries() []AllowlistEntry {
	g.mu.RLock()
	defer g.mu.RUnlock()
	entries := make([]AllowlistEntry, 0, len(g.allowlist))
	for key := range g.allowlist {
		parts := strings.SplitN(key, " ", 3)
		if len(parts) == 3 {
			entries = append(entries, AllowlistEntry{
				Registry: parts[0],
				Package:  parts[1],
				Version:  parts[2],
			})
		}
	}
	return entries
}

// SetMinAge updates the minimum age threshold.
func (g *Gate) SetMinAge(d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.minAge = d
}

// MinAge returns the current minimum age threshold.
func (g *Gate) MinAge() time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.minAge
}

// CachePublishDate stores a publish date in the cache.
func (g *Gate) CachePublishDate(registry, pkg, version string, published time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cache[cacheKey(registry, pkg, version)] = published
}

// GetCachedPublishDate retrieves a cached publish date. Returns the time and
// true if found, or the zero time and false otherwise.
func (g *Gate) GetCachedPublishDate(registry, pkg, version string) (time.Time, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	t, ok := g.cache[cacheKey(registry, pkg, version)]
	return t, ok
}

// SaveAllowlist writes the allowlist to a file, one "registry package version"
// entry per line.
func (g *Gate) SaveAllowlist(path string) error {
	g.mu.RLock()
	entries := make([]string, 0, len(g.allowlist))
	for key := range g.allowlist {
		entries = append(entries, key)
	}
	g.mu.RUnlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, entry := range entries {
		if _, err := fmt.Fprintln(w, entry); err != nil {
			return err
		}
	}
	return w.Flush()
}

// LoadAllowlist reads an allowlist from a file. Lines starting with "#" are
// treated as comments and blank lines are skipped. Returns nil if the file
// does not exist.
func (g *Gate) LoadAllowlist(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	g.mu.Lock()
	defer g.mu.Unlock()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) == 3 {
			g.allowlist[allowlistKey(parts[0], parts[1], parts[2])] = struct{}{}
		}
	}
	return scanner.Err()
}
