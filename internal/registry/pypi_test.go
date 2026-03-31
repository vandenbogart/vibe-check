package registry

import (
	"context"
	"testing"
	"time"
)

func TestPyPIFetchPublishDate_Flask(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := NewPyPI()
	ctx := context.Background()

	publishDate, err := client.FetchPublishDate(ctx, "flask", "3.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Flask 3.1.0 published around 2024-11-13
	expected := time.Date(2024, 11, 13, 0, 0, 0, 0, time.UTC)
	diff := publishDate.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > 24*time.Hour {
		t.Errorf("publish date %v not within 24h of expected %v (diff: %v)", publishDate, expected, diff)
	}
}

func TestPyPIFetchPublishDate_NonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := NewPyPI()
	ctx := context.Background()

	_, err := client.FetchPublishDate(ctx, "this-package-does-not-exist-xyz-abc-123", "0.0.0")
	if err == nil {
		t.Fatal("expected error for non-existent package, got nil")
	}
}
