package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type npmPackageDoc struct {
	Time map[string]string `json:"time"`
}

// NPM implements Client for the npm registry.
type NPM struct {
	client *http.Client
}

// NewNPM creates a new npm registry client with a 15-second timeout.
func NewNPM() *NPM {
	return &NPM{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FetchPublishDate retrieves the publish date for a specific package version from npm.
func (n *NPM) FetchPublishDate(ctx context.Context, pkg, version string) (time.Time, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkg)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("creating request: %w", err)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("fetching from npm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("npm returned status %d for %s", resp.StatusCode, pkg)
	}

	var doc npmPackageDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return time.Time{}, fmt.Errorf("decoding npm response: %w", err)
	}

	raw, ok := doc.Time[version]
	if !ok {
		return time.Time{}, fmt.Errorf("version %s not found in npm time map for %s", version, pkg)
	}

	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing publish time %q: %w", raw, err)
	}

	return t, nil
}
