package querier

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	livestore_client "github.com/grafana/tempo/v3/modules/livestore/client"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/modules/storage"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// optsStore records the search options the querier hands to storage.
type optsStore struct {
	storage.Store
	opts []common.SearchOptions
}

func (s *optsStore) SearchTags(_ context.Context, _ *backend.BlockMeta, _ *tempopb.SearchTagsBlockRequest, opts common.SearchOptions) (*tempopb.SearchTagsV2Response, error) {
	s.opts = append(s.opts, opts)
	return &tempopb.SearchTagsV2Response{}, nil
}

func (s *optsStore) FetchTagNames(_ context.Context, _ *backend.BlockMeta, _ traceql.FetchTagsRequest, _ traceql.FetchTagsCallback, _ common.MetricsCallback, opts common.SearchOptions) error {
	s.opts = append(s.opts, opts)
	return nil
}

// The frontend shards a tag-name search into page ranges. Each job's range has
// to reach storage, or every job scans the whole block.
func TestSearchTagsBlockPassesShardToStorage(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	for name, query := range map[string]string{
		"without TraceQL": "",
		"with TraceQL":    `{ span.foo = "bar" }`,
	} {
		t.Run(name, func(t *testing.T) {
			store := &optsStore{}
			q, err := New(Config{}, nil, livestore_client.Config{}, nil, false, store, o)
			require.NoError(t, err)

			ctx := user.InjectOrgID(context.Background(), "test")
			_, err = q.SearchTagsBlocksV2(ctx, &tempopb.SearchTagsBlockRequest{
				BlockID:       uuid.NewString(),
				StartPage:     3,
				PagesToSearch: 2,
				SearchReq:     &tempopb.SearchTagsRequest{Query: query, Scope: "span"},
			})
			require.NoError(t, err)

			require.NotEmpty(t, store.opts, "storage was not called")
			for _, opts := range store.opts {
				require.Equal(t, 3, opts.StartPage)
				require.Equal(t, 2, opts.TotalPages)
			}
		})
	}
}
