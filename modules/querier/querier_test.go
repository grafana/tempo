package querier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uber-go/atomic"

	"github.com/grafana/dskit/user"
	generator_client "github.com/grafana/tempo/modules/generator/client"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier/external"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestQuerierUsesSearchExternalEndpoint(t *testing.T) {
	numExternalRequests := atomic.NewInt32(0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		numExternalRequests.Inc()
	}))
	defer srv.Close()

	tests := []struct {
		cfg              Config
		queriesToExecute int
		externalExpected int32
	}{
		// SearchExternalEndpoints is respected
		{
			cfg: Config{
				Search: SearchConfig{
					ExternalEndpoints: []string{srv.URL},
				},
			},
			queriesToExecute: 3,
			externalExpected: 3,
		},
		// No SearchExternalEndpoints causes the querier to service everything internally
		{
			cfg:              Config{},
			queriesToExecute: 3,
			externalExpected: 0,
		},
		{
			cfg: Config{
				Search: SearchConfig{
					ExternalBackend: "google_cloud_run",
					CloudRun: &external.CloudRunConfig{
						Endpoints: []string{srv.URL},
						NoAuth:    true,
					},
				},
			},
			queriesToExecute: 1,
			externalExpected: 1,
		},
		// SearchPreferSelf is respected. this test won't pass b/c SearchBlock fails instantly and so
		//  all 3 queries are executed locally and nothing is proxied to the external endpoint.
		//  we'd have to mock the storage.Store interface to get this to pass. it's a big interface.
		// {
		// 	cfg: Config{
		// 		SearchExternalEndpoints: []string{srv.URL},
		// 		SearchPreferSelf:        2,
		// 	},
		// 	queriesToExecute: 3,
		// 	externalExpected: 1,
		// },
	}

	ctx := user.InjectOrgID(context.Background(), "blerg")

	for _, tc := range tests {
		numExternalRequests.Store(0)

		o, err := overrides.NewOverrides(overrides.Config{})
		require.NoError(t, err)

		q, err := New(tc.cfg, ingester_client.Config{}, nil, generator_client.Config{}, nil, nil, o)
		require.NoError(t, err)

		for i := 0; i < tc.queriesToExecute; i++ {
			// ignore error purposefully here. all queries will error, but we don't care
			// numExternalRequests will tell us what we need to know
			_, _ = q.SearchBlock(ctx, &tempopb.SearchBlockRequest{})
		}

		require.Equal(t, tc.externalExpected, numExternalRequests.Load())
	}
}

func TestVirtualTagsDoesntHitBackend(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	q, err := New(Config{}, ingester_client.Config{}, nil, generator_client.Config{}, nil, nil, o)
	require.NoError(t, err)

	ctx := user.InjectOrgID(context.Background(), "blerg")

	// duration should return nothing
	resp, err := q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "duration",
	})
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchTagValuesV2Response{}, resp)

	// traceDuration should return nothing
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "traceDuration",
	})
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchTagValuesV2Response{}, resp)

	// status should return a static list
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "status",
	})
	require.NoError(t, err)
	sort.Slice(resp.TagValues, func(i, j int) bool { return resp.TagValues[i].Value < resp.TagValues[j].Value })
	require.Equal(t, &tempopb.SearchTagValuesV2Response{
		TagValues: []*tempopb.TagValue{
			{
				Type:  "keyword",
				Value: "error",
			},
			{
				Type:  "keyword",
				Value: "ok",
			},
			{
				Type:  "keyword",
				Value: "unset",
			},
		},
	}, resp)

	// kind should return a static list
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "kind",
	})
	require.NoError(t, err)
	sort.Slice(resp.TagValues, func(i, j int) bool { return resp.TagValues[i].Value < resp.TagValues[j].Value })
	require.Equal(t, &tempopb.SearchTagValuesV2Response{
		TagValues: []*tempopb.TagValue{
			{
				Type:  "keyword",
				Value: "client",
			},
			{
				Type:  "keyword",
				Value: "consumer",
			},
			{
				Type:  "keyword",
				Value: "internal",
			},
			{
				Type:  "keyword",
				Value: "producer",
			},
			{
				Type:  "keyword",
				Value: "server",
			},
			{
				Type:  "keyword",
				Value: "unspecified",
			},
		},
	}, resp)

	// this should error b/c it will attempt to hit the unconfigured backend
	_, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: ".foo",
	})
	require.Error(t, err)
}
