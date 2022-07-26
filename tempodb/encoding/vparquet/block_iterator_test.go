package vparquet

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

func TestIteratorReadsAllRows(t *testing.T) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	ctx := context.Background()

	blocks, err := r.Blocks(ctx, "single-tenant")
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	meta, err := r.BlockMeta(ctx, blocks[0], "single-tenant")
	require.NoError(t, err)

	b := newBackendBlock(meta, r)

	iter, err := b.Iterator(context.Background(), &sync.Pool{})
	require.NoError(t, err)
	defer iter.Close()

	actualCount := 0
	for {
		tr, err := iter.Next(context.Background())
		if tr == nil {
			break
		}
		actualCount++
		require.NoError(t, err)
	}

	require.Equal(t, meta.TotalObjects, actualCount)
}

/*func BenchmarkIterator(b *testing.B) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(b, err)

	r := backend.NewReader(rawR)
	ctx := context.Background()

	blocks, err := r.Blocks(ctx, "single-tenant")
	require.NoError(b, err)
	require.Len(b, blocks, 1)

	meta, err := r.BlockMeta(ctx, blocks[0], "single-tenant")
	require.NoError(b, err)

	bl := newBackendBlock(meta, r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		iter, _ := bl.Iterator(ctx)

		for {
			tr, _ := iter.Next(ctx)
			if tr == nil {
				break
			}
			tracePool.Put(tr)
		}

		iter.Close()
	}
}*/
