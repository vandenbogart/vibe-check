package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/ebo/vibe-check/internal/gate"
	"github.com/ebo/vibe-check/internal/registry"
)

const (
	pypiHost  = "pypi.org"
	filesHost = "files.pythonhosted.org"
)

var pypiURLRewriteRe = regexp.MustCompile(`https://files\.pythonhosted\.org/`)

// ParsePyPIFilename extracts the package name and version from a PyPI filename.
// It handles .tar.gz, .zip (sdist) and .whl (wheel) files.
// The name is normalized to lowercase with underscores replaced by hyphens.
func ParsePyPIFilename(filename string) (pkg, version string, ok bool) {
	if filename == "" {
		return "", "", false
	}

	// Remove URL fragment (#sha256=...)
	if idx := strings.Index(filename, "#"); idx >= 0 {
		filename = filename[:idx]
	}

	// Strip .metadata suffix (PEP 658 — pip requests metadata before full download)
	filename = strings.TrimSuffix(filename, ".metadata")

	// Determine type and strip extension
	var base string
	switch {
	case strings.HasSuffix(filename, ".whl"):
		base = strings.TrimSuffix(filename, ".whl")
		return parseWheel(base)
	case strings.HasSuffix(filename, ".tar.gz"):
		base = strings.TrimSuffix(filename, ".tar.gz")
		return parseSdist(base)
	case strings.HasSuffix(filename, ".zip"):
		base = strings.TrimSuffix(filename, ".zip")
		return parseSdist(base)
	default:
		return "", "", false
	}
}

// parseWheel parses a wheel filename: {name}-{version}-{python}-{abi}-{platform}
func parseWheel(base string) (string, string, bool) {
	parts := strings.SplitN(base, "-", 3)
	if len(parts) < 3 {
		return "", "", false
	}
	name := normalizeName(parts[0])
	ver := parts[1]
	if ver == "" {
		return "", "", false
	}
	return name, ver, true
}

// parseSdist parses an sdist filename: {name}-{version}
// Finds the last hyphen before a digit to split name and version.
func parseSdist(base string) (string, string, bool) {
	// Walk backwards to find the last '-' followed by a digit
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '-' && i+1 < len(base) && unicode.IsDigit(rune(base[i+1])) {
			name := normalizeName(base[:i])
			ver := base[i+1:]
			if name == "" || ver == "" {
				return "", "", false
			}
			return name, ver, true
		}
	}
	return "", "", false
}

// normalizeName lowercases and replaces underscores with hyphens.
func normalizeName(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "_", "-")
}

// IsPyPIDownloadPath returns true if the path is a PyPI package download path.
func IsPyPIDownloadPath(p string) bool {
	return strings.HasPrefix(p, "/packages/")
}

// RewritePyPISimpleHTML replaces absolute files.pythonhosted.org URLs with
// relative URLs so that download requests come back through the proxy.
func RewritePyPISimpleHTML(html string) string {
	return pypiURLRewriteRe.ReplaceAllString(html, "/")
}

// PyPIProxy is a reverse proxy for PyPI that applies age-gating to downloads.
type PyPIProxy struct {
	gate     *gate.Gate
	registry *registry.PyPI
	logger   *slog.Logger
}

// NewPyPIProxy creates a new PyPI proxy with the given gate, registry client, and logger.
func NewPyPIProxy(g *gate.Gate, reg *registry.PyPI, logger *slog.Logger) *PyPIProxy {
	return &PyPIProxy{
		gate:     g,
		registry: reg,
		logger:   logger,
	}
}

// Handler returns an http.Handler that routes requests to the appropriate
// upstream PyPI server with age-gating applied to package downloads.
func (p *PyPIProxy) Handler() http.Handler {
	pypiTarget, _ := url.Parse("https://" + pypiHost)
	filesTarget, _ := url.Parse("https://" + filesHost)

	pypiProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = pypiTarget.Scheme
			req.URL.Host = pypiTarget.Host
			req.Host = pypiTarget.Host
			// Remove Accept-Encoding so upstream returns uncompressed HTML,
			// allowing ModifyResponse to rewrite URLs in the body.
			req.Header.Del("Accept-Encoding")
		},
		ModifyResponse: func(resp *http.Response) error {
			// Only rewrite HTML responses from the simple API
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/vnd.pypi.simple") {
				return nil
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}

			rewritten := RewritePyPISimpleHTML(string(body))
			resp.Body = io.NopCloser(bytes.NewReader([]byte(rewritten)))
			resp.ContentLength = int64(len(rewritten))
			resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
			return nil
		},
	}

	filesProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = filesTarget.Scheme
			req.URL.Host = filesTarget.Host
			req.Host = filesTarget.Host
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsPyPIDownloadPath(r.URL.Path) {
			p.handleDownload(w, r, filesProxy)
			return
		}
		p.handleSimple(w, r, pypiProxy)
	})
}

func (p *PyPIProxy) handleSimple(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	p.logger.Log(context.Background(), slog.Level(-8), "pypi simple request",
		"path", r.URL.Path,
	)
	proxy.ServeHTTP(w, r)
}

func (p *PyPIProxy) handleDownload(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	filename := path.Base(r.URL.Path)
	pkg, version, ok := ParsePyPIFilename(filename)

	p.logger.Log(context.Background(), slog.Level(-8), "pypi download request",
		"path", r.URL.Path,
		"filename", filename,
		"pkg", pkg,
		"version", version,
		"parsed", ok,
	)

	// Fail closed: if we can't parse the filename, block the download
	if !ok {
		p.logger.Warn("pypi: unparseable filename, blocking download",
			"filename", filename,
		)
		http.Error(w, fmt.Sprintf("BLOCKED: could not identify package from download path: %s\nUnable to verify package age. Download blocked for safety.", r.URL.Path), http.StatusForbidden)
		return
	}

	// Check allowlist
	if p.gate.IsAllowed("pypi", pkg, version) {
		p.logger.Info("pypi: allowlisted, proxying through",
			"pkg", pkg,
			"version", version,
		)
		proxy.ServeHTTP(w, r)
		return
	}

	// Check cache first, then fetch publish date
	published, cached := p.gate.GetCachedPublishDate("pypi", pkg, version)
	if !cached {
		var err error
		published, err = p.registry.FetchPublishDate(r.Context(), pkg, version)
		if err != nil {
			// Fail closed: if we can't determine the publish date, block
			p.logger.Error("pypi: failed to fetch publish date",
				"pkg", pkg,
				"version", version,
				"error", err,
			)
			http.Error(w, fmt.Sprintf("failed to fetch publish date for %s@%s: %v", pkg, version, err), http.StatusBadGateway)
			return
		}
		p.gate.CachePublishDate("pypi", pkg, version, published)
	}

	// Check age
	result := p.gate.CheckAge("pypi", pkg, version, published)
	if result.Blocked {
		p.logger.Warn("pypi: blocked by age gate",
			"pkg", pkg,
			"version", version,
			"age_days", int(result.Age.Hours()/24),
			"min_age_days", int(result.MinAge.Hours()/24),
		)
		http.Error(w, result.Message(), http.StatusForbidden)
		return
	}

	p.logger.Info("pypi: age check passed, proxying through",
		"pkg", pkg,
		"version", version,
	)
	proxy.ServeHTTP(w, r)
}
