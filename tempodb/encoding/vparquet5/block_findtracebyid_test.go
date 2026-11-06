package vparquet5

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/prometheus/client_golang/prometheus"
	prom_testutil "github.com/prometheus/client_golang/prometheus/testutil"
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
	cr := &countingRawReader{RawReader: rawR}

	var (
		ctx       = context.Background()
		numTraces = 16
		r         = backend.NewReader(cr)
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

	stages := []string{
		common.FindBlocksStageBloomChecked,
		common.FindBlocksStageBloomPass,
		common.FindBlocksStageIndexPass,
		common.FindBlocksStageRead,
	}
	counters := make([]prometheus.Counter, len(stages))
	for i, stage := range stages {
		counters[i] = common.MetricFindBlocks.WithLabelValues(newMeta.TenantID, stage)
	}
	before := make([]float64, len(counters))
	for i, c := range counters {
		before[i] = prom_testutil.ToFloat64(c)
	}

	// Now find and verify all test traces
	for _, tr := range traces {
		gotProto, err := b.FindTraceByID(ctx, tr.id, common.DefaultSearchOptions())
		require.NoError(t, err)

		// Sort both actual and expected for comparison
		trace.SortTraceAndAttributes(gotProto.Trace)
		trace.SortTraceAndAttributes(tr.trace)
		require.Equal(t, tr.trace, gotProto.Trace)
	}

	for i, c := range counters {
		require.Equal(t, float64(numTraces), prom_testutil.ToFloat64(c)-before[i])
	}

	// A smaller TraceByIDReadBufferSize replaces ReadBufferSize, so the lookup takes more reads.
	readsWith := func(bufSize int) int64 {
		opts := common.DefaultSearchOptions()
		opts.TraceByIDReadBufferSize = bufSize
		before := cr.reads.Load()
		_, err := b.FindTraceByID(ctx, traces[0].id, opts)
		require.NoError(t, err)
		return cr.reads.Load() - before
	}
	require.Greater(t, readsWith(64), readsWith(0))
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
		require.NotNil(t, protoTr.Trace)
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
				require.NotNil(b, tr.Trace)
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

		// rowGroup is -1, so the page indexes are not loaded and rr is unused.
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

func TestPreloadPageIndexes(t *testing.T) {
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
		colIndex, _, _ := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
		for rowGroup := 1; rowGroup < numRowGroups; rowGroup++ {
			before := cr.reads.Load()
			require.NoError(t, preloadPageIndexes(rr, pf, rowGroup, colIndex))
			require.Equal(t, int64(2), cr.reads.Load()-before, "preload should read the column and offset index regions once each")

			// All indexes are now loaded, so per-chunk OffsetIndex calls read nothing more.
			for _, cc := range pf.RowGroups()[rowGroup].ColumnChunks() {
				oi, err := cc.OffsetIndex()
				require.NoError(t, err)
				require.Positive(t, oi.NumPages())
			}

			// The column index of the searched column is loaded too.
			_, err := pf.RowGroups()[rowGroup].ColumnChunks()[colIndex].ColumnIndex()
			require.NoError(t, err)

			require.Equal(t, int64(2), cr.reads.Load()-before, "per-chunk index calls should not read after preload")
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

func TestBackendBlockFindTraceByID_BloomAndIndexBytes(t *testing.T) {
	var (
		ctx       = context.Background()
		numTraces = 20
		ids       = make([]common.ID, numTraces)
		meta      = backend.NewBlockMeta("fake", uuid.New(), VersionString)
	)
	meta.DedicatedColumns = test.MakeDedicatedColumns()

	// sorted by trace ID, like in a real block
	for i := range ids {
		ids[i] = test.ValidTraceID(nil)
	}
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i], ids[j]) < 0 })

	traces := make([]*Trace, 0, numTraces)
	for _, id := range ids {
		tr, _ := traceToParquet(meta, id, test.MakeTrace(2, id), nil)
		traces = append(traces, tr)
	}
	b := makeBackendBlockWithTraces(t, traces)

	// objectSize returns the size of an object in the block directory, which is what a lookup reads
	objectSize := func(t *testing.T, name string) uint64 {
		obj, err := b.r.Read(ctx, name, uuid.UUID(b.meta.BlockID), b.meta.TenantID, nil)
		require.NoError(t, err)
		require.NotEmpty(t, obj)
		return uint64(len(obj))
	}
	bloomSize := func(t *testing.T, id common.ID) uint64 {
		return objectSize(t, common.BloomName(common.ShardKeyForTraceID(id, int(b.meta.BloomShardCount))))
	}

	// a trace ID that is not in the block and rejected by the bloom filter
	absentID := func(t *testing.T) common.ID {
		for range 1000 {
			id := test.ValidTraceID(nil)
			found, _, err := b.checkBloom(ctx, id)
			require.NoError(t, err)
			if !found {
				return id
			}
		}
		require.Fail(t, "no trace ID rejected by the bloom filter")
		return nil
	}

	// bigger than the last trace ID of the block, so the index rules it out
	beyondLastID := bytes.Repeat([]byte{0xff}, 16)

	// failingBlock returns a copy of the block that fails reads of the given object with failErr, "bloom" matches every bloom shard
	failingBlock := func(failName string, failErr error) *backendBlock {
		rawR := &backend.MockRawReader{
			ReadFn: func(_ context.Context, name string, _ backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
				if name == failName || (failName == "bloom" && strings.HasPrefix(name, "bloom")) {
					return nil, 0, failErr
				}
				obj, err := b.r.Read(ctx, name, uuid.UUID(b.meta.BlockID), b.meta.TenantID, cacheInfo)
				return io.NopCloser(bytes.NewReader(obj)), int64(len(obj)), err
			},
		}
		return newBackendBlock(b.meta, backend.NewReader(rawR))
	}

	// blockWithBloom returns a copy of the block that serves the given bloom filter for every shard
	blockWithBloom := func(t *testing.T, filter *bloom.BloomFilter) (*backendBlock, uint64) {
		buf := &bytes.Buffer{}
		_, err := filter.WriteTo(buf)
		require.NoError(t, err)
		bloomObj := buf.Bytes()

		rawR := &backend.MockRawReader{
			ReadFn: func(_ context.Context, name string, _ backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
				if strings.HasPrefix(name, "bloom") {
					return io.NopCloser(bytes.NewReader(bloomObj)), int64(len(bloomObj)), nil
				}
				obj, err := b.r.Read(ctx, name, uuid.UUID(b.meta.BlockID), b.meta.TenantID, cacheInfo)
				return io.NopCloser(bytes.NewReader(obj)), int64(len(obj)), err
			},
		}
		return newBackendBlock(b.meta, backend.NewReader(rawR)), uint64(len(bloomObj))
	}

	t.Run("trace found", func(t *testing.T) {
		for _, indexEnabled := range []string{"0", EnvVarIndexEnabledValue} {
			t.Run(EnvVarIndexName+"="+indexEnabled, func(t *testing.T) {
				t.Setenv(EnvVarIndexName, indexEnabled)

				expectedBytes := bloomSize(t, ids[3])
				if indexEnabled == EnvVarIndexEnabledValue {
					expectedBytes += objectSize(t, common.NameIndex)
				}

				// the data file is read on top of the bloom filter and index
				opts := common.DefaultSearchOptions()
				if opts.TraceByIDReadBufferSize > 0 {
					opts.ReadBufferSize = opts.TraceByIDReadBufferSize
				}
				_, rowGroup, _, err := b.checkIndex(ctx, ids[3])
				require.NoError(t, err)
				pf, rr, err := b.openForSearch(ctx, opts)
				require.NoError(t, err)
				_, err = findTraceByID(ctx, ids[3], b.meta, pf, rr, rowGroup)
				require.NoError(t, err)
				expectedBytes += rr.BytesRead()

				resp, err := b.FindTraceByID(ctx, ids[3], common.DefaultSearchOptions())
				require.NoError(t, err)
				require.NotNil(t, resp.Trace)
				require.Equal(t, expectedBytes, resp.Metrics.InspectedBytes)
			})
		}
	})

	t.Run("rejected by bloom filter", func(t *testing.T) {
		id := absentID(t)

		resp, err := b.FindTraceByID(ctx, id, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Trace)
		require.Equal(t, bloomSize(t, id), resp.Metrics.InspectedBytes)
	})

	t.Run("rejected by index", func(t *testing.T) {
		t.Setenv(EnvVarIndexName, EnvVarIndexEnabledValue)

		// the bloom filter lets the ID pass, so the index is what rules the block out
		filter := bloom.NewWithEstimates(100, 0.01)
		filter.Add(beyondLastID)
		block, bloomBytes := blockWithBloom(t, filter)

		resp, err := block.FindTraceByID(ctx, beyondLastID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Trace)
		require.Equal(t, bloomBytes+objectSize(t, common.NameIndex), resp.Metrics.InspectedBytes)
	})

	t.Run("check bloom", func(t *testing.T) {
		found, bytesRead, err := b.checkBloom(ctx, ids[0])
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, bloomSize(t, ids[0]), bytesRead)

		found, bytesRead, err = failingBlock("bloom", errors.New("boom")).checkBloom(ctx, ids[0])
		require.Error(t, err)
		require.False(t, found)
		require.Zero(t, bytesRead)
	})

	t.Run("check index", func(t *testing.T) {
		testCases := []struct {
			name          string
			indexEnabled  string
			block         *backendBlock
			id            common.ID
			expectedOK    bool
			expectedBytes uint64
			expectedErr   bool
		}{
			{
				name:         "disabled",
				indexEnabled: "0",
				block:        b,
				id:           beyondLastID,
				expectedOK:   true,
			},
			{
				name:          "trace may be in the block",
				indexEnabled:  EnvVarIndexEnabledValue,
				block:         b,
				id:            ids[5],
				expectedOK:    true,
				expectedBytes: objectSize(t, common.NameIndex),
			},
			{
				name:          "ruled out by the index",
				indexEnabled:  EnvVarIndexEnabledValue,
				block:         b,
				id:            beyondLastID,
				expectedOK:    false,
				expectedBytes: objectSize(t, common.NameIndex),
			},
			{
				name:         "index does not exist",
				indexEnabled: EnvVarIndexEnabledValue,
				block:        failingBlock(common.NameIndex, backend.ErrDoesNotExist),
				id:           beyondLastID,
				expectedOK:   true,
			},
			{
				name:         "read error",
				indexEnabled: EnvVarIndexEnabledValue,
				block:        failingBlock(common.NameIndex, errors.New("boom")),
				id:           ids[5],
				expectedErr:  true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Setenv(EnvVarIndexName, tc.indexEnabled)

				ok, _, bytesRead, err := tc.block.checkIndex(ctx, tc.id)
				if tc.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, tc.expectedOK, ok)
				require.Equal(t, tc.expectedBytes, bytesRead)
			})
		}
	})

	t.Run("read error", func(t *testing.T) {
		resp, err := failingBlock("bloom", errors.New("boom")).FindTraceByID(ctx, ids[0], common.DefaultSearchOptions())
		require.Error(t, err)
		require.Nil(t, resp)
	})
}
