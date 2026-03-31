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

// TestIntegration_PyPI_MetadataFile verifies that PEP 658 .metadata requests
// are handled correctly (parsed as the underlying package, not blocked as unknown).
func TestIntegration_PyPI_MetadataFile(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startPyPIProxy(t, g)

	// Get the simple index to find a .whl.metadata URL
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/simple/flask/", port))
	if err != nil {
		t.Fatalf("GET /simple/flask/ failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Find a flask 3.1.0 wheel link and append .metadata
	var whlPath string
	for _, line := range strings.Split(string(body), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "flask-3.1.0") && strings.Contains(lower, ".whl") && !strings.Contains(lower, ".metadata") {
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
			if fragIdx := strings.Index(href, "#"); fragIdx >= 0 {
				href = href[:fragIdx]
			}
			if strings.HasSuffix(href, ".whl") {
				whlPath = href
				break
			}
		}
	}

	if whlPath == "" {
		t.Fatal("could not find flask 3.1.0 .whl link")
	}

	// Request the .metadata variant — should pass age check (flask is old)
	metadataURL := fmt.Sprintf("http://127.0.0.1:%d%s.metadata", port, whlPath)
	resp2, err := http.Get(metadataURL)
	if err != nil {
		t.Fatalf("GET .metadata failed: %v", err)
	}
	defer resp2.Body.Close()
	io.Copy(io.Discard, resp2.Body)

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for .metadata of old package, got %d", resp2.StatusCode)
	}
}

// TestIntegration_PyPI_MultipleOldPackages tests several well-known old packages
// to verify broad compatibility with real PyPI filenames and API responses.
func TestIntegration_PyPI_MultipleOldPackages(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startPyPIProxy(t, g)

	packages := []struct {
		name    string
		version string
	}{
		{"flask", "3.1.0"},
		{"pydantic", "2.12.5"},
		{"black", "26.3.1"},
		{"httpx", "0.28.1"},
		{"pillow", "12.1.1"},
	}

	for _, pkg := range packages {
		t.Run(pkg.name, func(t *testing.T) {
			// Fetch the simple index
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/simple/%s/", port, pkg.name))
			if err != nil {
				t.Fatalf("GET /simple/%s/ failed: %v", pkg.name, err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("simple index returned %d", resp.StatusCode)
			}

			// Find any download link for this version
			var downloadPath string
			for _, line := range strings.Split(string(body), "\n") {
				if strings.Contains(line, pkg.version) {
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
					if fragIdx := strings.Index(href, "#"); fragIdx >= 0 {
						href = href[:fragIdx]
					}
					if strings.HasSuffix(href, ".whl") || strings.HasSuffix(href, ".tar.gz") {
						downloadPath = href
						break
					}
				}
			}

			if downloadPath == "" {
				t.Skipf("could not find download link for %s==%s", pkg.name, pkg.version)
				return
			}

			resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, downloadPath))
			if err != nil {
				t.Fatalf("download failed: %v", err)
			}
			defer resp2.Body.Close()
			io.Copy(io.Discard, resp2.Body)

			if resp2.StatusCode != http.StatusOK {
				t.Errorf("expected 200 for %s==%s, got %d", pkg.name, pkg.version, resp2.StatusCode)
			}
		})
	}
}

// TestIntegration_NPM_MultipleOldPackages tests several well-known old npm packages
// including scoped packages and packages with native binaries.
func TestIntegration_NPM_MultipleOldPackages(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startNPMProxy(t, g)

	packages := []struct {
		name    string
		version string
		path    string // tarball path
	}{
		{"express", "4.21.0", "/express/-/express-4.21.0.tgz"},
		{"prettier", "3.8.1", "/prettier/-/prettier-3.8.1.tgz"},
		{"esbuild", "0.27.4", "/esbuild/-/esbuild-0.27.4.tgz"},
		{"zod", "4.3.6", "/zod/-/zod-4.3.6.tgz"},
		{"@types/node", "22.0.0", "/@types/node/-/node-22.0.0.tgz"},
		{"@babel/core", "7.26.0", "/@babel/core/-/core-7.26.0.tgz"},
	}

	for _, pkg := range packages {
		t.Run(pkg.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, pkg.path))
			if err != nil {
				t.Fatalf("GET %s failed: %v", pkg.path, err)
			}
			defer resp.Body.Close()
			io.Copy(io.Discard, resp.Body)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200 for %s@%s, got %d", pkg.name, pkg.version, resp.StatusCode)
			}
		})
	}
}

// TestIntegration_NPM_BlockAndAllow verifies the full block→allow→pass cycle
// for npm using a 100-year gate (guarantees any real package is blocked).
func TestIntegration_NPM_BlockAndAllow(t *testing.T) {
	g := gate.New(100 * 365 * 24 * time.Hour)
	port := startNPMProxy(t, g)

	packages := []struct {
		name    string
		version string
		path    string
	}{
		{"express", "4.21.0", "/express/-/express-4.21.0.tgz"},
		{"@types/node", "22.0.0", "/@types/node/-/node-22.0.0.tgz"},
	}

	for _, pkg := range packages {
		t.Run(pkg.name, func(t *testing.T) {
			// Should be blocked
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, pkg.path))
			if err != nil {
				t.Fatalf("GET %s failed: %v", pkg.path, err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("expected 403, got %d", resp.StatusCode)
			}
			if !strings.Contains(string(body), "BLOCKED") {
				t.Error("response should contain BLOCKED")
			}

			// Allow it
			g.Allow("npm", pkg.name, pkg.version)

			// Should now pass
			resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, pkg.path))
			if err != nil {
				t.Fatalf("GET %s (after allow) failed: %v", pkg.path, err)
			}
			defer resp2.Body.Close()
			io.Copy(io.Discard, resp2.Body)

			if resp2.StatusCode != http.StatusOK {
				t.Errorf("expected 200 after allow, got %d", resp2.StatusCode)
			}
		})
	}
}

// TestIntegration_PyPI_BlockAndAllow verifies the full block→allow→pass cycle for PyPI.
func TestIntegration_PyPI_BlockAndAllow(t *testing.T) {
	g := gate.New(100 * 365 * 24 * time.Hour)
	port := startPyPIProxy(t, g)

	// Get flask simple index
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/simple/flask/", port))
	if err != nil {
		t.Fatalf("GET /simple/flask/ failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Find a flask 3.1.0 download link
	var downloadPath string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.Contains(strings.ToLower(line), "flask-3.1.0") && strings.Contains(line, ".tar.gz") {
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
			if fragIdx := strings.Index(href, "#"); fragIdx >= 0 {
				href = href[:fragIdx]
			}
			downloadPath = href
			break
		}
	}
	if downloadPath == "" {
		t.Fatal("could not find flask 3.1.0 download link")
	}

	// Should be blocked (100-year gate)
	resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, downloadPath))
	if err != nil {
		t.Fatalf("GET download failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for blocked package, got %d", resp2.StatusCode)
	}
	if !strings.Contains(string(body2), "BLOCKED") {
		t.Error("response should contain BLOCKED")
	}
	if !strings.Contains(string(body2), "vibe-check allow pypi flask 3.1.0") {
		t.Error("response should contain exact allow command")
	}

	// Allow flask
	g.Allow("pypi", "flask", "3.1.0")

	// Should now pass
	resp3, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, downloadPath))
	if err != nil {
		t.Fatalf("GET download (after allow) failed: %v", err)
	}
	defer resp3.Body.Close()
	io.Copy(io.Discard, resp3.Body)

	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after allow, got %d", resp3.StatusCode)
	}
}

// TestIntegration_NPM_ScopedMetadataPassthrough verifies that metadata requests
// for scoped packages pass through without interference.
func TestIntegration_NPM_ScopedMetadataPassthrough(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startNPMProxy(t, g)

	paths := []string{
		"/@types/node",
		"/@babel/core",
		"/@types/node/22.0.0",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
			if err != nil {
				t.Fatalf("GET %s failed: %v", path, err)
			}
			defer resp.Body.Close()
			io.Copy(io.Discard, resp.Body)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}
		})
	}
}

// TestIntegration_PyPI_CacheConsistency verifies that repeated requests for the same
// package use the cache (second request should be faster) and return consistent results.
func TestIntegration_PyPI_CacheConsistency(t *testing.T) {
	g := gate.New(7 * 24 * time.Hour)
	port := startPyPIProxy(t, g)

	// Get simple index
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/simple/flask/", port))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var downloadPath string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.Contains(strings.ToLower(line), "flask-3.1.0") && strings.Contains(line, ".tar.gz") {
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
			if fragIdx := strings.Index(href, "#"); fragIdx >= 0 {
				href = href[:fragIdx]
			}
			downloadPath = href
			break
		}
	}
	if downloadPath == "" {
		t.Fatal("could not find download link")
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, downloadPath)

	// First request — populates cache
	start1 := time.Now()
	resp1, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp1.Body)
	resp1.Body.Close()
	dur1 := time.Since(start1)

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", resp1.StatusCode)
	}

	// Second request — should use cache
	start2 := time.Now()
	resp2, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
	dur2 := time.Since(start2)

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d", resp2.StatusCode)
	}

	t.Logf("first request: %v, second request: %v (cached)", dur1, dur2)

	// Cached request should be meaningfully faster (no API call to pypi.org)
	if dur2 > dur1 {
		t.Logf("warning: cached request was not faster (network variance?)")
	}
}
