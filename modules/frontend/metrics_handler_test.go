package frontend

import (
	"net/http/httptest"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestQueryRangeHandler(t *testing.T) {
	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return &tempopb.QueryRangeResponse{
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBytes:  1,
				},
			}
		},
	}, nil, nil, nil)
	tenant := "foo"

	httpReq := httptest.NewRequest("GET", api.PathMetricsQueryRange, nil)
	httpReq = api.BuildQueryRangeRequest(httpReq, &tempopb.QueryRangeRequest{})

	ctx := user.InjectOrgID(httpReq.Context(), tenant)
	httpReq = httpReq.WithContext(ctx)

	httpResp := httptest.NewRecorder()

	f.QueryRangeHandler.ServeHTTP(httpResp, httpReq)

	require.Equal(t, 200, httpResp.Code)

	actualResp := &tempopb.QueryRangeResponse{}
	err := jsonpb.Unmarshal(httpResp.Body, actualResp)
	require.NoError(t, err)
	require.Equal(t, &tempopb.QueryRangeResponse{}, actualResp)
}
