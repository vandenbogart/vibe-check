package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/ebo/vibe-check/internal/gate"
	"github.com/ebo/vibe-check/internal/registry"
)

const npmHost = "registry.npmjs.org"

// ParseNPMTarballPath extracts the package name and version from an npm
// tarball download path.
//
// Unscoped: /name/-/name-version.tgz
// Scoped:   /@scope/name/-/name-version.tgz
//
// Returns the full package name (including scope for scoped packages),
// the version string, and true if parsing succeeded.
func ParseNPMTarballPath(path string) (pkg, version string, ok bool) {
	if !strings.HasSuffix(path, ".tgz") {
		return "", "", false
	}

	// Strip leading slash
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")

	// Unscoped: [name, -, name-version.tgz]
	// Scoped:   [@scope, name, -, name-version.tgz]
	var fullName string
	var shortName string
	var tarball string

	switch {
	case len(parts) == 3 && parts[1] == "-":
		// Unscoped: name/-/name-version.tgz
		fullName = parts[0]
		shortName = parts[0]
		tarball = parts[2]
	case len(parts) == 4 && strings.HasPrefix(parts[0], "@") && parts[2] == "-":
		// Scoped: @scope/name/-/name-version.tgz
		fullName = parts[0] + "/" + parts[1]
		shortName = parts[1]
		tarball = parts[3]
	default:
		return "", "", false
	}

	// Extract version from tarball: strip .tgz, then strip shortName- prefix
	base := strings.TrimSuffix(tarball, ".tgz")
	prefix := shortName + "-"
	if !strings.HasPrefix(base, prefix) {
		return "", "", false
	}
	version = base[len(prefix):]
	if version == "" {
		return "", "", false
	}

	return fullName, version, true
}

// IsNPMTarballPath returns true if the path is an npm tarball download path.
func IsNPMTarballPath(path string) bool {
	_, _, ok := ParseNPMTarballPath(path)
	return ok
}

// NPMProxy is a reverse proxy for npm that applies age-gating to tarball downloads.
type NPMProxy struct {
	gate     *gate.Gate
	registry *registry.NPM
	logger   *slog.Logger
}

// NewNPMProxy creates a new npm proxy with the given gate, registry client, and logger.
func NewNPMProxy(g *gate.Gate, reg *registry.NPM, logger *slog.Logger) *NPMProxy {
	return &NPMProxy{
		gate:     g,
		registry: reg,
		logger:   logger,
	}
}

// Handler returns an http.Handler that routes requests to the npm registry
// with age-gating applied to tarball downloads.
func (n *NPMProxy) Handler() http.Handler {
	target, _ := url.Parse("https://" + npmHost)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsNPMTarballPath(r.URL.Path) {
			n.handleTarball(w, r, proxy)
			return
		}
		n.handlePassthrough(w, r, proxy)
	})
}

func (n *NPMProxy) handlePassthrough(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	n.logger.Log(context.Background(), slog.Level(-8), "npm metadata request",
		"path", r.URL.Path,
	)
	proxy.ServeHTTP(w, r)
}

func (n *NPMProxy) handleTarball(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	pkg, version, ok := ParseNPMTarballPath(r.URL.Path)

	n.logger.Log(context.Background(), slog.Level(-8), "npm tarball request",
		"path", r.URL.Path,
		"pkg", pkg,
		"version", version,
		"parsed", ok,
	)

	// Fail closed: if we can't parse the path, block the download
	if !ok {
		n.logger.Warn("npm: unparseable tarball path, blocking download",
			"path", r.URL.Path,
		)
		http.Error(w, fmt.Sprintf("BLOCKED: could not identify package from download path: %s\nUnable to verify package age. Download blocked for safety.", r.URL.Path), http.StatusForbidden)
		return
	}

	// Check allowlist
	if n.gate.IsAllowed("npm", pkg, version) {
		n.logger.Info("npm: allowlisted, proxying through",
			"pkg", pkg,
			"version", version,
		)
		proxy.ServeHTTP(w, r)
		return
	}

	// Check cache first, then fetch publish date
	published, cached := n.gate.GetCachedPublishDate("npm", pkg, version)
	if !cached {
		var err error
		published, err = n.registry.FetchPublishDate(r.Context(), pkg, version)
		if err != nil {
			// Fail closed: if we can't determine the publish date, block
			n.logger.Error("npm: failed to fetch publish date",
				"pkg", pkg,
				"version", version,
				"error", err,
			)
			http.Error(w, fmt.Sprintf("failed to fetch publish date for %s@%s: %v", pkg, version, err), http.StatusBadGateway)
			return
		}
		n.gate.CachePublishDate("npm", pkg, version, published)
	}

	// Check age
	result := n.gate.CheckAge("npm", pkg, version, published)
	if result.Blocked {
		n.logger.Warn("npm: blocked by age gate",
			"pkg", pkg,
			"version", version,
			"age_days", int(result.Age.Hours()/24),
			"min_age_days", int(result.MinAge.Hours()/24),
		)
		http.Error(w, result.Message(), http.StatusForbidden)
		return
	}

	n.logger.Info("npm: age check passed, proxying through",
		"pkg", pkg,
		"version", version,
	)
	proxy.ServeHTTP(w, r)
}
