package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vandenbogart/vibe-check/internal/config"
	"github.com/vandenbogart/vibe-check/internal/gate"
	"github.com/vandenbogart/vibe-check/internal/logging"
)

func unixClient(sockPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", sockPath)
			},
		},
	}
}

func startTestServer(t *testing.T) (sockPath string, g *gate.Gate, srv *Server) {
	t.Helper()

	tmpDir := t.TempDir()
	sockPath = filepath.Join(tmpDir, "admin.sock")

	g = gate.New(7 * 24 * time.Hour)
	cfg := config.Default()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	// Set a known log level for tests.
	logging.SetLevel(slog.LevelInfo)

	srv = NewServer(g, &cfg, tmpDir, logger)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}

	go func() {
		_ = srv.Serve(ln)
	}()

	t.Cleanup(func() {
		_ = srv.Close()
	})

	return sockPath, g, srv
}

func TestAdminAllow(t *testing.T) {
	sockPath, g, _ := startTestServer(t)
	client := unixClient(sockPath)

	body, _ := json.Marshal(AllowRequest{
		Registry: "npm",
		Package:  "lodash",
		Version:  "4.17.21",
	})

	resp, err := client.Post("http://localhost/admin/allow", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /admin/allow: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", result["status"])
	}

	if !g.IsAllowed("npm", "lodash", "4.17.21") {
		t.Fatal("expected lodash to be allowed after POST")
	}
}

func TestAdminStatus(t *testing.T) {
	sockPath, g, _ := startTestServer(t)
	client := unixClient(sockPath)

	// Add an entry to the gate.
	g.Allow("pypi", "requests", "2.31.0")

	resp, err := client.Get("http://localhost/admin/status")
	if err != nil {
		t.Fatalf("GET /admin/status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if status.AllowlistCount != 1 {
		t.Fatalf("expected allowlist_count=1, got %d", status.AllowlistCount)
	}
	if status.MinAge != "7d" {
		t.Fatalf("expected min_age=7d, got %q", status.MinAge)
	}
}

func TestAdminConfig(t *testing.T) {
	sockPath, g, _ := startTestServer(t)
	client := unixClient(sockPath)

	body, _ := json.Marshal(ConfigRequest{
		MinAge: "14d",
	})

	req, _ := http.NewRequest(http.MethodPut, "http://localhost/admin/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT /admin/config: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	expected := 14 * 24 * time.Hour
	if g.MinAge() != expected {
		t.Fatalf("expected gate min_age=%v, got %v", expected, g.MinAge())
	}
}

func TestAdminLogLevel(t *testing.T) {
	sockPath, _, _ := startTestServer(t)
	client := unixClient(sockPath)

	body, _ := json.Marshal(LogLevelRequest{
		Level: "trace",
	})

	req, _ := http.NewRequest(http.MethodPut, "http://localhost/admin/log-level", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT /admin/log-level: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if logging.GetLevel() != logging.LevelTrace {
		t.Fatalf("expected log level TRACE, got %v", logging.GetLevel())
	}
}
