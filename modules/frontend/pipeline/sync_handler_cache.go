package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/proto" //nolint:all
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	tempopb "github.com/grafana/tempo/pkg/tempopb"
)

type CacheableRespType int

const (
	RespTypeUncacheable CacheableRespType = iota
	RespTypeSearch
	RespTypeTags
	RespTypeTagValues
	RespTypeQueryRange
	RespTypeQueryInstant
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
		body := c.cache.fetchBytes(req.Context(), key)
		if len(body) > 0 {
			responseType := determineResponseType(req)

			// Determine content type based on first byte
			var contentType string
			// TODO - Cache should capture all of the relevant parts of the
			// original response including both content-type and content-length headers, possibly more.
			// But upgrading the cache format requires migration/detection of previous format either way.
			// It's tempting to use https://pkg.go.dev/net/http#DetectContentType but it doesn't detect
			// json or proto.
			if body[0] == '{' {
				contentType = api.HeaderAcceptJSON
			} else {
				contentType = api.HeaderAcceptProtobuf
			}

			if newBody, err := resetMetricsInResponse(body, responseType, contentType); err == nil {
				body = newBody
			}

			resp := &http.Response{
				Header:        http.Header{},
				StatusCode:    http.StatusOK,
				Status:        http.StatusText(http.StatusOK),
				Body:          io.NopCloser(bytes.NewBuffer(body)),
				ContentLength: int64(len(body)),
			}

			resp.Header.Add(api.HeaderContentType, contentType)

			resp.Header.Add("X-Tempo-Cache", "HIT")

			return resp, nil
		}
	}

	resp, err := c.next.RoundTrip(req)

	// Add cache miss header for all responses that weren't from cache
	if resp != nil && resp.Header != nil {
		resp.Header.Add("X-Tempo-Cache", "MISS")
	}

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
		maxItemSize := c.cache.c.MaxItemSize()
		if maxItemSize > 0 && resp.ContentLength > int64(maxItemSize) {
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

func resetMetricsInResponse(body []byte, responseType CacheableRespType, contentType string) ([]byte, error) {
	if responseType == RespTypeUncacheable {
		return body, nil
	}

	if contentType == api.HeaderAcceptJSON {
		switch responseType {
		case RespTypeSearch:
			var resp tempopb.SearchResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return json.Marshal(&resp)

		case RespTypeTags:
			var resp tempopb.SearchTagsResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return json.Marshal(&resp)

		case RespTypeTagValues:
			var resp tempopb.SearchTagValuesResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return json.Marshal(&resp)

		case RespTypeQueryRange:
			var resp tempopb.QueryRangeResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return json.Marshal(&resp)

		case RespTypeQueryInstant:
			var resp tempopb.QueryInstantResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return json.Marshal(&resp)
		}
	} else if contentType == api.HeaderAcceptProtobuf {
		switch responseType {
		case RespTypeSearch:
			var resp tempopb.SearchResponse
			if err := proto.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return proto.Marshal(&resp)

		case RespTypeTags:
			var resp tempopb.SearchTagsResponse
			if err := proto.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return proto.Marshal(&resp)

		case RespTypeTagValues:
			var resp tempopb.SearchTagValuesResponse
			if err := proto.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return proto.Marshal(&resp)

		case RespTypeQueryRange:
			var resp tempopb.QueryRangeResponse
			if err := proto.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return proto.Marshal(&resp)

		case RespTypeQueryInstant:
			var resp tempopb.QueryInstantResponse
			if err := proto.Unmarshal(body, &resp); err != nil {
				return body, err
			}
			if resp.Metrics != nil {
				resp.Metrics.Reset()
			}
			return proto.Marshal(&resp)
		}
	}

	return body, nil
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
func (c *frontendCache) fetchBytes(ctx context.Context, key string) []byte {
	if c.c == nil {
		return nil
	}

	if len(key) == 0 {
		return nil
	}

	buf, found := c.c.FetchKey(ctx, key)
	if !found {
		return nil
	}

	return buf
}

func determineResponseType(req Request) CacheableRespType {
	if req == nil || req.HTTPRequest() == nil {
		return RespTypeUncacheable
	}

	path := req.HTTPRequest().URL.Path

	if strings.Contains(path, api.PathSearch) {
		return RespTypeSearch
	}

	if strings.Contains(path, api.PathSearchTags) || strings.Contains(path, api.PathSearchTagsV2) {
		return RespTypeTags
	}

	if regexp.MustCompile(`^/api(/v2)?/search/tag/[^/]+/values`).MatchString(path) {
		return RespTypeTagValues
	}

	if strings.Contains(path, api.PathMetricsQueryRange) {
		return RespTypeQueryRange
	}

	if strings.Contains(path, api.PathMetricsQueryInstant) {
		return RespTypeQueryInstant
	}

	return RespTypeUncacheable
}
