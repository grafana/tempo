package frontend

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v2/pkg/api"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	v1 "github.com/grafana/tempo/v2/pkg/tempopb/common/v1"
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
	}, nil, nil, nil, func(c *Config) {
		c.Metrics.Sharder.Interval = time.Hour
	})
	tenant := "foo"

	httpReq := httptest.NewRequest("GET", api.PathMetricsQueryRange, nil)
	httpReq = api.BuildQueryRangeRequest(httpReq, &tempopb.QueryRangeRequest{
		Query: "{} | rate()",
		Start: uint64(1100 * time.Second),
		End:   uint64(1200 * time.Second),
		Step:  uint64(100 * time.Second),
	})

	ctx := user.InjectOrgID(httpReq.Context(), tenant)
	httpReq = httpReq.WithContext(ctx)

	httpResp := httptest.NewRecorder()

	f.MetricsQueryRangeHandler.ServeHTTP(httpResp, httpReq)

	require.Equal(t, 200, httpResp.Code)

	// for reasons I don't understand, this query turns into 408 jobs.
	expectedResp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   4,
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
				},
			},
		},
	}

	actualResp := &tempopb.QueryRangeResponse{}
	err := jsonpb.Unmarshal(httpResp.Body, actualResp)
	require.NoError(t, err)
	require.Equal(t, expectedResp, actualResp)
}

func TestQueryRangeHandlerRespectsSamplingRate(t *testing.T) {
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
	}, nil, nil, nil, func(c *Config) {
		c.Metrics.Sharder.Interval = time.Hour
	})
	tenant := "foo"

	httpReq := httptest.NewRequest("GET", api.PathMetricsQueryRange, nil)
	httpReq = api.BuildQueryRangeRequest(httpReq, &tempopb.QueryRangeRequest{
		Query: "{} | rate() with (sample=.2)",
		Start: uint64(1100 * time.Second),
		End:   uint64(1200 * time.Second),
		Step:  uint64(100 * time.Second),
	})

	ctx := user.InjectOrgID(httpReq.Context(), tenant)
	httpReq = httpReq.WithContext(ctx)

	httpResp := httptest.NewRecorder()

	f.MetricsQueryRangeHandler.ServeHTTP(httpResp, httpReq)

	require.Equal(t, 200, httpResp.Code)

	expectedResp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   1,
			InspectedTraces: 1,
			InspectedBytes:  1,
			TotalJobs:       1,
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
						Value:       5,
					},
					{
						TimestampMs: 1200_000,
						Value:       10,
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
