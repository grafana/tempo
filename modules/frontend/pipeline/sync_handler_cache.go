package pipeline

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/cache"
)

var cacheKey = struct{}{}

func NewCachingWare(cacheProvider cache.Provider, role cache.Role, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return cachingWare{
			next:  next,
			cache: newFrontendCache(cacheProvider, role, logger),
		}
	})
}

type cachingWare struct {
	next  http.RoundTripper
	cache *frontendCache
}

// RoundTrip implements http.RoundTripper
func (c cachingWare) RoundTrip(req *http.Request) (*http.Response, error) {
	// short circuit everything if there cache is no cache
	if c.cache == nil {
		return c.next.RoundTrip(req)
	}

	// extract cache key
	key, ok := req.Context().Value(cacheKey).(string)
	if ok && len(key) > 0 {
		body := c.cache.fetchBytes(key)
		if len(body) > 0 {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(bytes.NewBuffer(body)),
			}, nil
		}
	}

	resp, err := c.next.RoundTrip(req)
	// do not cache if there was an error
	if err != nil {
		return resp, err
	}

	// do not cache if response is not HTTP 2xx
	if !shouldCache(resp.StatusCode) {
		return resp, nil
	}

	if len(key) > 0 {
		// cache the response
		//  todo: currently this is blindly caching any 200 status codes. it would be a bug, but it's possible for a querier
		//  to return a 200 status code with a response that does not parse as the expected type in the combiner.
		//  technically this should never happen...
		//  long term we should migrate the sync part of the pipeline to use generics so we can do the parsing early in the pipeline
		//  and be confident it's cacheable.
		b, err := io.ReadAll(resp.Body)

		// reset the body so the caller can read it
		resp.Body = io.NopCloser(bytes.NewBuffer(b))
		if err != nil {
			return resp, nil
		}

		c.cache.store(req.Context(), key, b)
	}

	return resp, nil
}

func shouldCache(statusCode int) bool {
	return statusCode/100 == 2
}

func AddCacheKey(key string, req *http.Request) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), cacheKey, key))
}

type frontendCache struct {
	c cache.Cache
}

func newFrontendCache(cacheProvider cache.Provider, role cache.Role, logger log.Logger) *frontendCache {
	var c cache.Cache
	if cacheProvider != nil {
		c = cacheProvider.CacheFor(role)
	}

	level.Info(logger).Log("msg", "init frontend cache", "role", role, "enabled", c != nil)

	if c == nil {
		return nil
	}

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

// fetch fetches the response body from the cache. the caller assumes the responsibility of closing the response body.
func (c *frontendCache) fetchBytes(key string) []byte {
	if c.c == nil {
		return nil
	}

	if len(key) == 0 {
		return nil
	}

	_, bufs, _ := c.c.Fetch(context.Background(), []string{key})
	if len(bufs) != 1 {
		return nil
	}

	return bufs[0]
}
