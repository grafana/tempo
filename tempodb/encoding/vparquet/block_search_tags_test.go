package vparquet

import (
	"context"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendBlockSearchTags(t *testing.T) {
	traces, attrs := makeTraces()
	block := makeBackendBlockWithTraces(t, traces)

	foundAttrs := map[string]struct{}{}

	cb := func(s string) {
		foundAttrs[s] = struct{}{}
	}

	ctx := context.Background()
	err := block.SearchTags(ctx, cb, common.DefaultSearchOptions())
	require.NoError(t, err)

	// test that all attrs are in found attrs
	for k := range attrs {
		_, ok := foundAttrs[k]
		require.True(t, ok)
	}
}

func TestBackendBlockSearchTagValues(t *testing.T) {
	traces, attrs := makeTraces()
	block := makeBackendBlockWithTraces(t, traces)

	ctx := context.Background()
	for tag, val := range attrs {
		wasCalled := false
		cb := func(s string) {
			wasCalled = true
			assert.Equal(t, val, s, tag)
		}

		err := block.SearchTagValues(ctx, tag, cb, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.True(t, wasCalled, tag)
	}
}

func BenchmarkBackendBlockSearchTags(b *testing.B) {
	ctx := context.TODO()
	tenantID := "1"
	blockID := uuid.MustParse("3685ee3d-cbbf-4f36-bf28-93447a19dea6")

	r, _, _, err := local.New(&local.Config{
		Path: path.Join("/Users/marty/src/tmp/"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	block := newBackendBlock(meta, rr)
	opts := common.DefaultSearchOptions()
	d := util.NewDistinctStringCollector(1_000_000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := block.SearchTags(ctx, d.Collect, opts)
		require.NoError(b, err)
	}
}

func BenchmarkBackendBlockSearchTagValues(b *testing.B) {
	testCases := []string{
		"foo",
		"http.url",
	}

	ctx := context.TODO()
	tenantID := "1"
	blockID := uuid.MustParse("3685ee3d-cbbf-4f36-bf28-93447a19dea6")

	r, _, _, err := local.New(&local.Config{
		Path: path.Join("/Users/marty/src/tmp/"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	block := newBackendBlock(meta, rr)
	opts := common.DefaultSearchOptions()

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			d := util.NewDistinctStringCollector(1_000_000)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := block.SearchTagValues(ctx, tc, d.Collect, opts)
				require.NoError(b, err)
			}
		})
	}
}
