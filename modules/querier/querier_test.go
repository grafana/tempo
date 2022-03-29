package querier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/atomic"
	"github.com/weaveworks/common/user"
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

		o, err := overrides.NewOverrides(overrides.Limits{})
		require.NoError(t, err)

		q, err := New(tc.cfg, client.Config{}, nil, nil, o)
		require.NoError(t, err)

		for i := 0; i < tc.queriesToExecute; i++ {
			// ignore error purposefully here. all queries will error, but we don't care
			// numExternalRequests will tell us what we need to know
			_, _ = q.SearchBlock(ctx, &tempopb.SearchBlockRequest{})
		}

		require.Equal(t, tc.externalExpected, numExternalRequests.Load())
	}
}
