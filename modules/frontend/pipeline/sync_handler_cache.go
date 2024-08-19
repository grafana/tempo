package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
)

func NewCachingWare(cacheProvider cache.Provider, role cache.Role, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next RoundTripper) RoundTripper {
		return cachingWare{
			next:  next,
			cache: newFrontendCache(cacheProvider, role, logger),
		}
	})
}

type cachingWare struct {
	next  RoundTripper
	cache *frontendCache
}

// RoundTrip implements http.RoundTripper
func (c cachingWare) RoundTrip(req Request) (*http.Response, error) {
	// short circuit everything if there cache is no cache
	if c.cache == nil {
		return c.next.RoundTrip(req)
	}

	// extract cache key
	key := req.CacheKey()
	if len(key) > 0 {
		body := c.cache.fetchBytes(key)
		if len(body) > 0 {
			resp := &http.Response{
				Header:     http.Header{},
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(bytes.NewBuffer(body)),
			}

			// We aren't capturing the original content type in the cache, just the raw bytes.
			// Detect it and readd it, so the upstream code can parse the body.
			// TODO - Cache should capture all of the relevant parts of the
			// original response including both content-type and content-length headers, possibly more.
			// But upgrading the cache format requires migration/detection of previous format either way.
			// It's tempting to use https://pkg.go.dev/net/http#DetectContentType but it doesn't detect
			// json or proto.
			if body[0] == '{' {
				resp.Header.Add(api.HeaderContentType, api.HeaderAcceptJSON)
			} else {
				resp.Header.Add(api.HeaderContentType, api.HeaderAcceptProtobuf)
			}
			return resp, nil
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
		// don't bother caching if the response is too large
		if c.cache.c.MaxItemSize() != 0 && resp.ContentLength > int64(c.cache.c.MaxItemSize()) {
			return resp, nil
		}

		buffer, err := api.ReadBodyToBuffer(resp)
		if err != nil {
			return resp, fmt.Errorf("failed to cache: %w", err)
		}

		// reset the body so the caller can read it
		resp.Body = io.NopCloser(buffer)

		// cache the response
		//  todo: currently this is blindly caching any 200 status codes. it would be a bug, but it's possible for a querier
		//  to return a 200 status code with a response that does not parse as the expected type in the combiner.
		//  technically this should never happen...
		//  long term we should migrate the sync part of the pipeline to use generics so we can do the parsing early in the pipeline
		//  and be confident it's cacheable.
		c.cache.store(req.Context(), key, buffer.Bytes())
	}

	return resp, nil
}

func shouldCache(statusCode int) bool {
	return statusCode/100 == 2
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
	return err == nil
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
