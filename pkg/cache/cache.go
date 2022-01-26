package cache

import (
	"context"
)

// Cache byte arrays by key.
//
// NB we intentionally do not return errors in this interface - caching is best
// effort by definition.  We found that when these methods did return errors,
// the caller would just log them - so its easier for implementation to do that.
// Whatsmore, we found partially successful Fetchs were often treated as failed
// when they returned an error.
type Cache interface {
	Store(ctx context.Context, key []string, buf [][]byte)
	Fetch(ctx context.Context, keys []string) (found []string, bufs [][]byte, missing []string)
	Stop()
}
