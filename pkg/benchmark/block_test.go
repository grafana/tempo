package benchmark

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/internal/benchtest"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

func TestLoadLocalBlock(t *testing.T) {
	want, _, bucket := benchtest.Block(t, 20)

	path := filepath.Join(bucket, want.TenantID, want.BlockID.String())
	got, r, err := LoadLocalBlock(context.Background(), path)
	require.NoError(t, err)
	require.NotNil(t, r)

	// meta.json is JSON, so times lose their monotonic reading; compare fields.
	require.Equal(t, want.BlockID, got.BlockID)
	require.Equal(t, want.TenantID, got.TenantID)
	require.Equal(t, want.Version, got.Version)
	require.Equal(t, want.TotalObjects, got.TotalObjects)
	require.Equal(t, want.TotalRecords, got.TotalRecords)
	require.Equal(t, want.Size_, got.Size_)
	require.True(t, want.StartTime.Equal(got.StartTime))
	require.True(t, want.EndTime.Equal(got.EndTime))

	// A trailing separator must address the same block.
	got, _, err = LoadLocalBlock(context.Background(), path+string(filepath.Separator))
	require.NoError(t, err)
	require.Equal(t, want.BlockID, got.BlockID)
}

func TestLoadLocalBlockRejectsBadPaths(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		path    string
		wantErr string
	}{
		{"block dir is not a UUID", filepath.Join(t.TempDir(), "tenant", "not-a-uuid"), "not a block directory"},
		{"no tenant above the block", "/00000000-0000-0000-0000-000000000000", "no tenant directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := LoadLocalBlock(ctx, tc.path)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// recordingReader wraps a raw reader, recording the objects a warmup read
// and failing the read of the one it is told to.
type recordingReader struct {
	backend.RawReader
	fail string
	// reads records the objects a warmup read.
	reads []string
}

func (r *recordingReader) Read(ctx context.Context, name string, keyPath backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
	if r.fail == name {
		return nil, 0, errors.New("backend unavailable")
	}
	r.reads = append(r.reads, name)
	return r.RawReader.Read(ctx, name, keyPath, cacheInfo)
}

// The warmup has to stream every object a case can read, so no case's
// measurement pays a cold read, and a failing object has to be named.
func TestWarmBlock(t *testing.T) {
	ctx := context.Background()

	meta, _, bucket := benchtest.Block(t, 50)
	_, raw, err := openLocalBlock(ctx, filepath.Join(bucket, meta.TenantID, meta.BlockID.String()))
	require.NoError(t, err)

	t.Run("reads every object", func(t *testing.T) {
		rec := &recordingReader{RawReader: raw}
		require.NoError(t, warmBlock(ctx, rec, meta))

		// data.parquet is what every supported encoding names its data object.
		want := []string{"data.parquet", common.NameIndex}
		for shard := range int(meta.BloomShardCount) {
			want = append(want, common.BloomName(shard))
		}
		require.Equal(t, want, rec.reads)
	})

	t.Run("names a failing object", func(t *testing.T) {
		rec := &recordingReader{RawReader: raw, fail: common.NameIndex}
		require.ErrorContains(t, warmBlock(ctx, rec, meta), common.NameIndex)
	})

	t.Run("rejects an unsupported version", func(t *testing.T) {
		meta := meta
		meta.Version = "v0"
		require.ErrorContains(t, warmBlock(ctx, raw, meta), "v0")
	})
}
