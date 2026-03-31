package registry

import (
	"context"
	"time"
)

// Client defines the interface for fetching package publish dates from registries.
type Client interface {
	FetchPublishDate(ctx context.Context, pkg, version string) (time.Time, error)
}
