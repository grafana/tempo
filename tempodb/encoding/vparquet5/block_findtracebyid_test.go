package vparquet5

import (
	"bytes"
	"context"
	"os"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
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

	blocks, _, err := r.Blocks(ctx, "single-tenant")
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
