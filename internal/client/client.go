package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/vandenbogart/vibe-check/internal/config"
)

// Client talks to the vibe-check daemon admin API over a Unix socket.
type Client struct {
	http     *http.Client
	sockPath string
}

// New creates a Client that connects to the daemon socket in dataDir.
func New(dataDir string) *Client {
	sockPath := filepath.Join(dataDir, "vibe-check.sock")
	return &Client{
		sockPath: sockPath,
		http: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", sockPath)
				},
			},
		},
	}
}

// Allow tells the daemon to add a package version to the allowlist.
func (c *Client) Allow(registry, pkg, version string) error {
	body, err := json.Marshal(map[string]string{
		"registry": registry,
		"package":  pkg,
		"version":  version,
	})
	if err != nil {
		return err
	}

	resp, err := c.http.Post("http://localhost/admin/allow", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot reach vibe-check daemon (is it running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned %d: %s", resp.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}

// SetMinAge tells the daemon to update the minimum package age.
// It validates the duration string locally before sending.
func (c *Client) SetMinAge(minAge string) error {
	if _, err := config.ParseMinAge(minAge); err != nil {
		return err
	}

	body, err := json.Marshal(map[string]string{
		"min_age": minAge,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, "http://localhost/admin/config", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach vibe-check daemon (is it running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned %d: %s", resp.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}

// SetLogLevel tells the daemon to change its log level.
func (c *Client) SetLogLevel(level string) error {
	body, err := json.Marshal(map[string]string{
		"level": level,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, "http://localhost/admin/log-level", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach vibe-check daemon (is it running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned %d: %s", resp.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}
