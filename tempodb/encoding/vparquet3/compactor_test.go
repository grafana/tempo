package vparquet3

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

	tempo_io "github.com/grafana/tempo/pkg/io"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func BenchmarkCompactor(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		benchmarkCompactor(b, 1000, 100, 100) // 10M spans
	})
	b.Run("Medium", func(b *testing.B) {
		benchmarkCompactor(b, 100, 100, 1000) // 10M spans
	})
	b.Run("Large", func(b *testing.B) {
		benchmarkCompactor(b, 10, 1000, 1000) // 10M spans
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

	meta := createTestBlock(b, ctx, cfg, r, w, traceCount, batchCount, spanCount, nil)

	inputs := []*backend.BlockMeta{meta}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c := NewCompactor(common.CompactionOptions{
			BlockConfig:      *cfg,
			OutputBlocks:     1,
			FlushSizeBytes:   30_000_000,
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
	meta := createTestBlock(b, ctx, cfg, r, w, 10, 1000, 1000, nil)
	inputs := []*backend.BlockMeta{meta, meta}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c := NewCompactor(common.CompactionOptions{
			BlockConfig:      *cfg,
			OutputBlocks:     1,
			FlushSizeBytes:   30_000_000,
			MaxBytesPerTrace: 50_000_000,
			ObjectsCombined:  func(compactionLevel, objects int) {},
			SpansDiscarded:   func(traceID, rootSpanName string, rootServiceName string, spans int) {},
		})

		_, err = c.Compact(ctx, l, r, w, inputs)
		require.NoError(b, err)
	}
}

// createTestBlock with the number of given traces and the needed sizes.
// Trace IDs are guaranteed to be monotonically increasing so that
// the block will be iterated in order.
// nolint: revive
func createTestBlock(t testing.TB, ctx context.Context, cfg *common.BlockConfig, r backend.Reader, w backend.Writer, traceCount, batchCount, spanCount int, dc backend.DedicatedColumns) *backend.BlockMeta {
	inMeta := &backend.BlockMeta{
		TenantID:         tenantID,
		BlockID:          uuid.New(),
		TotalObjects:     traceCount,
		DedicatedColumns: dc,
	}

	sb := newStreamingBlock(ctx, cfg, inMeta, r, w, tempo_io.NewBufferedWriter)

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

	return sb.meta
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

	sch := parquet.SchemaOf(new(Trace))
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
		FlushSizeBytes:  30_000_000,
		ObjectsCombined: func(compactionLevel, objects int) {},
	})

	dedicatedColumns := backend.DedicatedColumns{
		{Scope: "resource", Name: "dedicated.resource.1", Type: "string"},
		{Scope: "span", Name: "dedicated.span.1", Type: "string"},
	}

	meta1 := createTestBlock(t, context.Background(), &blockConfig, r, w, 10, 10, 10, dedicatedColumns)
	meta2 := createTestBlock(t, context.Background(), &blockConfig, r, w, 10, 10, 10, dedicatedColumns)

	inputs := []*backend.BlockMeta{meta1, meta2}

	newMeta, err := c.Compact(context.Background(), log.NewNopLogger(), r, w, inputs)
	require.NoError(t, err)
	require.Len(t, newMeta, 1)
	require.Equal(t, 20, newMeta[0].TotalObjects)
	require.Equal(t, dedicatedColumns, newMeta[0].DedicatedColumns)
}
