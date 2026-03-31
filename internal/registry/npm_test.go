package registry

import (
	"context"
	"testing"
	"time"
)

func TestNPMFetchPublishDate_Express(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := NewNPM()
	ctx := context.Background()

	publishDate, err := client.FetchPublishDate(ctx, "express", "4.21.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// express 4.21.0 published around 2024-09-11
	expected := time.Date(2024, 9, 11, 0, 0, 0, 0, time.UTC)
	diff := publishDate.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > 48*time.Hour {
		t.Errorf("publish date %v not within 48h of expected %v (diff: %v)", publishDate, expected, diff)
	}
}

func TestNPMFetchPublishDate_ScopedPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := NewNPM()
	ctx := context.Background()

	publishDate, err := client.FetchPublishDate(ctx, "@types/node", "22.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if publishDate.IsZero() {
		t.Error("expected non-zero publish date for @types/node 22.0.0")
	}
}

func TestNPMFetchPublishDate_NonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := NewNPM()
	ctx := context.Background()

	_, err := client.FetchPublishDate(ctx, "this-package-does-not-exist-xyz-abc-123", "0.0.0")
	if err == nil {
		t.Fatal("expected error for non-existent package, got nil")
	}
}
