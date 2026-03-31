package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type pypiVersionResponse struct {
	URLs []struct {
		UploadTimeISO string `json:"upload_time_iso_8601"`
	} `json:"urls"`
}

// PyPI implements Client for the Python Package Index.
type PyPI struct {
	client *http.Client
}

// NewPyPI creates a new PyPI registry client with a 15-second timeout.
func NewPyPI() *PyPI {
	return &PyPI{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FetchPublishDate retrieves the publish date for a specific package version from PyPI.
func (p *PyPI) FetchPublishDate(ctx context.Context, pkg, version string) (time.Time, error) {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/%s/json", pkg, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("fetching from PyPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("PyPI returned status %d for %s/%s", resp.StatusCode, pkg, version)
	}

	var result pypiVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return time.Time{}, fmt.Errorf("decoding PyPI response: %w", err)
	}

	if len(result.URLs) == 0 {
		return time.Time{}, fmt.Errorf("no URLs in PyPI response for %s/%s", pkg, version)
	}

	t, err := time.Parse(time.RFC3339Nano, result.URLs[0].UploadTimeISO)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing upload time %q: %w", result.URLs[0].UploadTimeISO, err)
	}

	return t, nil
}
