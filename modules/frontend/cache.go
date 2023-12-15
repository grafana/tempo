package frontend

import (
	"bytes"
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/cache"
)

type frontendCache struct {
	c cache.Cache
}

func newFrontendCache(cacheProvider cache.Provider, role cache.Role, logger log.Logger) *frontendCache {
	var c cache.Cache
	if cacheProvider != nil {
		c = cacheProvider.CacheFor(role)
	}

	level.Info(logger).Log("msg", "init frontend cache", "role", role, "enabled", c != nil)

	return &frontendCache{
		c: c,
	}
}

// store stores the response body in the cache. the caller assumes the responsibility of closing the response body
func (c *frontendCache) store(ctx context.Context, key string, buffer []byte) {
	if c.c == nil {
		return
	}

	if key == "" {
		return
	}

	if len(buffer) == 0 {
		return
	}

	c.c.Store(ctx, []string{key}, [][]byte{buffer})
}

// fetch fetches the response body from the cache. the caller assumes the responsibility of closing the response body.
func (c *frontendCache) fetch(key string, pb proto.Message) bool {
	if c.c == nil {
		return false
	}

	if len(key) == 0 {
		return false
	}

	_, bufs, _ := c.c.Fetch(context.Background(), []string{key})
	if len(bufs) != 1 {
		return false
	}

	err := (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(bytes.NewReader(bufs[0]), pb)
	if err != nil {
		return false
	}

	return true
}
