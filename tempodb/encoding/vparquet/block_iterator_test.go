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

func TestIteratorReusesObjectsViaPool(t *testing.T) {
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

	// track number of objects created through pool.New()
	count := 0
	pool := sync.Pool{New: func() any { count++; return &Trace{} }}

	iter, err := b.Iterator(context.Background(), &pool)
	require.NoError(t, err)
	defer iter.Close()

	for {
		tr, err := iter.Next(context.Background())
		if tr == nil {
			break
		}
		require.NoError(t, err)

		// reuse objects
		pool.Put(tr)
	}

	// only the first object should be created via pool
	require.Equal(t, 1, count)
}
