package cache

import (
	"context"

	"github.com/grafana/dskit/services"
)

type Role string

const (
	// individual roles
	RoleNone             Role = "none"
	RoleBloom            Role = "bloom"
	RoleTraceIDIdx       Role = "trace-id-index"
	RoleParquetFooter    Role = "parquet-footer"
	RoleParquetColumnIdx Role = "parquet-column-idx"
	RoleParquetOffsetIdx Role = "parquet-offset-idx"
	RoleFrontendSearch   Role = "frontend-search"
	RoleParquetPage      Role = "parquet-page"
)

// Provider is an object that can return a cache for a requested role
type Provider interface {
	services.Service

	CacheFor(role Role) Cache
	AddCache(role Role, c Cache) error
}

// Cache byte arrays by key.
//
// NB we intentionally do not return errors in this interface - caching is best
// effort by definition.  We found that when these methods did return errors,
// the caller would just log them - so its easier for implementation to do that.
// Whatsmore, we found partially successful Fetchs were often treated as failed
// when they returned an error.
type Cache interface {
	Store(ctx context.Context, key []string, buf [][]byte)
	// TODO: both cached backend clients support deletion. Should we implement?
	// Remove(ctx context.Context, key []string)
	Fetch(ctx context.Context, keys []string) (found []string, bufs [][]byte, missing []string)
	Stop()
}
