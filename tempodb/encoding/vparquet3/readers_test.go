package vparquet3

import (
	"context"
	"io"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

var tenantID = "single-tenant"

type dummyReader struct {
	r           io.ReaderAt
	footer      bool
	columnIndex bool
	offsetIndex bool
}

func (d *dummyReader) ReadAt(p []byte, off int64) (int, error) { return d.r.ReadAt(p, off) }

func (d *dummyReader) SetFooterSection(_, _ int64)      { d.footer = true }
func (d *dummyReader) SetColumnIndexSection(_, _ int64) { d.columnIndex = true }
func (d *dummyReader) SetOffsetIndexSection(_, _ int64) { d.offsetIndex = true }

// TestParquetGoSetsMetadataSections tests if the special metadata sections are set correctly for caching.
// It is the best way right now to ensure that the interface used by the underlying parquet-go library does not drift.
// If this test starts failing at some point, we should update the interface used by `parquetOptimizedReaderAt` to match
// the specification in parquet-go
func TestParquetGoSetsMetadataSections(t *testing.T) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	ctx := context.Background()

	blocks, _, err := r.Blocks(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	meta, err := r.BlockMeta(ctx, blocks[0], tenantID)
	require.NoError(t, err)

	br := NewBackendReaderAt(ctx, r, DataFileName, meta)
	dr := &dummyReader{r: br}
	_, err = parquet.OpenFile(dr, int64(meta.Size))
	require.NoError(t, err)

	require.True(t, dr.footer)
	require.True(t, dr.columnIndex)
	require.True(t, dr.offsetIndex)
}

func TestCachingReaderShortcircuitsFooterHeader(t *testing.T) {
	rr := &recordingReaderAt{}
	pr := newCachedReaderAt(rr, 1000, 1000, 100)

	expectedReads := []read{}

	// magic number doesn't pass through
	_, err := pr.ReadAt(make([]byte, 4), 0)
	require.NoError(t, err)

	// footer size doesn't pass through
	_, err = pr.ReadAt(make([]byte, 8), 992)
	require.NoError(t, err)

	// other calls pass through
	_, err = pr.ReadAt(make([]byte, 13), 25)
	require.NoError(t, err)
	expectedReads = append(expectedReads, read{13, 25, cache.RoleParquetPage})

	_, err = pr.ReadAt(make([]byte, 97), 118)
	require.NoError(t, err)
	expectedReads = append(expectedReads, read{97, 118, cache.RoleParquetPage})

	_, err = pr.ReadAt(make([]byte, 59), 421)
	require.NoError(t, err)
	expectedReads = append(expectedReads, read{59, 421, cache.RoleParquetPage})

	require.Equal(t, expectedReads, rr.reads)
}

func TestCachingReaderAt(t *testing.T) {
	rr := &recordingReaderAt{}
	cr := newCachedReaderAt(rr, 1000, 100000, 10)

	expectedReads := []read{}

	// specially cached sections
	cr.SetColumnIndexSection(1, 34)
	_, err := cr.ReadAt(make([]byte, 34), 1)
	expectedReads = append(expectedReads, read{34, 1, cache.RoleParquetColumnIdx})
	require.NoError(t, err)

	cr.SetFooterSection(14, 20)
	_, err = cr.ReadAt(make([]byte, 20), 14)
	expectedReads = append(expectedReads, read{20, 14, cache.RoleParquetFooter})
	require.NoError(t, err)

	cr.SetOffsetIndexSection(13, 12)
	_, err = cr.ReadAt(make([]byte, 12), 13)
	expectedReads = append(expectedReads, read{12, 13, cache.RoleParquetOffsetIdx})
	require.NoError(t, err)

	// everything else is a parquet page
	_, err = cr.ReadAt(make([]byte, 13), 25)
	require.NoError(t, err)
	expectedReads = append(expectedReads, read{13, 25, cache.RoleParquetPage})

	_, err = cr.ReadAt(make([]byte, 97), 118)
	require.NoError(t, err)
	expectedReads = append(expectedReads, read{97, 118, cache.RoleParquetPage})

	// unless it's larger than the page size
	_, err = cr.ReadAt(make([]byte, 1001), 421)
	require.NoError(t, err)
	expectedReads = append(expectedReads, read{1001, 421, cache.RoleNone})

	require.Equal(t, expectedReads, rr.reads)
}

type read struct {
	len  int
	off  int64
	role cache.Role
}
type recordingReaderAt struct {
	reads []read
}

func (r *recordingReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	r.reads = append(r.reads, read{len(p), off, ""})

	return len(p), nil
}

func (r *recordingReaderAt) ReadAtWithCache(p []byte, off int64, role cache.Role) (n int, err error) {
	r.reads = append(r.reads, read{len(p), off, role})

	return len(p), nil
}
