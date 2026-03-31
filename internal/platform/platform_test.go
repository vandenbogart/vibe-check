package platform

import (
	"strings"
	"testing"
)

func testOpts() ServiceOpts {
	return ServiceOpts{
		BinaryPath: "/usr/local/bin/vibe-check",
		DataDir:    "/home/user/.vibe-check",
		PyPIPort:   3141,
		NPMPort:    3142,
		MinAge:     "7d",
		LogLevel:   "info",
	}
}

func TestLaunchdPlist(t *testing.T) {
	opts := testOpts()
	plist := GenerateLaunchdPlist(opts)

	checks := []string{
		opts.BinaryPath,
		"com.vibe-check",
		"RunAtLoad",
		"3141",
		"3142",
		"serve",
		"KeepAlive",
		"7d",
		"/home/user/.vibe-check/vibe-check.log",
	}
	for _, want := range checks {
		if !strings.Contains(plist, want) {
			t.Errorf("plist missing %q\n\nGot:\n%s", want, plist)
		}
	}
}

func TestSystemdUnit(t *testing.T) {
	opts := testOpts()
	unit := GenerateSystemdUnit(opts)

	checks := []string{
		opts.BinaryPath,
		"serve",
		"WantedBy=default.target",
		"--pypi-port 3141",
		"--npm-port 3142",
		"--min-age 7d",
		"--data-dir /home/user/.vibe-check",
		"--log-level info",
		"Restart=on-failure",
		"RestartSec=5",
		"After=network.target",
	}
	for _, want := range checks {
		if !strings.Contains(unit, want) {
			t.Errorf("unit missing %q\n\nGot:\n%s", want, unit)
		}
	}
}

func TestDetect(t *testing.T) {
	mgr := Detect()
	if mgr == nil {
		t.Skip("unsupported platform for Detect()")
	}
	// On darwin we expect Launchd, on linux Systemd
	switch mgr.(type) {
	case *Launchd, *Systemd:
		// ok
	default:
		t.Errorf("unexpected ServiceManager type: %T", mgr)
	}
}

func TestFindBinary(t *testing.T) {
	path, err := FindBinary()
	if err != nil {
		t.Fatalf("FindBinary() error: %v", err)
	}
	if path == "" {
		t.Error("FindBinary() returned empty path")
	}
}
