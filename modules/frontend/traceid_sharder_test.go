package frontend

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/blockboundary"
)

func TestBuildShardedRequests(t *testing.T) {
	queryShards := 2

	sharder := &asyncTraceSharder{
		cfg: &TraceByIDConfig{
			QueryShards: queryShards,
		},
		blockBoundaries: blockboundary.CreateBlockBoundaries(queryShards - 1),
	}

	ctx := user.InjectOrgID(context.Background(), "blerg")
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	shardedReqs, err := sharder.buildShardedRequests(ctx, req)
	require.NoError(t, err)
	require.Len(t, shardedReqs, queryShards)

	require.Equal(t, "/querier?mode=ingesters", shardedReqs[0].RequestURI)
	urisEqual(t, []string{"/querier?blockEnd=ffffffffffffffffffffffffffffffff&blockStart=00000000000000000000000000000000&mode=blocks"}, []string{shardedReqs[1].RequestURI})
}
