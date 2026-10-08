package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet5"
)

// writeParquetRewriteBloom writes a bloom filter created with the given parameters
// as bloom-N files in dir, matching how block creation writes bloom shards.
func writeParquetRewriteBloom(t *testing.T, dir string, fp float64, shardSizeBytes int, n int) (bitsPerShard, k uint64, shardCount int) {
	t.Helper()

	b := common.NewBloom(fp, uint(shardSizeBytes), uint(n))
	shards, err := b.Marshal()
	require.NoError(t, err)
	for i, shard := range shards {
		require.NoError(t, os.WriteFile(filepath.Join(dir, common.BloomName(i)), shard, 0o600))
	}

	var header [16]byte
	copy(header[:], shards[0][:16])
	bitsPerShard = binary.BigEndian.Uint64(header[0:8])
	k = binary.BigEndian.Uint64(header[8:16])

	return bitsPerShard, k, b.GetShardCount()
}

func TestParquetIteratorsYieldAllTraces(t *testing.T) {
	ctx := context.Background()

	t.Run("vparquet3", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "data.parquet")
		f, err := os.Create(path)
		require.NoError(t, err)

		w := parquet.NewGenericWriter[*vparquet3.Trace](f)
		traces := make([]*vparquet3.Trace, 0, 3)
		for i := range 3 {
			traces = append(traces, &vparquet3.Trace{
				TraceID:      []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(i + 1)},
				TraceIDText:  fmt.Sprintf("%032d", i+1),
				RootSpanName: fmt.Sprintf("root-%d", i),
			})
		}
		_, err = w.Write(traces)
		require.NoError(t, err)
		require.NoError(t, w.Close())
		require.NoError(t, f.Close())

		in, err := os.Open(path)
		require.NoError(t, err)
		t.Cleanup(func() { _ = in.Close() })
		stat, err := in.Stat()
		require.NoError(t, err)
		pf, err := parquet.OpenFile(in, stat.Size())
		require.NoError(t, err)

		iter := &parquetIterator3{
			r: parquet.NewGenericReader[*vparquet3.Trace](pf),
			m: &backend.BlockMeta{},
		}
		t.Cleanup(iter.Close)

		yielded := 0
		for {
			_, tr, err := iter.Next(ctx)
			if tr != nil {
				yielded++
			}
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
		}
		require.Equal(t, 3, yielded)
	})

	t.Run("vparquet4", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "data.parquet")
		f, err := os.Create(path)
		require.NoError(t, err)

		w := parquet.NewGenericWriter[*vparquet4.Trace](f)
		traces := make([]*vparquet4.Trace, 0, 3)
		for i := range 3 {
			traces = append(traces, &vparquet4.Trace{
				TraceID:      []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(i + 1)},
				TraceIDText:  fmt.Sprintf("%032d", i+1),
				RootSpanName: fmt.Sprintf("root-%d", i),
			})
		}
		_, err = w.Write(traces)
		require.NoError(t, err)
		require.NoError(t, w.Close())
		require.NoError(t, f.Close())

		in, err := os.Open(path)
		require.NoError(t, err)
		t.Cleanup(func() { _ = in.Close() })
		stat, err := in.Stat()
		require.NoError(t, err)
		pf, err := parquet.OpenFile(in, stat.Size())
		require.NoError(t, err)

		iter := &parquetIterator4{
			r: parquet.NewGenericReader[*vparquet4.Trace](pf),
			m: &backend.BlockMeta{},
		}
		t.Cleanup(iter.Close)

		yielded := 0
		for {
			_, tr, err := iter.Next(ctx)
			if tr != nil {
				yielded++
			}
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
		}
		require.Equal(t, 3, yielded)
	})
}

func TestInheritedBloomParams(t *testing.T) {
	// inheritedBloomParams must reconstruct parameters that reproduce the original
	// bloom: same shard size, same hash count k, and same shard count.
	tests := []struct {
		name           string
		fp             float64
		shardSizeBytes int
		n              int
	}{
		// common.NewBloom(.01, 100*1024, n) creates k=7 shards of 100KiB
		{"default-like bloom", 0.01, 100 * 1024, 1_000_000},
		{"small block", 0.01, 100 * 1024, 100},
		{"tiny shard", 0.01, 1024, 50},
		{"high fp", 0.99, 1024 * 1024, 162},
		{"multi shard", 0.01, 1024, 1_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			bitsPerShard, k, shardCount := writeParquetRewriteBloom(t, dir, tt.fp, tt.shardSizeBytes, tt.n)
			meta := &backend.BlockMeta{
				TotalObjects:    int64(tt.n),
				BloomShardCount: uint32(shardCount),
			}

			inheritedFP, shardSizeBytes, err := inheritedBloomParams(dir, meta)
			require.NoError(t, err)
			require.Equal(t, tt.shardSizeBytes, shardSizeBytes)
			require.Greater(t, inheritedFP, 0.0)
			require.Less(t, inheritedFP, 1.0)

			// the reconstructed rate must reproduce the original bloom
			rebuilt := common.NewBloom(inheritedFP, uint(shardSizeBytes), uint(tt.n))
			require.Equal(t, shardCount, rebuilt.GetShardCount())

			rebuiltShards, err := rebuilt.Marshal()
			require.NoError(t, err)
			var header [16]byte
			copy(header[:], rebuiltShards[0][:16])
			require.Equal(t, bitsPerShard, binary.BigEndian.Uint64(header[0:8]))
			require.Equal(t, k, binary.BigEndian.Uint64(header[8:16]))
		})
	}

	t.Run("missing bloom shard", func(t *testing.T) {
		_, _, err := inheritedBloomParams(t.TempDir(), &backend.BlockMeta{})
		require.Error(t, err)
	})

	t.Run("truncated header", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, common.BloomName(0)), make([]byte, 8), 0o600))

		_, _, err := inheritedBloomParams(dir, &backend.BlockMeta{})
		require.Error(t, err)
	})

	t.Run("invalid header", func(t *testing.T) {
		tests := []struct {
			name string
			m, k uint64
		}{
			{"zero bits", 0, 7},
			{"non-byte-aligned bits", 8193, 7},
			{"zero hash functions", 8192, 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := t.TempDir()
				var header bytes.Buffer
				require.NoError(t, binary.Write(&header, binary.BigEndian, tt.m))
				require.NoError(t, binary.Write(&header, binary.BigEndian, tt.k))
				require.NoError(t, os.WriteFile(filepath.Join(dir, common.BloomName(0)), header.Bytes(), 0o600))

				_, _, err := inheritedBloomParams(dir, &backend.BlockMeta{})
				require.Error(t, err)
			})
		}
	})
}

func TestParquetRewriteBlockConfig(t *testing.T) {
	const (
		fixtureShardSizeBytes = 10240
		fixtureObjects        = 1000
	)

	newFixture := func(t *testing.T) (string, *backend.BlockMeta, float64) {
		t.Helper()

		dir := t.TempDir()
		bitsPerShard, k, shardCount := writeParquetRewriteBloom(t, dir, 0.03, fixtureShardSizeBytes, fixtureObjects)
		meta := &backend.BlockMeta{
			TotalObjects:    fixtureObjects,
			BloomShardCount: uint32(shardCount),
		}
		// the rate blockConfig inherits for this fixture. TestInheritedBloomParams
		// pins that this rate reproduces the fixture's bloom.
		inheritedFP := reconstructBloomFP(fixtureObjects, bitsPerShard, uint64(shardCount), k)

		return dir, meta, inheritedFP
	}

	tests := []struct {
		name    string
		cmd     parquetRewrite
		want    common.BlockConfig
		wantErr bool
	}{
		{
			name: "row group size defaults, bloom inherited",
			cmd:  parquetRewrite{},
			want: common.BlockConfig{
				RowGroupSizeBytes:   defaultRowGroupSizeBytes,
				BloomFP:             0, // filled in below from the fixture
				BloomShardSizeBytes: fixtureShardSizeBytes,
			},
		},
		{
			name: "all parameters overridden",
			cmd: parquetRewrite{
				RowGroupSizeBytes:   555,
				BloomFP:             0.02,
				BloomShardSizeBytes: 999,
			},
			want: common.BlockConfig{
				RowGroupSizeBytes:   555,
				BloomFP:             0.02,
				BloomShardSizeBytes: 999,
			},
		},
		{
			name: "row group size overridden, bloom inherited",
			cmd:  parquetRewrite{RowGroupSizeBytes: 555},
			want: common.BlockConfig{
				RowGroupSizeBytes:   555,
				BloomFP:             0, // filled in below from the fixture
				BloomShardSizeBytes: fixtureShardSizeBytes,
			},
		},
		{
			name: "bloom fp overridden, row group size defaults",
			cmd:  parquetRewrite{BloomFP: 0.02},
			want: common.BlockConfig{
				RowGroupSizeBytes:   defaultRowGroupSizeBytes,
				BloomFP:             0.02,
				BloomShardSizeBytes: fixtureShardSizeBytes,
			},
		},
		{
			name:    "negative row group size",
			cmd:     parquetRewrite{RowGroupSizeBytes: -1},
			wantErr: true,
		},
		{
			name:    "bloom fp out of range",
			cmd:     parquetRewrite{BloomFP: 3},
			wantErr: true,
		},
		{
			name:    "negative bloom shard size",
			cmd:     parquetRewrite{BloomShardSizeBytes: -1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, meta, inheritedFP := newFixture(t)
			tt.cmd.In = dir
			tt.want.Version = vparquet5.VersionString
			if tt.want.BloomFP == 0 && tt.cmd.BloomFP == 0 {
				tt.want.BloomFP = inheritedFP
			}

			got, err := tt.cmd.blockConfig(vparquet5.VersionString, meta)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, &tt.want, got)
		})
	}

	t.Run("missing bloom shard cannot be inherited", func(t *testing.T) {
		dir, meta, _ := newFixture(t)
		require.NoError(t, os.Remove(filepath.Join(dir, common.BloomName(0))))

		cmd := parquetRewrite{In: dir}
		_, err := cmd.blockConfig(vparquet5.VersionString, meta)
		require.Error(t, err)
	})
}
