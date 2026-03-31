package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ebo/vibe-check/internal/admin"
	"github.com/ebo/vibe-check/internal/config"
	"github.com/ebo/vibe-check/internal/gate"
	"github.com/ebo/vibe-check/internal/logging"
	"github.com/ebo/vibe-check/internal/proxy"
	"github.com/ebo/vibe-check/internal/registry"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the proxy server",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntP("pypi-port", "", 3141, "Port for the PyPI proxy")
	serveCmd.Flags().IntP("npm-port", "", 3142, "Port for the npm proxy")
	serveCmd.Flags().StringP("min-age", "", "7d", "Minimum package age before allowing installation")
	serveCmd.Flags().StringP("data-dir", "", "", "Directory for persistent data")
	serveCmd.Flags().StringP("log-level", "", "info", "Log level (debug, info, warn, error)")
}

func runServe(cmd *cobra.Command, args []string) error {
	// 1. Parse flags
	pypiPort, _ := cmd.Flags().GetInt("pypi-port")
	npmPort, _ := cmd.Flags().GetInt("npm-port")
	minAgeStr, _ := cmd.Flags().GetString("min-age")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	logLevelStr, _ := cmd.Flags().GetString("log-level")

	if dataDir == "" {
		dataDir = config.DefaultDataDir()
	}
	os.MkdirAll(dataDir, 0o755)

	// 2. Setup logger
	logLevel, err := logging.ParseLevel(logLevelStr)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	logger := logging.New(os.Stdout, logLevel)

	// 3. Load config from disk, CLI flags override saved config
	cfgPath := filepath.Join(dataDir, "config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// If flags were explicitly set, override loaded config
	if cmd.Flags().Changed("min-age") {
		d, err := config.ParseMinAge(minAgeStr)
		if err != nil {
			return fmt.Errorf("invalid min-age: %w", err)
		}
		cfg.MinAge = d
		cfg.MinAgeS = minAgeStr
	}
	if cmd.Flags().Changed("pypi-port") {
		cfg.PyPIPort = pypiPort
	} else {
		pypiPort = cfg.PyPIPort
	}
	if cmd.Flags().Changed("npm-port") {
		cfg.NPMPort = npmPort
	} else {
		npmPort = cfg.NPMPort
	}

	// 4. Create gate, load allowlist
	g := gate.New(cfg.MinAge)
	allowPath := filepath.Join(dataDir, "allowlist.txt")
	if err := g.LoadAllowlist(allowPath); err != nil {
		return fmt.Errorf("loading allowlist: %w", err)
	}

	// 5. Create registry clients
	pypiReg := registry.NewPyPI()
	npmReg := registry.NewNPM()

	// 6. Create proxy handlers
	pypiProxy := proxy.NewPyPIProxy(g, pypiReg, logger)
	npmProxy := proxy.NewNPMProxy(g, npmReg, logger)

	// 7. Create HTTP servers
	pypiServer := &http.Server{Addr: fmt.Sprintf(":%d", pypiPort), Handler: pypiProxy.Handler()}
	npmServer := &http.Server{Addr: fmt.Sprintf(":%d", npmPort), Handler: npmProxy.Handler()}

	// 8. Setup admin unix socket
	sockPath := filepath.Join(dataDir, "vibe-check.sock")
	os.Remove(sockPath) // clean up stale socket
	adminLn, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("creating admin socket: %w", err)
	}
	adminSrv := admin.NewServer(g, &cfg, dataDir, logger)

	// 9. Start all 3 servers in goroutines
	errCh := make(chan error, 3)
	go func() { errCh <- pypiServer.ListenAndServe() }()
	go func() { errCh <- npmServer.ListenAndServe() }()
	go func() { errCh <- adminSrv.Serve(adminLn) }()

	// 10. Log startup
	logger.Info("vibe-check started", "pypi_port", pypiPort, "npm_port", npmPort,
		"min_age", config.FormatMinAge(cfg.MinAge), "data_dir", dataDir)

	// 11. Wait for SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logger.Info("shutting down...")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// 12. Graceful shutdown
	pypiServer.Shutdown(context.Background())
	npmServer.Shutdown(context.Background())
	adminSrv.Close()
	adminLn.Close()
	os.Remove(sockPath)

	return nil
}
