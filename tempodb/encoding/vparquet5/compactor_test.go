package vparquet5

import (
	"context"
	crand "crypto/rand"
	"flag"
	"math/rand"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/v3/pkg/io"
	tempoUtil "github.com/grafana/tempo/v3/pkg/util"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

func BenchmarkCompactor(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		benchmarkCompactor(b, 10000, 10, 10) // 1M spans total, 100 spans per trace
	})
	b.Run("Medium", func(b *testing.B) {
		benchmarkCompactor(b, 100, 100, 100) // 1M spans total, 10K spans per trace
	})
	b.Run("Large", func(b *testing.B) {
		benchmarkCompactor(b, 10, 100, 1000) // 1M spans total, 100K spans per trace
	})
}

func benchmarkCompactor(b *testing.B, traceCount, batchCount, spanCount int) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: b.TempDir(),
	})
	require.NoError(b, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	ctx := context.Background()
	l := log.NewNopLogger()

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
		RowGroupSizeBytes:   20_000_000,
	}

	meta := createTestBlock(b, ctx, cfg, r, w, traceCount, batchCount, spanCount, 0, nil)

	inputs := []*backend.BlockMeta{meta}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c := NewCompactor(common.CompactionOptions{
			BlockConfig:      *cfg,
			OutputBlocks:     1,
			MaxBytesPerTrace: 50_000_000,
		})

		_, err = c.Compact(ctx, l, r, w, inputs)
		require.NoError(b, err)
	}
}

func BenchmarkCompactorDupes(b *testing.B) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: b.TempDir(),
	})
	require.NoError(b, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	ctx := context.Background()
	l := log.NewNopLogger()

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
		RowGroupSizeBytes:   20_000_000,
	}

	// 1M span traces
	meta := createTestBlock(b, ctx, cfg, r, w, 10, 1000, 1000, 1, nil)
	inputs := []*backend.BlockMeta{meta, meta}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c := NewCompactor(common.CompactionOptions{
			BlockConfig:      *cfg,
			OutputBlocks:     1,
			MaxBytesPerTrace: 50_000_000,
			ObjectsCombined:  func(_, _ int) {},
			SpansDiscarded:   func(_, _, _ string, _ int) {},
		})

		_, err = c.Compact(ctx, l, r, w, inputs)
		require.NoError(b, err)
	}
}

// createTestBlock with the number of given traces and the needed sizes.
// Trace IDs are guaranteed to be monotonically increasing so that
// the block will be iterated in order.
// nolint: revive
func createTestBlock(t testing.TB, ctx context.Context, cfg *common.BlockConfig, r backend.Reader, w backend.Writer, traceCount, batchCount, spanCount, replicationFactor int, dc backend.DedicatedColumns) *backend.BlockMeta {
	inMeta := &backend.BlockMeta{
		TenantID:          tenantID,
		BlockID:           backend.NewUUID(),
		TotalObjects:      int64(traceCount),
		ReplicationFactor: uint32(replicationFactor),
		DedicatedColumns:  dc,
	}

	sb, outMeta := newStreamingBlock(ctx, cfg, inMeta, r, w, tempo_io.NewBufferedWriter)

	for i := 0; i < traceCount; i++ {
		id := make([]byte, 16)
		_, err := crand.Read(id)
		require.NoError(t, err)

		tr := test.AddDedicatedAttributes(test.MakeTraceWithSpanCount(batchCount, spanCount, id))
		trp, connected := traceToParquet(inMeta, id, tr, nil)
		require.False(t, connected)

		require.NoError(t, sb.Add(trp, 0, 0))
		if sb.EstimatedBufferedBytes() > 20_000_000 {
			_, err := sb.Flush()
			require.NoError(t, err)
		}
	}

	_, err := sb.Complete()
	require.NoError(t, err)

	return outMeta
}

func TestValueAlloc(_ *testing.T) {
	_ = make([]parquet.Value, 1_000_000)
}

func TestCountSpans(t *testing.T) {
	// It causes high mem usage when batchSize and spansEach are too big (> 500)
	batchSize := 300 + rand.Intn(25)
	spansEach := 250 + rand.Intn(25)

	rootSpan := "foo"
	rootService := "bar"

	sch, _, _ := SchemaWithDynamicChanges(TraceSchemaConfig{DedicatedColumns: backend.DedicatedColumns{}})
	traceID := make([]byte, 16)
	_, err := crand.Read(traceID)
	require.NoError(t, err)

	// make Trace and convert to parquet.Row
	tr := test.MakeTraceWithSpanCount(batchSize, spansEach, traceID)
	trp, connected := traceToParquet(&backend.BlockMeta{}, traceID, tr, nil)
	require.False(t, connected)
	trp.RootServiceName = rootService
	trp.RootSpanName = rootSpan
	row := sch.Deconstruct(nil, trp)

	// count spans for generated rows.
	tID, rootSpanName, rootServiceName, spans := countSpans(sch, row)
	require.Equal(t, tID, tempoUtil.TraceIDToHexString(traceID))
	require.Equal(t, spans, batchSize*spansEach)
	require.Equal(t, rootSpan, rootSpanName)
	require.Equal(t, rootService, rootServiceName)
}

func TestCompact(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)

	blockConfig := common.BlockConfig{Version: VersionString}
	blockConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	require.NoError(t, common.ValidateConfig(&blockConfig))

	c := NewCompactor(common.CompactionOptions{
		BlockConfig:     blockConfig,
		OutputBlocks:    1,
		ObjectsCombined: func(_, _ int) {},
	})

	dedicatedColumns := backend.DedicatedColumns{
		{Scope: "resource", Name: "dedicated.resource.1", Type: "string"},
		{Scope: "span", Name: "dedicated.span.1", Type: "string"},
	}

	meta1 := createTestBlock(t, context.Background(), &blockConfig, r, w, 10, 10, 10, 1, dedicatedColumns)
	meta2 := createTestBlock(t, context.Background(), &blockConfig, r, w, 10, 10, 10, 1, dedicatedColumns)

	inputs := []*backend.BlockMeta{meta1, meta2}

	newMeta, err := c.Compact(context.Background(), log.NewNopLogger(), r, w, inputs)
	require.NoError(t, err)
	require.Len(t, newMeta, 1)
	require.Equal(t, int64(20), newMeta[0].TotalObjects)
	require.Equal(t, uint32(1), newMeta[0].ReplicationFactor)
	require.Equal(t, dedicatedColumns, newMeta[0].DedicatedColumns)
}

func createSimilarityLifecycleBlock(t *testing.T, ctx context.Context, cfg *common.BlockConfig, r backend.Reader, w backend.Writer, id byte, signatureVersion uint32, similarity *TraceSimilarity) *backend.BlockMeta {
	t.Helper()

	meta := backend.NewBlockMeta(tenantID, uuid.New(), VersionString)
	meta.TraceSignatureVersion = signatureVersion
	meta.TotalObjects = 1

	sb, outMeta := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)
	traceID := make([]byte, 16)
	traceID[15] = id

	require.NoError(t, sb.Add(&Trace{
		TraceID:         traceID,
		RootServiceName: "service",
		Similarity:      similarity,
	}, 1, 2))
	_, err := sb.Complete()
	require.NoError(t, err)

	return outMeta
}

func readSimilarityLifecycleTraces(t *testing.T, ctx context.Context, meta *backend.BlockMeta, r backend.Reader) map[string]*TraceSimilarity {
	t.Helper()

	block := newBackendBlock(meta, r)
	iter, err := block.rawIter(ctx, newRowPool(2))
	require.NoError(t, err)
	defer iter.Close()

	schema, _, _ := SchemaWithDynamicChanges(TraceSchemaConfig{
		DedicatedColumns: meta.DedicatedColumns,
		SignatureVersion: meta.TraceSignatureVersion,
	})

	out := map[string]*TraceSimilarity{}
	for {
		id, row, err := iter.Next(ctx)
		if row == nil {
			require.NoError(t, err)
			break
		}
		require.NoError(t, err)

		tr := new(Trace)
		require.NoError(t, schema.Reconstruct(tr, row))
		out[string(id)] = tr.Similarity
	}

	return out
}

func TestTraceSimilarityLifecycle(t *testing.T) {
	tests := []struct {
		name            string
		unsignedVersion uint32
		signedFirst     bool
		wantErr         bool
	}{
		{name: "v1_then_v1", unsignedVersion: TraceSignatureVersionV1},
		{name: "v0_then_v1", unsignedVersion: TraceSignatureVersionNone, wantErr: true},
		{name: "v1_then_v0", unsignedVersion: TraceSignatureVersionNone, signedFirst: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
			require.NoError(t, err)

			r := backend.NewReader(rawR)
			w := backend.NewWriter(rawW)
			ctx := context.Background()

			cfg := &common.BlockConfig{Version: VersionString}
			cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			require.NoError(t, common.ValidateConfig(cfg))

			similarity := &TraceSimilarity{ShingleCount: 7}
			similarity.Signature[0] = 42

			unsignedMeta := createSimilarityLifecycleBlock(t, ctx, cfg, r, w, 1, tt.unsignedVersion, nil)
			signedMeta := createSimilarityLifecycleBlock(t, ctx, cfg, r, w, 2, TraceSignatureVersionV1, similarity)

			before := readSimilarityLifecycleTraces(t, ctx, signedMeta, r)
			require.Len(t, before, 1)
			signedTraceID := make([]byte, 16)
			signedTraceID[15] = 2
			require.Equal(t, similarity, before[string(signedTraceID)])

			inputs := []*backend.BlockMeta{unsignedMeta, signedMeta}
			if tt.signedFirst {
				inputs[0], inputs[1] = inputs[1], inputs[0]
			}

			c := NewCompactor(common.CompactionOptions{
				BlockConfig:     *cfg,
				OutputBlocks:    1,
				ObjectsCombined: func(_, _ int) {},
			})
			output, err := c.Compact(ctx, log.NewNopLogger(), r, w, inputs)
			if tt.wantErr {
				require.ErrorContains(t, err, "cannot compact blocks with different trace signature versions")
				require.Nil(t, output)
				return
			}
			require.NoError(t, err)
			require.Len(t, output, 1)
			require.Equal(t, TraceSignatureVersionV1, output[0].TraceSignatureVersion)

			after := readSimilarityLifecycleTraces(t, ctx, output[0], r)
			require.Len(t, after, 2)
			for id, similarity := range after {
				switch id[len(id)-1] {
				case 1:
					require.Nil(t, similarity)
				case 2:
					require.NotNil(t, similarity)
					require.Equal(t, uint32(7), similarity.ShingleCount)
					require.Equal(t, byte(42), similarity.Signature[0])
				default:
					t.Fatalf("unexpected trace ID %x", id)
				}
			}
		})
	}
}

func TestCompactTraceWritten(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	ctx := context.Background()

	blockConfig := common.BlockConfig{Version: VersionString}
	blockConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	require.NoError(t, common.ValidateConfig(&blockConfig))

	meta1 := createTestBlock(t, ctx, &blockConfig, r, w, 10, 10, 10, 1, nil)
	meta2 := createTestBlock(t, ctx, &blockConfig, r, w, 10, 10, 10, 1, nil)
	metaLevel2 := createTestBlock(t, ctx, &blockConfig, r, w, 10, 10, 10, 1, nil)
	metaLevel2.CompactionLevel = 2

	// Same trace ID with different spans in each block, so compaction takes the real combine path.
	sch, _, _ := SchemaWithDynamicChanges(TraceSchemaConfig{DedicatedColumns: nil, SignatureVersion: 0})
	splitID := test.ValidTraceID(nil)
	var splitFragments []*backend.BlockMeta
	largestFragment := 0
	for range 2 {
		inMeta := &backend.BlockMeta{TenantID: tenantID, BlockID: backend.NewUUID(), TotalObjects: 1, ReplicationFactor: 1}
		sb, outMeta := newStreamingBlock(ctx, &blockConfig, inMeta, r, w, tempo_io.NewBufferedWriter)
		trp, _ := traceToParquet(inMeta, splitID, test.MakeTraceWithSpanCount(1, 10, splitID), nil)
		largestFragment = max(largestFragment, estimateProtoSizeFromParquetRow(sch.Deconstruct(nil, trp)))
		require.NoError(t, sb.Add(trp, 0, 0))
		_, err := sb.Complete()
		require.NoError(t, err)
		splitFragments = append(splitFragments, outMeta)
	}

	tests := []struct {
		name      string
		inputs    []*backend.BlockMeta
		drop      bool
		wantCount int
		wantLevel int
		wantMin   int
	}{
		{name: "distinct traces", inputs: []*backend.BlockMeta{meta1, meta2}, wantCount: 20, wantMin: 1},
		{name: "duplicate traces", inputs: []*backend.BlockMeta{meta1, meta1}, wantCount: 10, wantMin: 1},
		{name: "split trace is combined", inputs: splitFragments, wantCount: 1, wantMin: largestFragment + 1},
		{name: "highest input level", inputs: []*backend.BlockMeta{meta1, metaLevel2}, wantCount: 20, wantLevel: 2, wantMin: 1},
		{name: "dropped traces", inputs: []*backend.BlockMeta{meta1}, drop: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sizes []int
			c := NewCompactor(common.CompactionOptions{
				BlockConfig:       blockConfig,
				OutputBlocks:      1,
				ObjectsCombined:   func(_, _ int) {},
				DedupedSpans:      func(_, _ int) {},
				DisconnectedTrace: func() {},
				RootlessTrace:     func() {},
				DropObject:        func(common.ID) bool { return tt.drop },
				TraceWritten: func(compactionLevel, bytes int) {
					require.Equal(t, tt.wantLevel, compactionLevel)
					sizes = append(sizes, bytes)
				},
			})

			_, err := c.Compact(ctx, log.NewNopLogger(), r, w, tt.inputs)
			require.NoError(t, err)
			require.Len(t, sizes, tt.wantCount)
			for _, s := range sizes {
				require.GreaterOrEqual(t, s, tt.wantMin)
			}
		})
	}
}
