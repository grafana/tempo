package frontend

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/pkg/blockboundary"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/require"
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

	shardedReqs, err := sharder.buildShardedRequests(pipeline.NewHTTPRequest(req), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, shardedReqs, queryShards)

	require.Equal(t, "/querier?mode=ingesters", shardedReqs[0].HTTPRequest().RequestURI)
	urisEqual(t, []string{"/querier?blockEnd=ffffffffffffffffffffffffffffffff&blockStart=00000000000000000000000000000000&mode=blocks"}, []string{shardedReqs[1].HTTPRequest().RequestURI})
}

func TestBuildShardedRequestsWithExternal(t *testing.T) {
	queryShards := 4

	sharder := &asyncTraceSharder{
		cfg: &TraceByIDConfig{
			QueryShards:     queryShards,
			ExternalEnabled: true,
		},
		blockBoundaries: blockboundary.CreateBlockBoundaries(queryShards - 2),
	}

	ctx := user.InjectOrgID(context.Background(), "blerg")
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	shardedReqs, err := sharder.buildShardedRequests(pipeline.NewHTTPRequest(req), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, shardedReqs, queryShards)

	require.Equal(t, "/querier?mode=ingesters", shardedReqs[0].HTTPRequest().RequestURI)
	require.Equal(t, "/querier?mode=external", shardedReqs[1].HTTPRequest().RequestURI)

	// Verify block shard requests
	for i := 2; i < queryShards; i++ {
		require.Contains(t, shardedReqs[i].HTTPRequest().RequestURI, "mode=blocks")
		require.Contains(t, shardedReqs[i].HTTPRequest().RequestURI, "blockStart")
		require.Contains(t, shardedReqs[i].HTTPRequest().RequestURI, "blockEnd")
	}
}

// TestBuildShardedRequestsBlocksPerShard verifies that when blocks_per_shard is set the
// number of block shards is derived from the live blocklist rather than query_shards.
func TestBuildShardedRequestsBlocksPerShard(t *testing.T) {
	tests := []struct {
		name            string
		blocksPerShard  uint
		numBlocks       int
		wantTotalShards int // ingester + block shards
		wantBlockShards int
	}{
		{
			name:            "zero blocks – single block shard",
			blocksPerShard:  10,
			numBlocks:       0,
			wantTotalShards: 2, // 1 ingester + 1 block shard (floor of 0 rounds up to 1)
			wantBlockShards: 1,
		},
		{
			name:            "blocks divide evenly",
			blocksPerShard:  5,
			numBlocks:       20,
			wantTotalShards: 5, // 1 ingester + 4 block shards
			wantBlockShards: 4,
		},
		{
			name:            "blocks do not divide evenly – rounds up",
			blocksPerShard:  5,
			numBlocks:       21,
			wantTotalShards: 6, // 1 ingester + 5 block shards (ceil(21/5) = 5)
			wantBlockShards: 5,
		},
		{
			name:            "single block",
			blocksPerShard:  10,
			numBlocks:       1,
			wantTotalShards: 2, // 1 ingester + 1 block shard
			wantBlockShards: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build a mock blocklist for the tenant "test-tenant"
			metas := make([]*backend.BlockMeta, tc.numBlocks)
			for i := range metas {
				metas[i] = &backend.BlockMeta{}
			}

			sharder := &asyncTraceSharder{
				cfg: &TraceByIDConfig{
					// query_shards is intentionally set to a value that would produce a
					// different answer than blocks_per_shard so we can confirm the right
					// path is taken.
					QueryShards:    100,
					BlocksPerShard: tc.blocksPerShard,
				},
				reader: &mockReader{metas: metas},
				// blockBoundaries is pre-computed from QueryShards but must NOT be used
				// when BlocksPerShard > 0.
				blockBoundaries: blockboundary.CreateBlockBoundaries(99),
			}

			ctx := user.InjectOrgID(context.Background(), "test-tenant")
			req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

			shardedReqs, err := sharder.buildShardedRequests(pipeline.NewHTTPRequest(req), time.Time{}, time.Time{})
			require.NoError(t, err)
			require.Len(t, shardedReqs, tc.wantTotalShards, "total shards (ingester + block shards)")

			// First request is always the ingester shard.
			require.Equal(t, "/querier?mode=ingesters", shardedReqs[0].HTTPRequest().RequestURI)

			// Remaining requests are block shards.
			for i := 1; i < len(shardedReqs); i++ {
				uri := shardedReqs[i].HTTPRequest().RequestURI
				require.Contains(t, uri, "mode=blocks", "shard %d should be a block shard", i)
				require.Contains(t, uri, "blockStart", "shard %d should contain blockStart", i)
				require.Contains(t, uri, "blockEnd", "shard %d should contain blockEnd", i)
			}
		})
	}
}

// TestBlocksPerShardTimeRangeFiltering verifies that when blocks_per_shard > 0 and startTime/endTime
// are non-zero, only blocks that overlap the time range are counted when computing block shards.
func TestBlocksPerShardTimeRangeFiltering(t *testing.T) {
	inRange := &backend.BlockMeta{
		StartTime: time.Unix(1000, 0),
		EndTime:   time.Unix(2000, 0),
	}
	outOfRange := &backend.BlockMeta{
		StartTime: time.Unix(5000, 0),
		EndTime:   time.Unix(6000, 0),
	}
	partialOverlap := &backend.BlockMeta{
		StartTime: time.Unix(1500, 0),
		EndTime:   time.Unix(3000, 0),
	}

	tests := []struct {
		name            string
		blocks          []*backend.BlockMeta
		blocksPerShard  uint
		startTime       time.Time
		endTime         time.Time
		wantTotalShards int // ingester + block shards
	}{
		{
			name:            "no time range – all blocks counted",
			blocks:          []*backend.BlockMeta{inRange, outOfRange},
			blocksPerShard:  1,
			startTime:       time.Time{},
			endTime:         time.Time{},
			wantTotalShards: 3, // 1 ingester + 2 block shards
		},
		{
			name:            "time range filters out-of-range blocks",
			blocks:          []*backend.BlockMeta{inRange, outOfRange},
			blocksPerShard:  1,
			startTime:       time.Unix(1000, 0),
			endTime:         time.Unix(2000, 0),
			wantTotalShards: 2, // 1 ingester + 1 block shard (only inRange matches)
		},
		{
			name:            "partial overlap is included",
			blocks:          []*backend.BlockMeta{inRange, outOfRange, partialOverlap},
			blocksPerShard:  1,
			startTime:       time.Unix(1000, 0),
			endTime:         time.Unix(2000, 0),
			wantTotalShards: 3, // 1 ingester + 2 block shards (inRange + partialOverlap)
		},
		{
			name:            "time range with no matching blocks falls back to 1 block shard",
			blocks:          []*backend.BlockMeta{inRange},
			blocksPerShard:  1,
			startTime:       time.Unix(9000, 0),
			endTime:         time.Unix(10000, 0),
			wantTotalShards: 2, // 1 ingester + 1 block shard (minimum)
		},
		{
			name:            "blocks per shard groups filtered blocks",
			blocks:          []*backend.BlockMeta{inRange, outOfRange, partialOverlap},
			blocksPerShard:  2,
			startTime:       time.Unix(1000, 0),
			endTime:         time.Unix(2000, 0),
			wantTotalShards: 2, // 1 ingester + 1 block shard (ceil(2/2)=1)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sharder := &asyncTraceSharder{
				cfg: &TraceByIDConfig{
					QueryShards:    100,
					BlocksPerShard: tc.blocksPerShard,
				},
				reader:          &mockReader{metas: tc.blocks},
				blockBoundaries: blockboundary.CreateBlockBoundaries(99),
			}

			ctx := user.InjectOrgID(context.Background(), "test-tenant")
			req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

			shardedReqs, err := sharder.buildShardedRequests(pipeline.NewHTTPRequest(req), tc.startTime, tc.endTime)
			require.NoError(t, err)
			require.Len(t, shardedReqs, tc.wantTotalShards)

			require.Equal(t, "/querier?mode=ingesters", shardedReqs[0].HTTPRequest().RequestURI)
			for i := 1; i < len(shardedReqs); i++ {
				uri := shardedReqs[i].HTTPRequest().RequestURI
				require.Contains(t, uri, "mode=blocks", "shard %d should be a block shard", i)
			}
		})
	}
}

// TestBlocksPerShardRespectsMaxOutstanding verifies that the number of block shards is
// capped so that total jobs never exceed maxOutstandingPerTenant.
func TestBlocksPerShardRespectsMaxOutstanding(t *testing.T) {
	tests := []struct {
		name                  string
		numBlocks             int
		blocksPerShard        uint
		maxDynamicBlockShards int // 0 = uncapped
		externalEnabled       bool
		wantTotalShards       int
	}{
		{
			name:                  "cap not reached",
			numBlocks:             10,
			blocksPerShard:        1,
			maxDynamicBlockShards: 19, // maxOutstandingPerTenant=20 minus 1 ingester
			wantTotalShards:       11, // 1 ingester + 10 block shards
		},
		{
			name:                  "cap exactly reached",
			numBlocks:             10,
			blocksPerShard:        1,
			maxDynamicBlockShards: 10, // maxOutstandingPerTenant=11 minus 1 ingester
			wantTotalShards:       11, // 1 ingester + 10 block shards
		},
		{
			name:                  "cap exceeded – block shards trimmed",
			numBlocks:             100,
			blocksPerShard:        1,
			maxDynamicBlockShards: 10, // maxOutstandingPerTenant=11 minus 1 ingester
			wantTotalShards:       11, // 1 ingester + 10 block shards (capped)
		},
		{
			name:                  "external enabled reduces available block shards",
			numBlocks:             100,
			blocksPerShard:        1,
			externalEnabled:       true,
			maxDynamicBlockShards: 9,  // maxOutstandingPerTenant=11 minus 1 ingester minus 1 external
			wantTotalShards:       11, // 1 ingester + 1 external + 9 block shards (capped)
		},
		{
			name:                  "zero maxDynamicBlockShards disables cap",
			numBlocks:             100,
			blocksPerShard:        1,
			maxDynamicBlockShards: 0,
			wantTotalShards:       101, // 1 ingester + 100 block shards (uncapped)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metas := make([]*backend.BlockMeta, tc.numBlocks)
			for i := range metas {
				metas[i] = &backend.BlockMeta{}
			}

			sharder := &asyncTraceSharder{
				cfg: &TraceByIDConfig{
					QueryShards:     100,
					BlocksPerShard:  tc.blocksPerShard,
					ExternalEnabled: tc.externalEnabled,
				},
				reader:                &mockReader{metas: metas},
				blockBoundaries:       blockboundary.CreateBlockBoundaries(99),
				maxDynamicBlockShards: tc.maxDynamicBlockShards,
			}

			ctx := user.InjectOrgID(context.Background(), "test-tenant")
			req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

			shardedReqs, err := sharder.buildShardedRequests(pipeline.NewHTTPRequest(req), time.Time{}, time.Time{})
			require.NoError(t, err)
			require.Len(t, shardedReqs, tc.wantTotalShards)
		})
	}
}

// TestBlocksPerShardFallsBackToQueryShards verifies that when blocks_per_shard == 0
// the sharder behaves identically to the legacy query_shards path.
func TestBlocksPerShardFallsBackToQueryShards(t *testing.T) {
	queryShards := 3
	sharder := &asyncTraceSharder{
		cfg: &TraceByIDConfig{
			QueryShards:    queryShards,
			BlocksPerShard: 0, // disabled – should fall back to QueryShards
		},
		// No reader needed because BlocksPerShard == 0.
		blockBoundaries: blockboundary.CreateBlockBoundaries(queryShards - 1),
	}

	ctx := user.InjectOrgID(context.Background(), "blerg")
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	shardedReqs, err := sharder.buildShardedRequests(pipeline.NewHTTPRequest(req), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, shardedReqs, queryShards)

	require.Equal(t, "/querier?mode=ingesters", shardedReqs[0].HTTPRequest().RequestURI)
	for i := 1; i < queryShards; i++ {
		uri := shardedReqs[i].HTTPRequest().RequestURI
		require.Contains(t, uri, "mode=blocks")
	}
}
