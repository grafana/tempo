package vparquet

import (
	"context"
	"io"
	"testing"

	"github.com/segmentio/parquet-go"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

var (
	tenantID = "single-tenant"
)

type dummyReader struct {
	r           io.ReaderAt
	footer      bool
	columnIndex bool
	offsetIndex bool
}

func (d *dummyReader) ReadAt(p []byte, off int64) (int, error) { return d.r.ReadAt(p, off) }

func (d *dummyReader) SetFooterSection(_ int64, _ int64)      { d.footer = true }
func (d *dummyReader) SetColumnIndexSection(_ int64, _ int64) { d.columnIndex = true }
func (d *dummyReader) SetOffsetIndexSection(_ int64, _ int64) { d.offsetIndex = true }

// TestParquetGoSetsMetadataSections tests if the special metadata sections are set correctly for caching.
// It is the best way right now to ensure that the interface used by the underlying parquet-go library does not drift.
func TestParquetGoSetsMetadataSections(t *testing.T) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	ctx := context.Background()

	blocks, err := r.Blocks(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	meta, err := r.BlockMeta(ctx, blocks[0], tenantID)
	require.NoError(t, err)

	br := NewBackendReaderAt(ctx, r, DataFileName, meta.BlockID, tenantID)
	dr := &dummyReader{r: br}
	_, err = parquet.OpenFile(dr, int64(meta.Size))
	require.NoError(t, err)

	require.True(t, dr.footer)
	require.True(t, dr.columnIndex)
	require.True(t, dr.offsetIndex)
}
