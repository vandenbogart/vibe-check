package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// LevelTrace is a custom log level below debug, used for full request/response details.
const LevelTrace = slog.Level(-8)

// levelVar is a package-level variable for atomic level switching.
var levelVar slog.LevelVar

// New creates a JSON-handler logger at the given level and sets the package-level LevelVar.
// The handler renders LevelTrace as "TRACE" in output.
func New(w io.Writer, level slog.Level) *slog.Logger {
	levelVar.Set(level)

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: &levelVar,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				lvl := a.Value.Any().(slog.Level)
				if lvl == LevelTrace {
					a.Value = slog.StringValue("TRACE")
				}
			}
			return a
		},
	})

	return slog.New(handler)
}

// SetLevel atomically changes the log level at runtime.
func SetLevel(l slog.Level) {
	levelVar.Set(l)
}

// GetLevel returns the current log level.
func GetLevel() slog.Level {
	return levelVar.Level()
}

// ParseLevel converts a case-insensitive string to a slog.Level.
// Supported values: "trace", "debug", "info".
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	default:
		return 0, fmt.Errorf("unknown log level: %q", s)
	}
}
