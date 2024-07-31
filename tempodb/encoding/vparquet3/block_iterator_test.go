package vparquet3

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v2/tempodb/backend"
	"github.com/grafana/tempo/v2/tempodb/backend/local"
)

func TestRawIteratorReadsAllRows(t *testing.T) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	ctx := context.Background()

	blocks, _, err := r.Blocks(ctx, "single-tenant")
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	meta, err := r.BlockMeta(ctx, blocks[0], "single-tenant")
	require.NoError(t, err)

	b := newBackendBlock(meta, r)

	iter, err := b.rawIter(context.Background(), newRowPool(10))
	require.NoError(t, err)
	defer iter.Close()

	actualCount := 0
	for {
		_, tr, err := iter.Next(context.Background())
		if tr == nil {
			break
		}
		actualCount++
		require.NoError(t, err)
	}

	require.Equal(t, meta.TotalObjects, actualCount)
}
