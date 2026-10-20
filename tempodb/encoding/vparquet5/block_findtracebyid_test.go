package vparquet5

import (
	"bytes"
	"context"
	"os"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/v3/pkg/io"
	"github.com/grafana/tempo/v3/pkg/model/trace"
	pq "github.com/grafana/tempo/v3/pkg/parquetquery"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

func TestBackendBlockFindTraceByID(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	var (
		ctx       = context.Background()
		numTraces = 16
		r         = backend.NewReader(rawR)
		w         = backend.NewWriter(rawW)
		cfg       = &common.BlockConfig{
			BloomFP:             0.01,
			BloomShardSizeBytes: 100 * 1024,
		}
	)

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
	meta.TotalObjects = int64(numTraces)
	meta.DedicatedColumns = test.MakeDedicatedColumns()

	// Test data - sorted by trace ID
	// Find trace by ID uses the column and page bounds,
	// which by default only stores 16 bytes, which is the first
	// half of the trace ID (which is stored as 32 hex text)
	// Therefore it is important that the test data here has
	// full-length trace IDs.
	// Additionally, we are populating and using the full set of
	// dedicated columns and attributes, with randomness, and deep
	// comparison that the trace is roundtripped correctly.
	var traces []struct {
		trace *tempopb.Trace
		id    common.ID
	}
	for i := 0; i < numTraces; i++ {
		var (
			id = test.ValidTraceID(nil)
			tr = test.MakeTrace(10, id)
		)
		test.AddRandomDedicatedAttributes(tr)
		traces = append(traces, struct {
			trace *tempopb.Trace
			id    common.ID
		}{
			trace: tr,
			id:    id,
		})
	}

	// Sort
	sort.Slice(traces, func(i, j int) bool {
		return bytes.Compare(traces[i].id, traces[j].id) == -1
	})

	s, newMeta := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)

	var (
		buffer       = &Trace{} // Buffer for reuse, which is important to test.
		rowGroupSize = 5        // Write test data, occasionally flushing (cutting new row group)
	)

	for _, tr := range traces {
		traceToParquet(newMeta, tr.id, tr.trace, buffer)
		require.NoError(t, s.Add(buffer, 0, 0))

		if s.CurrentBufferedObjects() >= rowGroupSize {
			_, err = s.Flush()
			require.NoError(t, err)
		}
	}
	_, err = s.Complete()
	require.NoError(t, err)

	b := newBackendBlock(newMeta, r)

	// Now find and verify all test traces
	for _, tr := range traces {
		gotProto, err := b.FindTraceByID(ctx, tr.id, common.DefaultSearchOptions())
		require.NoError(t, err)

		// Sort both actual and expected for comparison
		trace.SortTraceAndAttributes(gotProto.Trace)
		trace.SortTraceAndAttributes(tr.trace)
		require.Equal(t, tr.trace, gotProto.Trace)
	}
}

func TestBackendBlockFindTraceByID_TestData(t *testing.T) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	ctx := context.Background()

	blocks, _, _, err := r.Blocks(ctx, "single-tenant")
	require.NoError(t, err)
	assert.Len(t, blocks, 1)

	meta, err := r.BlockMeta(ctx, blocks[0], "single-tenant")
	require.NoError(t, err)

	b := newBackendBlock(meta, r)

	iter, err := b.rawIter(context.Background(), newRowPool(10))
	require.NoError(t, err)

	sch, _, _ := SchemaWithDynamicChanges(meta.DedicatedColumns)
	for {
		_, row, err := iter.Next(context.Background())
		require.NoError(t, err)

		if row == nil {
			break
		}

		tr := &Trace{}
		err = sch.Reconstruct(tr, row)
		require.NoError(t, err)

		protoTr, err := b.FindTraceByID(ctx, tr.TraceID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, protoTr)
	}
}

func TestBackendBlockTraceRoundtrip(t *testing.T) {
	testCases := []struct {
		name string
		tr   *Trace
		dc   backend.DedicatedColumns
	}{
		{
			name: "fullypopulated",
			tr:   fullyPopulatedTestTrace(test.ValidTraceID(nil)),
			dc:   test.MakeDedicatedColumns(),
		},
		{
			name: "mixed array/non-array",
			tr:   func() *Trace { tr, _ := mixedArrayTestTrace(); return tr }(),
			dc:   func() backend.DedicatedColumns { _, dc := mixedArrayTestTrace(); return dc }(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				ctx       = t.Context()
				block     = makeBackendBlockWithTracesWithDedicatedColumns(t, []*Trace{tc.tr}, tc.dc)
				wantProto = ParquetTraceToTempopbTrace(block.meta, tc.tr)
			)

			gotProto, err := block.FindTraceByID(ctx, tc.tr.TraceID, common.DefaultSearchOptions())
			require.NoError(t, err)
			require.Equal(t, wantProto, gotProto.Trace)
		})
	}
}

func BenchmarkFindTraceByID(b *testing.B) {
	ctx := context.TODO()
	traceID := []byte{}
	block := blockForBenchmarks(b)

	// index := genIndex(b, block)
	// writeBlockMeta(ctx, ww, meta, &common.ShardedBloomFilter{}, index)

	for _, tc := range []string{"0", EnvVarIndexEnabledValue} {
		b.Run(EnvVarIndexName+"="+tc, func(b *testing.B) {
			os.Setenv(EnvVarIndexName, tc)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				tr, err := block.FindTraceByID(ctx, traceID, common.DefaultSearchOptions())
				require.NoError(b, err)
				require.NotNil(b, tr)
			}
		})
	}
}

func TestFindRowGroupByStats(t *testing.T) {
	const tracesPerRowGroup = 5

	// IDs are 10, 20, ..., 150 in the first byte so there are gaps within and between row groups.
	id := func(b byte) common.ID {
		id := make(common.ID, 16)
		id[0] = b
		id[15] = 1
		return id
	}

	writeFile := func(t *testing.T, opts ...parquet.WriterOption) (*parquet.File, *backend.BlockMeta) {
		meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
		buf := &bytes.Buffer{}
		w := parquet.NewGenericWriter[*Trace](buf, opts...)
		for i := 1; i <= 15; i++ {
			tr, _ := traceToParquet(meta, id(byte(i*10)), test.MakeTrace(1, id(byte(i*10))), nil)
			_, err := w.Write([]*Trace{tr})
			require.NoError(t, err)
			if i%tracesPerRowGroup == 0 {
				require.NoError(t, w.Flush())
			}
		}
		require.NoError(t, w.Close())

		pf, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)
		require.Len(t, pf.RowGroups(), 3)
		return pf, meta
	}

	t.Run("statistics", func(t *testing.T) {
		pf, _ := writeFile(t)
		colIndex, _, _ := pq.GetColumnIndexByPath(pf, TraceIDColumnName)

		testCases := []struct {
			name     string
			id       byte
			rowGroup int
		}{
			{"before first", 5, -1},
			{"first", 10, 0},
			{"within first", 25, 0},
			{"last of first", 50, 0},
			{"between first and second", 55, -1},
			{"first of second", 60, 1},
			{"last", 150, 2},
			{"after last", 155, -1},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rowGroup, ok := findRowGroupByStats(pf, colIndex, id(tc.id))
				require.True(t, ok)
				require.Equal(t, tc.rowGroup, rowGroup)
			})
		}
	})

	t.Run("missing statistics", func(t *testing.T) {
		pf, meta := writeFile(t, parquet.SkipPageBounds(TraceIDColumnName))
		colIndex, _, _ := pq.GetColumnIndexByPath(pf, TraceIDColumnName)

		rr := NewBackendReaderAt(t.Context(), backend.NewReader(&backend.MockRawReader{}), DataFileName, meta)

		_, ok := findRowGroupByStats(pf, colIndex, id(60))
		require.False(t, ok)

		for i := 1; i <= 15; i++ {
			tr, err := findTraceByID(t.Context(), id(byte(i*10)), meta, pf, rr, -1)
			require.NoError(t, err)
			require.NotNil(t, tr)
		}
		tr, err := findTraceByID(t.Context(), id(55), meta, pf, rr, -1)
		require.NoError(t, err)
		require.Nil(t, tr)
	})
}

func TestPreloadOffsetIndexes(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)
	cr := &countingRawReader{RawReader: rawR}

	var (
		ctx       = context.Background()
		numTraces = 256
		r         = backend.NewReader(cr)
		w         = backend.NewWriter(rawW)
	)

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)

	// Write the parquet file with a small page buffer so that the TraceID
	// column (plain-encoded 16-byte values) spans multiple pages per column
	// chunk: 128 rows at ~64 rows per page = 2 pages.
	buf := &bytes.Buffer{}
	pw := parquet.NewGenericWriter[*Trace](buf, parquet.PageBufferSize(1024))
	for i := 0; i < numTraces; i++ {
		id := test.ValidTraceID(nil)
		tr, _ := traceToParquet(meta, id, test.MakeTrace(1, id), nil)
		_, err := pw.Write([]*Trace{tr})
		require.NoError(t, err)
		if i == numTraces/2-1 {
			require.NoError(t, pw.Flush()) // second row group
		}
	}
	require.NoError(t, pw.Close())
	require.NoError(t, w.Write(ctx, DataFileName, uuid.UUID(meta.BlockID), meta.TenantID, buf.Bytes(), nil))
	meta.Size_ = uint64(buf.Len())

	rr := NewBackendReaderAt(ctx, r, DataFileName, meta)
	pf, err := parquet.OpenFile(rr, int64(meta.Size_), parquet.SkipPageIndex(true))
	require.NoError(t, err)
	numRowGroups := len(pf.RowGroups())
	require.Equal(t, 2, numRowGroups)

	t.Run("lazy per-chunk reads without preload", func(t *testing.T) {
		chunks := pf.RowGroups()[0].ColumnChunks()
		before := cr.reads.Load()
		for _, cc := range chunks {
			_, err := cc.OffsetIndex()
			require.NoError(t, err)
		}
		require.Equal(t, int64(len(chunks)), cr.reads.Load()-before)

		// The test is only meaningful if column chunks span multiple pages.
		colIndex, _, _ := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
		oi, err := chunks[colIndex].OffsetIndex()
		require.NoError(t, err)
		require.GreaterOrEqual(t, oi.NumPages(), 2)
	})

	t.Run("single read with preload", func(t *testing.T) {
		for rowGroup := 1; rowGroup < numRowGroups; rowGroup++ {
			before := cr.reads.Load()
			require.NoError(t, preloadOffsetIndexes(rr, pf, rowGroup))
			require.Equal(t, int64(1), cr.reads.Load()-before, "preload should read the offset index region once")

			// All indexes are now loaded, so per-chunk OffsetIndex calls read nothing more.
			for _, cc := range pf.RowGroups()[rowGroup].ColumnChunks() {
				oi, err := cc.OffsetIndex()
				require.NoError(t, err)
				require.Positive(t, oi.NumPages())
			}
			require.Equal(t, int64(1), cr.reads.Load()-before, "per-chunk OffsetIndex calls should not read after preload")
		}
	})
}

type countingRawReader struct {
	backend.RawReader
	reads atomic.Int64
}

func (c *countingRawReader) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	c.reads.Add(1)
	return c.RawReader.ReadRange(ctx, name, keypath, offset, buffer, cacheInfo)
}
