package db

import (
	"context"

	"github.com/shammianand/queryit/internal/cache"
)

// Driver is the interface every database backend must implement.
// Each Tab holds one Driver. Implementations must be safe to call
// concurrently (i.e. acquire their own connection per operation).
type Driver interface {
	// Execute runs a query and returns a ResultSet. Errors from the
	// database itself are returned inside ResultSet.IsError, not as
	// a Go error, so the TUI can display them inline.
	Execute(ctx context.Context, query string) (*ResultSet, error)

	// Introspect fetches the live schema and returns a snapshot.
	Introspect(ctx context.Context) (*cache.SchemaSnapshot, error)

	// Ping checks connectivity.
	Ping(ctx context.Context) error

	// Close releases all resources (pool, tunnel, file handles).
	Close()

	// DriverName returns the canonical driver identifier.
	DriverName() string
}
