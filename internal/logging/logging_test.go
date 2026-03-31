package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLevelParse(t *testing.T) {
	tests := []struct {
		input   string
		want    slog.Level
		wantErr bool
	}{
		{"info", slog.LevelInfo, false},
		{"debug", slog.LevelDebug, false},
		{"trace", LevelTrace, false},
		{"INFO", slog.LevelInfo, false},
		{"bogus", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseLevel(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseLevel(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, slog.LevelInfo)

	// Debug message should be hidden at info level.
	logger.Debug("should be hidden")
	if strings.Contains(buf.String(), "should be hidden") {
		t.Fatal("debug message should not appear at info level")
	}

	// Change to debug level.
	SetLevel(slog.LevelDebug)

	buf.Reset()
	logger.Debug("should be visible")
	if !strings.Contains(buf.String(), "should be visible") {
		t.Fatal("debug message should appear at debug level")
	}
}

func TestTraceLevelBelowDebug(t *testing.T) {
	if LevelTrace >= slog.LevelDebug {
		t.Fatalf("LevelTrace (%d) should be below slog.LevelDebug (%d)", LevelTrace, slog.LevelDebug)
	}
}
