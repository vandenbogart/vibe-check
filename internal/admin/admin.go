package admin

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"

	"github.com/ebo/vibe-check/internal/config"
	"github.com/ebo/vibe-check/internal/gate"
	"github.com/ebo/vibe-check/internal/logging"
)

// AllowRequest is the JSON body for POST /admin/allow.
type AllowRequest struct {
	Registry string `json:"registry"`
	Package  string `json:"package"`
	Version  string `json:"version"`
}

// ConfigRequest is the JSON body for PUT /admin/config.
type ConfigRequest struct {
	MinAge string `json:"min_age"`
}

// LogLevelRequest is the JSON body for PUT /admin/log-level.
type LogLevelRequest struct {
	Level string `json:"level"`
}

// StatusResponse is returned by GET /admin/status.
type StatusResponse struct {
	MinAge         string `json:"min_age"`
	AllowlistCount int    `json:"allowlist_count"`
	LogLevel       string `json:"log_level"`
}

// Server provides an admin API over a Unix socket for runtime configuration.
type Server struct {
	gate    *gate.Gate
	cfg     *config.Config
	dataDir string
	logger  *slog.Logger
	server  *http.Server
}

// NewServer creates an admin Server wired to the given gate and config.
func NewServer(g *gate.Gate, cfg *config.Config, dataDir string, logger *slog.Logger) *Server {
	s := &Server{
		gate:    g,
		cfg:     cfg,
		dataDir: dataDir,
		logger:  logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/allow", s.handleAllow)
	mux.HandleFunc("PUT /admin/config", s.handleConfig)
	mux.HandleFunc("PUT /admin/log-level", s.handleLogLevel)
	mux.HandleFunc("GET /admin/status", s.handleStatus)

	s.server = &http.Server{Handler: mux}
	return s
}

// Serve starts serving on the provided listener (typically a Unix socket).
func (s *Server) Serve(ln net.Listener) error {
	return s.server.Serve(ln)
}

// Close gracefully shuts down the server.
func (s *Server) Close() error {
	return s.server.Close()
}

func (s *Server) handleAllow(w http.ResponseWriter, r *http.Request) {
	var req AllowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.gate.Allow(req.Registry, req.Package, req.Version)

	if err := s.gate.SaveAllowlist(filepath.Join(s.dataDir, "allowlist.txt")); err != nil {
		s.logger.Error("failed to persist allowlist", "error", err)
		http.Error(w, "failed to persist allowlist", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	d, err := config.ParseMinAge(req.MinAge)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.gate.SetMinAge(d)
	s.cfg.MinAge = d
	s.cfg.MinAgeS = req.MinAge

	if err := config.Save(*s.cfg, filepath.Join(s.dataDir, "config.json")); err != nil {
		s.logger.Error("failed to persist config", "error", err)
		http.Error(w, "failed to persist config", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleLogLevel(w http.ResponseWriter, r *http.Request) {
	var req LogLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lvl, err := logging.ParseLevel(req.Level)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logging.SetLevel(lvl)
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	entries := s.gate.AllowlistEntries()
	minAge := config.FormatMinAge(s.gate.MinAge())
	lvl := logging.GetLevel()

	var levelStr string
	switch lvl {
	case logging.LevelTrace:
		levelStr = "trace"
	case slog.LevelDebug:
		levelStr = "debug"
	case slog.LevelInfo:
		levelStr = "info"
	case slog.LevelWarn:
		levelStr = "warn"
	case slog.LevelError:
		levelStr = "error"
	default:
		levelStr = lvl.String()
	}

	writeJSON(w, StatusResponse{
		MinAge:         minAge,
		AllowlistCount: len(entries),
		LogLevel:       levelStr,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
