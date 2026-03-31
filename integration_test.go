//go:build integration

package main_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vandenbogart/vibe-check/internal/gate"
	"github.com/vandenbogart/vibe-check/internal/proxy"
	"github.com/vandenbogart/vibe-check/internal/registry"
)

func startPyPIProxy(t *testing.T, g *gate.Gate) int {
	t.Helper()

	reg := registry.NewPyPI()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.Level(-8)}))
	p := proxy.NewPyPIProxy(g, reg, logger)

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := &http.Server{Handler: p.Handler()}
	go srv.Serve(ln)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	})

	return ln.Addr().(*net.TCPAddr).Port
}

func startNPMProxy(t *testing.T, g *gate.Gate) int {
	t.Helper()

	reg := registry.NewNPM()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.Level(-8)}))
	p := proxy.NewNPMProxy(g, reg, logger)

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := &http.Server{Handler: p.Handler()}
	go srv.Serve(ln)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	})

	return ln.Addr().(*net.TCPAddr).Port
}

func TestIntegration_PyPI_SimpleIndex(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startPyPIProxy(t, g)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/simple/flask/", port))
	if err != nil {
		t.Fatalf("GET /simple/flask/ failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if strings.Contains(bodyStr, "files.pythonhosted.org") {
		t.Error("response should NOT contain files.pythonhosted.org (URLs should be rewritten)")
	}

	if !strings.Contains(bodyStr, "/packages/") {
		t.Error("response should contain /packages/ (rewritten URLs)")
	}

	if !strings.Contains(strings.ToLower(bodyStr), "flask") {
		t.Error("response should contain 'flask'")
	}
}

func TestIntegration_PyPI_OldPackageAllowed(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startPyPIProxy(t, g)

	// Get the simple index for flask
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/simple/flask/", port))
	if err != nil {
		t.Fatalf("GET /simple/flask/ failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	// Find a .tar.gz link for flask 3.1.0
	// Links look like: href="/packages/.../flask-3.1.0.tar.gz#sha256=..."
	var downloadPath string
	for _, line := range strings.Split(bodyStr, "\n") {
		if strings.Contains(line, "flask-3.1.0.tar.gz") || strings.Contains(line, "Flask-3.1.0.tar.gz") {
			// Extract href value
			idx := strings.Index(line, "href=\"")
			if idx < 0 {
				continue
			}
			start := idx + len("href=\"")
			end := strings.Index(line[start:], "\"")
			if end < 0 {
				continue
			}
			href := line[start : start+end]
			// Strip fragment
			if fragIdx := strings.Index(href, "#"); fragIdx >= 0 {
				href = href[:fragIdx]
			}
			if strings.HasSuffix(href, ".tar.gz") {
				downloadPath = href
				break
			}
		}
	}

	if downloadPath == "" {
		t.Fatal("could not find flask 3.1.0 .tar.gz link in simple index")
	}

	t.Logf("found download path: %s", downloadPath)

	// GET the download URL through the proxy
	resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, downloadPath))
	if err != nil {
		t.Fatalf("GET %s failed: %v", downloadPath, err)
	}
	defer resp2.Body.Close()
	io.Copy(io.Discard, resp2.Body)

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for old package, got %d", resp2.StatusCode)
	}
}

func TestIntegration_NPM_MetadataPassthrough(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startNPMProxy(t, g)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/express", port))
	if err != nil {
		t.Fatalf("GET /express failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestIntegration_NPM_OldTarballAllowed(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startNPMProxy(t, g)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/express/-/express-4.21.0.tgz", port))
	if err != nil {
		t.Fatalf("GET express tarball failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestIntegration_NPM_ScopedPackage(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startNPMProxy(t, g)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/@types/node/-/node-22.0.0.tgz", port))
	if err != nil {
		t.Fatalf("GET @types/node tarball failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestIntegration_Allowlist_Bypass(t *testing.T) {
	// 100-year min age — everything should be blocked
	g := gate.New(100 * 365 * 24 * time.Hour)
	port := startNPMProxy(t, g)

	// First request: should be blocked
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/express/-/express-4.21.0.tgz", port))
	if err != nil {
		t.Fatalf("GET express tarball failed: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for blocked package, got %d", resp.StatusCode)
	}

	if !strings.Contains(bodyStr, "BLOCKED") {
		t.Error("blocked response should contain 'BLOCKED'")
	}

	if !strings.Contains(bodyStr, "vibe-check allow") {
		t.Error("blocked response should contain 'vibe-check allow'")
	}

	// Add to allowlist
	g.Allow("npm", "express", "4.21.0")

	// Second request: should now pass
	resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/express/-/express-4.21.0.tgz", port))
	if err != nil {
		t.Fatalf("GET express tarball (after allow) failed: %v", err)
	}
	defer resp2.Body.Close()
	io.Copy(io.Discard, resp2.Body)

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after allowlist, got %d", resp2.StatusCode)
	}
}
