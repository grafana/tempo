package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryRangeHandlerSucceeds(t *testing.T) {
	resp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  1,
		},
		Series: []*tempopb.TimeSeries{
			{
				Labels: []v1.KeyValue{
					{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1200_000,
						Value:       2,
					},
					{
						TimestampMs: 1100_000,
						Value:       1,
					},
				},
			},
		},
	}

	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return resp
		},
	}, nil, nil, nil, func(c *Config, _ *overrides.Config) {
		c.Metrics.Sharder.Interval = time.Hour
	})

	tenant := "foo"

	httpReq := httptest.NewRequest("GET", api.PathMetricsQueryRange, nil)
	httpReq = api.BuildQueryRangeRequest(httpReq, &tempopb.QueryRangeRequest{
		Query: "{} | rate()",
		Start: uint64(1100 * time.Second),
		End:   uint64(1300 * time.Second),
		Step:  uint64(100 * time.Second),
	}, "")

	ctx := user.InjectOrgID(httpReq.Context(), tenant)
	httpReq = httpReq.WithContext(ctx)

	httpResp := httptest.NewRecorder()

	f.MetricsQueryRangeHandler.ServeHTTP(httpResp, httpReq)

	require.Equal(t, 200, httpResp.Code)

	expectedResp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   4, // 2 blocks, each with 2 row groups that take 1 job
			InspectedTraces: 4,
			InspectedBytes:  4,
			TotalJobs:       4,
			TotalBlocks:     2,
			TotalBlockBytes: 419430400,
		},
		Series: []*tempopb.TimeSeries{
			{
				Labels: []v1.KeyValue{
					{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1100_000,
						Value:       4,
					},
					{
						TimestampMs: 1200_000,
						Value:       8,
					},
					{
						TimestampMs: 1300_000,
						Value:       0,
					},
				},
			},
		},
	}

	actualResp := &tempopb.QueryRangeResponse{}
	err := jsonpb.Unmarshal(httpResp.Body, actualResp)
	require.NoError(t, err)
	require.Equal(t, expectedResp, actualResp)
}

func TestQueryRangeAccessesCache(t *testing.T) {
	tenant := "foo"
	meta := &backend.BlockMeta{
		StartTime:         time.Unix(150, 0),
		EndTime:           time.Unix(160, 0),
		Size_:             defaultTargetBytesPerRequest,
		TotalRecords:      1,
		BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000123"),
		ReplicationFactor: 1,
	}
	retResp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  1,
		},
		Series: []*tempopb.TimeSeries{
			{
				Labels: []v1.KeyValue{
					{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 12_000_000,
						Value:       2,
					},
					{
						TimestampMs: 11_000_000,
						Value:       1,
					},
				},
			},
		},
	}

	rdr := &mockReader{
		metas: []*backend.BlockMeta{meta},
	}

	// setup mock cache
	c := test.NewMockClient()
	p := test.NewMockProvider()
	err := p.AddCache(cache.RoleFrontendSearch, c)
	require.NoError(t, err)
	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return retResp
		},
	}, rdr, nil, p)

	// setup query
	step := 1000000000
	query := "{} | rate()"
	hash := hashForQueryRangeRequest(&tempopb.QueryRangeRequest{Query: query, Step: uint64(step)})
	startNS := 100 * time.Second
	endNS := 200 * time.Second
	cacheKey := queryRangeCacheKey(tenant, hash, time.Unix(0, int64(startNS)), time.Unix(0, int64(endNS)), meta, 0, 1)

	// confirm cache key coesn't exist
	_, bufs, _ := c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 0, len(bufs))

	// execute query
	path := fmt.Sprintf("/?start=%d&end=%d&q=%s", startNS, endNS, url.QueryEscape(query)) // encapsulates block above
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, tenant)
	req = req.WithContext(ctx)

	respWriter := httptest.NewRecorder()
	f.MetricsQueryRangeHandler.ServeHTTP(respWriter, req)

	resp := respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	actualResp := &tempopb.QueryRangeResponse{}
	bytesResp, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// confirm cache key exists and matches the response above
	_, bufs, _ = c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 1, len(bufs))

	actualCache := &tempopb.QueryRangeResponse{}
	err = jsonpb.Unmarshal(bytes.NewReader(bufs[0]), actualCache)
	require.NoError(t, err)
}

func TestQueryRangeHandlerV2MaxSeries(t *testing.T) {
	resp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  1,
		},
		Series: []*tempopb.TimeSeries{
			{
				Labels: []v1.KeyValue{
					{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1200_000,
						Value:       2,
					},
					{
						TimestampMs: 1100_000,
						Value:       1,
					},
				},
			},
			{
				Labels: []v1.KeyValue{
					{Key: "abc", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "xyz"}}},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1200_000,
						Value:       2,
					},
					{
						TimestampMs: 1100_000,
						Value:       1,
					},
				},
			},
		},
	}

	maxSeries := 1

	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return resp
		},
	}, nil, nil, nil, func(c *Config, _ *overrides.Config) {
		c.Metrics.Sharder.Interval = time.Hour
		c.Metrics.Sharder.MaxResponseSeries = maxSeries
	})

	tenant := "foo"

	httpReq := httptest.NewRequest("GET", api.PathMetricsQueryRange, nil)
	httpReq = api.BuildQueryRangeRequest(httpReq, &tempopb.QueryRangeRequest{
		Query: "{} | rate()",
		Start: uint64(1100 * time.Second),
		End:   uint64(1200 * time.Second),
		Step:  uint64(100 * time.Second),
	}, "")

	ctx := user.InjectOrgID(httpReq.Context(), tenant)
	httpReq = httpReq.WithContext(ctx)

	httpResp := httptest.NewRecorder()

	f.MetricsQueryRangeHandler.ServeHTTP(httpResp, httpReq)

	require.Equal(t, 200, httpResp.Code)

	actualResp := &tempopb.QueryRangeResponse{}
	err := jsonpb.Unmarshal(httpResp.Body, actualResp)
	require.NoError(t, err)
	require.Equal(t, maxSeries, len(actualResp.Series))
	require.Equal(t, tempopb.PartialStatus_PARTIAL, actualResp.Status)
}

func TestQueryRangeCachedMetrics(t *testing.T) {
	// set up backend
	tenant := "foo"
	meta := &backend.BlockMeta{
		StartTime:         time.Unix(150, 0),
		EndTime:           time.Unix(160, 0),
		Size_:             defaultTargetBytesPerRequest,
		TotalRecords:      1,
		BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000123"),
		ReplicationFactor: 1,
	}
	rdr := &mockReader{
		metas: []*backend.BlockMeta{meta},
	}

	// set up cache
	c := test.NewMockClient()
	p := test.NewMockProvider()
	err := p.AddCache(cache.RoleFrontendSearch, c)
	require.NoError(t, err)
	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return &tempopb.QueryRangeResponse{
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 2,
					InspectedBytes:  33,
				},
				Series: []*tempopb.TimeSeries{
					{
						Labels: []v1.KeyValue{
							{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
						},
						Samples: []tempopb.Sample{
							{
								TimestampMs: 11_000_000,
								Value:       1,
							},
						},
					},
				},
			}
		},
	}, rdr, nil, p, func(c *Config, _ *overrides.Config) {
		c.Metrics.Sharder.Interval = time.Hour
	})

	// set up query
	query := "{} | rate()"
	var step uint64 = 1000000000
	hash := hashForQueryRangeRequest(&tempopb.QueryRangeRequest{Query: query, Step: step})
	startNS := uint64(100 * time.Second)
	endNS := uint64(200 * time.Second)
	cacheKey := queryRangeCacheKey(tenant, hash, time.Unix(0, int64(startNS)), time.Unix(0, int64(endNS)), meta, 0, 1)

	// confirm cache key doesn't exist
	_, bufs, _ := c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 0, len(bufs))

	// execute query
	path := fmt.Sprintf("/?start=%d&end=%d&q=%s", startNS, endNS, url.QueryEscape(query))
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, tenant)
	req = req.WithContext(ctx)
	respWriter := httptest.NewRecorder()
	f.MetricsQueryRangeHandler.ServeHTTP(respWriter, req)
	resp := respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	// parse response
	actualResp := &tempopb.QueryRangeResponse{}
	bytesResp, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// verify metrics are collected
	require.Equal(t, uint64(33), actualResp.Metrics.InspectedBytes)
	require.Equal(t, uint32(2), actualResp.Metrics.InspectedTraces)
	require.Equal(t, uint32(1), actualResp.Metrics.CompletedJobs)
	require.Equal(t, uint32(1), actualResp.Metrics.TotalJobs)
	require.Equal(t, uint32(1), actualResp.Metrics.TotalBlocks)
	require.Equal(t, uint64(defaultTargetBytesPerRequest), actualResp.Metrics.TotalBlockBytes)

	// execute query again
	respWriter = httptest.NewRecorder()
	f.MetricsQueryRangeHandler.ServeHTTP(respWriter, req)
	resp = respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	// parse cached response
	actualResp = &tempopb.QueryRangeResponse{}
	bytesResp, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// verify metrics are 0 because the response was cached
	require.Equal(t, uint64(0), actualResp.Metrics.InspectedBytes)
	require.Equal(t, uint32(0), actualResp.Metrics.InspectedTraces)
	require.Equal(t, uint32(1), actualResp.Metrics.CompletedJobs)
	// these are metadata metrics and are not affected by caching
	require.Equal(t, uint32(1), actualResp.Metrics.TotalJobs)
	require.Equal(t, uint32(1), actualResp.Metrics.TotalBlocks)
	require.Equal(t, uint64(defaultTargetBytesPerRequest), actualResp.Metrics.TotalBlockBytes)
}

func TestQueryRangeHandlerWithEndCutoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		step := 10 * time.Second

		time.Sleep(123 * time.Millisecond) // to make default time for synctest imperfect
		now := time.Now()
		alignedNow := now.Truncate(step) // align back to step

		start := now.Add(-100 * time.Second).UnixNano()
		cutoff := 30 * time.Second

		tenant := "foo"
		meta := &backend.BlockMeta{
			StartTime:         now.Add(-100 * time.Second),
			EndTime:           now,
			Size_:             defaultTargetBytesPerRequest,
			TotalRecords:      1,
			BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000123"),
			ReplicationFactor: 1,
		}
		retResp := &tempopb.QueryRangeResponse{
			Metrics: &tempopb.SearchMetrics{
				InspectedTraces: 1,
				InspectedBytes:  1,
			},
			Series: []*tempopb.TimeSeries{
				{
					Labels: []v1.KeyValue{
						{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
					},
					Samples: []tempopb.Sample{
						{
							TimestampMs: 12_000_000,
							Value:       2,
						},
						{
							TimestampMs: 11_000_000,
							Value:       1,
						},
					},
				},
			},
		}

		rdr := &mockReader{
			metas: []*backend.BlockMeta{meta},
		}
		rt := &mockRoundTripperWithCapture{
			rt: mockRoundTripper{
				responseFn: func() proto.Message {
					return retResp
				},
			},
		}

		for _, tc := range []struct {
			name        string
			end         time.Time
			expectedEnd time.Time
		}{
			{
				name:        "now",
				end:         now,
				expectedEnd: alignedNow.Add(-cutoff),
			},
			{
				name:        "unaligned",
				end:         now.Add(-100 * time.Millisecond),
				expectedEnd: alignedNow.Add(-cutoff),
			},
			{
				name:        "before cutoff",
				end:         now.Add(-cutoff - 20*time.Second),
				expectedEnd: alignedNow.Add(-cutoff - 10*time.Second), // aligned to right
			},
			{
				name:        "before unaligned",
				end:         now.Add(-cutoff - 3*time.Second),
				expectedEnd: alignedNow.Add(-cutoff),
			},
		} {
			f := frontendWithSettings(t, rt, rdr, nil, nil, func(c *Config, _ *overrides.Config) {
				c.Metrics.Sharder.Interval = time.Hour
				c.QueryEndCutoff = cutoff
			})

			httpReq := httptest.NewRequest("GET", api.PathMetricsQueryRange, nil)
			httpReq = api.BuildQueryRangeRequest(httpReq, &tempopb.QueryRangeRequest{
				Query: "{} | rate()",
				Start: uint64(start),
				End:   uint64(tc.end.UnixNano()),
				Step:  uint64(step),
			}, "")

			ctx := user.InjectOrgID(httpReq.Context(), tenant)
			httpReq = httpReq.WithContext(ctx)

			httpResp := httptest.NewRecorder()

			f.MetricsQueryRangeHandler.ServeHTTP(httpResp, httpReq)

			resp := httpResp.Result()
			require.Equal(t, 200, resp.StatusCode)

			actualResp := &tempopb.QueryRangeResponse{}
			bytesResp, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
			require.NoError(t, err)

			require.NotNil(t, rt.req)
			actualEnd := time.Unix(0, int64(rt.req.End))
			assert.Equal(t, tc.expectedEnd, actualEnd, "[%s] actual end %s is not equal to expected end %s", tc.name, actualEnd, tc.expectedEnd)
		}
	})
}

// TestQueryRangeHandlerExemplarNormalization verifies that exemplars from shard responses are
// kept in the final response when the client omits req.Exemplars (sends 0). Before the fix,
// the frontend combiner was created with req.Exemplars=0 which caused it to immediately discard
// all exemplars returned by backend shards.
func TestQueryRangeHandlerExemplarNormalization(t *testing.T) {
	start := uint64(1100 * time.Second)
	end := uint64(1200 * time.Second)
	step := uint64(100 * time.Second)
	// Exemplar timestamp in the middle of the query range (milliseconds)
	exemplarTS := int64(1150 * 1000)

	mockResp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{InspectedTraces: 1, InspectedBytes: 1},
		Series: []*tempopb.TimeSeries{
			{
				Labels: []v1.KeyValue{
					{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
				},
				Samples: []tempopb.Sample{
					{TimestampMs: 1100_000, Value: 1},
					{TimestampMs: 1200_000, Value: 2},
				},
				Exemplars: []tempopb.Exemplar{
					{TimestampMs: exemplarTS, Value: 1.5},
				},
			},
		},
	}

	makeRequestWithQuery := func(query string, exemplars uint32) *http.Request {
		httpReq := httptest.NewRequest("GET", api.PathMetricsQueryRange, nil)
		httpReq = api.BuildQueryRangeRequest(httpReq, &tempopb.QueryRangeRequest{
			Query:     query,
			Start:     start,
			End:       end,
			Step:      step,
			Exemplars: exemplars,
		}, "")
		ctx := user.InjectOrgID(httpReq.Context(), "foo")
		return httpReq.WithContext(ctx)
	}
	makeRequest := func(exemplars uint32) *http.Request {
		return makeRequestWithQuery("{} | rate()", exemplars)
	}

	t.Run("client omits exemplars defaults to cfg.MaxExemplars", func(t *testing.T) {
		f := frontendWithSettings(t, &mockRoundTripper{
			responseFn: func() proto.Message { return mockResp },
		}, nil, nil, nil, func(c *Config, _ *overrides.Config) {
			c.Metrics.Sharder.Interval = time.Hour
			c.Metrics.Sharder.MaxExemplars = 10
		})

		httpResp := httptest.NewRecorder()
		f.MetricsQueryRangeHandler.ServeHTTP(httpResp, makeRequest(0))
		require.Equal(t, 200, httpResp.Code)

		actualResp := &tempopb.QueryRangeResponse{}
		require.NoError(t, jsonpb.Unmarshal(httpResp.Body, actualResp))

		var total int
		for _, s := range actualResp.Series {
			total += len(s.Exemplars)
		}
		assert.Greater(t, total, 0, "exemplars should be kept when client omits exemplars and MaxExemplars > 0")
	})

	t.Run("exemplars disabled when cfg.MaxExemplars is zero", func(t *testing.T) {
		f := frontendWithSettings(t, &mockRoundTripper{
			responseFn: func() proto.Message { return mockResp },
		}, nil, nil, nil, func(c *Config, _ *overrides.Config) {
			c.Metrics.Sharder.Interval = time.Hour
			c.Metrics.Sharder.MaxExemplars = 0
		})

		httpResp := httptest.NewRecorder()
		f.MetricsQueryRangeHandler.ServeHTTP(httpResp, makeRequest(0))
		require.Equal(t, 200, httpResp.Code)

		actualResp := &tempopb.QueryRangeResponse{}
		require.NoError(t, jsonpb.Unmarshal(httpResp.Body, actualResp))

		for _, s := range actualResp.Series {
			assert.Empty(t, s.Exemplars, "exemplars should be empty when MaxExemplars is disabled")
		}
	})

	t.Run("client requests exemplars but MaxExemplars is zero disables them", func(t *testing.T) {
		f := frontendWithSettings(t, &mockRoundTripper{
			responseFn: func() proto.Message { return mockResp },
		}, nil, nil, nil, func(c *Config, _ *overrides.Config) {
			c.Metrics.Sharder.Interval = time.Hour
			c.Metrics.Sharder.MaxExemplars = 0
		})

		httpResp := httptest.NewRecorder()
		f.MetricsQueryRangeHandler.ServeHTTP(httpResp, makeRequest(5)) // client requests exemplars
		require.Equal(t, 200, httpResp.Code)

		actualResp := &tempopb.QueryRangeResponse{}
		require.NoError(t, jsonpb.Unmarshal(httpResp.Body, actualResp))

		for _, s := range actualResp.Series {
			assert.Empty(t, s.Exemplars, "exemplars should be empty when MaxExemplars is zero even if client requests them")
		}
	})

	t.Run("client-specified exemplars capped to cfg.MaxExemplars", func(t *testing.T) {
		f := frontendWithSettings(t, &mockRoundTripper{
			responseFn: func() proto.Message { return mockResp },
		}, nil, nil, nil, func(c *Config, _ *overrides.Config) {
			c.Metrics.Sharder.Interval = time.Hour
			c.Metrics.Sharder.MaxExemplars = 10
		})

		httpResp := httptest.NewRecorder()
		f.MetricsQueryRangeHandler.ServeHTTP(httpResp, makeRequest(1000)) // request more than cfg cap
		require.Equal(t, 200, httpResp.Code)

		actualResp := &tempopb.QueryRangeResponse{}
		require.NoError(t, jsonpb.Unmarshal(httpResp.Body, actualResp))

		var total int
		for _, s := range actualResp.Series {
			total += len(s.Exemplars)
		}
		assert.Greater(t, total, 0, "exemplars should be kept when client requests more than cfg cap")
	})

	t.Run("invalid query returns 400", func(t *testing.T) {
		f := frontendWithSettings(t, &mockRoundTripper{
			responseFn: func() proto.Message { return mockResp },
		}, nil, nil, nil, func(c *Config, _ *overrides.Config) {
			c.Metrics.Sharder.Interval = time.Hour
			c.Metrics.Sharder.MaxExemplars = 10
		})

		httpResp := httptest.NewRecorder()
		f.MetricsQueryRangeHandler.ServeHTTP(httpResp, makeRequestWithQuery("this is not valid traceql", 0))
		require.Equal(t, http.StatusBadRequest, httpResp.Code)
	})
}

func TestNormalizeRequestExemplars(t *testing.T) {
	const maxExemplars = uint32(20)

	tcs := []struct {
		name          string
		query         string
		reqExemplars  uint32
		wantExemplars uint32
		wantErr       bool
	}{
		// no hint: 0 (unspecified) defaults to maxExemplars
		{
			name:          "no hint, unspecified defaults to max",
			query:         "{} | rate()",
			reqExemplars:  0,
			wantExemplars: maxExemplars,
		},
		// no hint: explicit value is preserved
		{
			name:          "no hint, explicit value preserved",
			query:         "{} | rate()",
			reqExemplars:  10,
			wantExemplars: 10,
		},
		// no hint: value above max is capped
		{
			name:          "no hint, above max capped",
			query:         "{} | rate()",
			reqExemplars:  maxExemplars + 5,
			wantExemplars: maxExemplars,
		},
		// hint int: overrides req.Exemplars
		{
			name:          "hint int overrides req value",
			query:         "{} | rate() with(exemplars=7)",
			reqExemplars:  10,
			wantExemplars: 7,
		},
		// hint int: overrides unspecified (0) req value without applying default
		{
			name:          "hint int=0 disables exemplars",
			query:         "{} | rate() with(exemplars=0)",
			reqExemplars:  0,
			wantExemplars: 0,
		},
		// hint int: capped to maxExemplars when above max
		{
			name:          "hint int above max is capped",
			query:         "{} | rate() with(exemplars=9999)",
			reqExemplars:  0,
			wantExemplars: maxExemplars,
		},
		// hint bool=false: disables exemplars regardless of req value
		{
			name:          "hint false disables exemplars",
			query:         "{} | rate() with(exemplars=false)",
			reqExemplars:  10,
			wantExemplars: 0,
		},
		// hint bool=true: no-op, falls through to req.Exemplars / default path
		{
			name:          "hint true is no-op, falls through to default",
			query:         "{} | rate() with(exemplars=true)",
			reqExemplars:  0,
			wantExemplars: maxExemplars,
		},
		// invalid query returns an error
		{
			name:    "invalid query returns error",
			query:   "this is not valid traceql",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := &tempopb.QueryRangeRequest{
				Query:     tc.query,
				Exemplars: tc.reqExemplars,
			}
			err := normalizeRequestExemplars(req, maxExemplars)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantExemplars, req.Exemplars)
		})
	}
}

func TestQueryRangeGRPCHandlerInvalidQueryReturnsError(t *testing.T) {
	f := frontendWithSettings(t, &mockRoundTripper{}, nil, nil, nil)

	srv := newMockStreamingServer[*tempopb.QueryRangeResponse]("foo", nil)
	err := f.MetricsQueryRange(&tempopb.QueryRangeRequest{
		Query: "this is not valid traceql",
		Start: 1,
		End:   2,
		Step:  1,
	}, srv)
	require.Error(t, err)
}

// mockRoundTripperWithCapture is a mitm helper that captures query range requests
type mockRoundTripperWithCapture struct {
	rt  mockRoundTripper
	req *tempopb.QueryRangeRequest
	mx  sync.Mutex
}

func (m *mockRoundTripperWithCapture) RoundTrip(req pipeline.Request) (*http.Response, error) {
	qrReq, err := api.ParseQueryRangeRequest(req.HTTPRequest())
	if err != nil {
		panic("wrong test setup")
	}
	m.mx.Lock()
	defer m.mx.Unlock()
	m.req = qrReq

	res, err := m.rt.RoundTrip(req)
	return res, err
}
