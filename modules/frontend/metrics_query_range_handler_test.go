package frontend

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
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
				},
			},
		},
	}

	actualResp := &tempopb.QueryRangeResponse{}
	err := jsonpb.Unmarshal(httpResp.Body, actualResp)
	require.NoError(t, err)
	require.Equal(t, expectedResp, actualResp)
}
