package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
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
				PromLabels: "foo",
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
				PromLabels: "foo",
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
		StartTime:         time.Unix(15, 0),
		EndTime:           time.Unix(16, 0),
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
				PromLabels: "foo",
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
	startNS := 10 * time.Second
	endNS := 20 * time.Second
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
				PromLabels: "foo",
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
				PromLabels: "abc",
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
		StartTime:         time.Unix(15, 0),
		EndTime:           time.Unix(16, 0),
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
						PromLabels: "foo",
						Labels: []v1.KeyValue{
							{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
						},
						Samples: []tempopb.Sample{
							{
								TimestampMs: 1100_000,
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
	startNS := uint64(10 * time.Second)
	endNS := uint64(20 * time.Second)
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
