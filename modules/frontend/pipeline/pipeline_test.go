package pipeline

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
	"github.com/stretchr/testify/require"
)

func TestHTTPRequestCloneDoesNotCopyCost(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/search", nil)
	require.NoError(t, err)

	req := NewHTTPRequest(httpReq)
	req.SetWeight(2)
	req.SetCacheKey("key")
	req.SetResponseData(5)
	req.SetCost(jobcost.Cost{Kind: jobcost.KindBlock, Base: 100, Estimate: 10})

	clone := req.CloneFromHTTPRequest(httpReq)

	require.Equal(t, jobcost.Cost{}, clone.Cost())
	require.Equal(t, 2, clone.Weight())
	require.Equal(t, "key", clone.CacheKey())
	require.Equal(t, 5, clone.ResponseData())

	// the clone does not touch the cost of the original
	clone.SetCost(jobcost.Cost{Kind: jobcost.KindLiveStore, Estimate: 1})
	require.Equal(t, jobcost.Cost{Kind: jobcost.KindBlock, Base: 100, Estimate: 10}, req.Cost())
}

func TestPipelineBridgeEchoesRequestDataAndCost(t *testing.T) {
	tcs := []struct {
		name         string
		responseData any
		cost         jobcost.Cost
	}{
		{name: "nothing"},
		{name: "cost only", cost: jobcost.Cost{Kind: jobcost.KindBlock, Base: 100, Estimate: 10}},
		{name: "response data only", responseData: 3},
		{
			name:         "response data and cost",
			responseData: 3,
			cost:         jobcost.Cost{Kind: jobcost.KindLiveStore, Estimate: 1},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			bridge := &pipelineBridge{
				next: RoundTripperFunc(func(Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK}, nil
				}),
			}

			httpReq, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/search", nil)
			require.NoError(t, err)
			req := NewHTTPRequest(httpReq)
			req.SetResponseData(tc.responseData)
			req.SetCost(tc.cost)

			resps, err := bridge.RoundTrip(req)
			require.NoError(t, err)

			resp, done, err := resps.Next(context.Background())
			require.NoError(t, err)
			require.True(t, done)
			require.Equal(t, tc.responseData, resp.RequestData())

			withCost, ok := resp.(interface{ Cost() jobcost.Cost })
			require.True(t, ok)
			require.Equal(t, tc.cost, withCost.Cost())
		})
	}
}

func TestPipelineBridgeError(t *testing.T) {
	bridge := &pipelineBridge{
		next: RoundTripperFunc(func(Request) (*http.Response, error) {
			return nil, errors.New("boom")
		}),
	}

	httpReq, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/search", nil)
	require.NoError(t, err)

	resps, err := bridge.RoundTrip(NewHTTPRequest(httpReq))
	require.EqualError(t, err, "boom")
	require.Nil(t, resps)
}
