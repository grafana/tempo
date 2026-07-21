package vparquet5

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"errors"
	"flag"
	"io"
	"math/rand"
	"sort"
	"testing"

	"github.com/go-kit/log"
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

	sch, _, _ := SchemaWithDynamicChanges(backend.DedicatedColumns{})
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

// createTestBlockWithIDs creates a block containing exactly the given trace IDs. Unlike
// createTestBlock (random IDs), this lets tests control which IDs land in which input block
// so dedup/attribution across blocks can be exercised deterministically. IDs are sorted
// ascending before writing: compaction's multiblock merge is a merge-sort across bookmarks
// that assumes each input block is already sorted by trace ID (like real WAL-flushed
// blocks), so an unsorted block would make cross-block ID matches miss their merge step.
func createTestBlockWithIDs(ctx context.Context, t testing.TB, cfg *common.BlockConfig, r backend.Reader, w backend.Writer, ids []common.ID) *backend.BlockMeta {
	sorted := append([]common.ID(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool { return bytes.Compare(sorted[i], sorted[j]) < 0 })

	inMeta := &backend.BlockMeta{
		TenantID:     tenantID,
		BlockID:      backend.NewUUID(),
		TotalObjects: int64(len(sorted)),
	}

	sb, outMeta := newStreamingBlock(ctx, cfg, inMeta, r, w, tempo_io.NewBufferedWriter)

	for _, id := range sorted {
		tr := test.MakeTraceWithSpanCount(1, 1, id)
		trp, _ := traceToParquet(inMeta, id, tr, nil)
		require.NoError(t, sb.Add(trp, 0, 0))
	}

	_, err := sb.Complete()
	require.NoError(t, err)

	return outMeta
}

// TestCompact_Callbacks_NilIsNoOp is a regression guard: compaction must work exactly as
// before when ObjectIDWritten/OutputBlockCompleted are left nil (the default for all
// existing callers), i.e. wiring them in must not remove the nil-check at each call site.
func TestCompact_Callbacks_NilIsNoOp(t *testing.T) {
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
		// ObjectIDWritten and OutputBlockCompleted intentionally left nil.
	})

	meta1 := createTestBlock(t, context.Background(), &blockConfig, r, w, 10, 10, 10, 1, nil)
	meta2 := createTestBlock(t, context.Background(), &blockConfig, r, w, 10, 10, 10, 1, nil)

	newMeta, err := c.Compact(context.Background(), log.NewNopLogger(), r, w, []*backend.BlockMeta{meta1, meta2})
	require.NoError(t, err)
	require.Len(t, newMeta, 1)
	require.Equal(t, int64(20), newMeta[0].TotalObjects)
}

// TestCompact_ObjectIDWritten_FiresForEveryWrittenID compacts two input blocks that share
// one trace ID (present in both) plus one ID unique to each, and asserts ObjectIDWritten
// fires exactly once per distinct output ID -- i.e. the shared ID is combined/deduped across
// inputs before reaching the output block, and is reported exactly once, not twice.
func TestCompact_ObjectIDWritten_FiresForEveryWrittenID(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)

	blockConfig := common.BlockConfig{Version: VersionString}
	blockConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	require.NoError(t, common.ValidateConfig(&blockConfig))

	id1 := make(common.ID, 16)
	id2 := make(common.ID, 16)
	shared := make(common.ID, 16)
	_, err = crand.Read(id1)
	require.NoError(t, err)
	_, err = crand.Read(id2)
	require.NoError(t, err)
	_, err = crand.Read(shared)
	require.NoError(t, err)

	meta1 := createTestBlockWithIDs(context.Background(), t, &blockConfig, r, w, []common.ID{id1, shared})
	meta2 := createTestBlockWithIDs(context.Background(), t, &blockConfig, r, w, []common.ID{id2, shared})

	var got []common.ID
	c := NewCompactor(common.CompactionOptions{
		BlockConfig:       blockConfig,
		OutputBlocks:      1,
		ObjectsCombined:   func(_, _ int) {},
		DedupedSpans:      func(_, _ int) {},
		DisconnectedTrace: func() {},
		RootlessTrace:     func() {},
		ObjectIDWritten: func(id common.ID) {
			// Defensive copy: the multiblock iterator's row/ID buffers get reused
			// on subsequent Next() calls.
			got = append(got, append(common.ID(nil), id...))
		},
	})

	newMeta, err := c.Compact(context.Background(), log.NewNopLogger(), r, w, []*backend.BlockMeta{meta1, meta2})
	require.NoError(t, err)
	require.Len(t, newMeta, 1)
	require.Equal(t, int64(3), newMeta[0].TotalObjects, "shared must be combined into a single output row")
	require.ElementsMatch(t, []common.ID{id1, id2, shared}, got)
}

// TestCompact_MultiOutput_AttributionExact is the critical attribution test: it forces
// rotation to multiple output blocks (OutputBlocks: 2 drives recordsPerBlock down to
// idsPerBlock) and verifies that the IDs attributed via ObjectIDWritten/OutputBlockCompleted
// to each output block exactly match what was actually persisted to that block, that every
// input ID lands in exactly one output block, and that the callback meta pointers are the
// same objects returned in newCompactedBlocks.
func TestCompact_MultiOutput_AttributionExact(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)

	blockConfig := common.BlockConfig{Version: VersionString}
	blockConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	require.NoError(t, common.ValidateConfig(&blockConfig))

	const idsPerBlock = 10
	ids1 := make([]common.ID, idsPerBlock)
	ids2 := make([]common.ID, idsPerBlock)
	for i := range ids1 {
		ids1[i] = make(common.ID, 16)
		_, err = crand.Read(ids1[i])
		require.NoError(t, err)
	}
	for i := range ids2 {
		ids2[i] = make(common.ID, 16)
		_, err = crand.Read(ids2[i])
		require.NoError(t, err)
	}

	meta1 := createTestBlockWithIDs(context.Background(), t, &blockConfig, r, w, ids1)
	meta2 := createTestBlockWithIDs(context.Background(), t, &blockConfig, r, w, ids2)

	allInputIDs := make(map[string]struct{}, idsPerBlock*2)
	for _, id := range append(append([]common.ID{}, ids1...), ids2...) {
		allInputIDs[string(id)] = struct{}{}
	}

	// perOutputIDs[i] collects the IDs attributed by ObjectIDWritten to the i-th completed
	// output block; `current` is reset every time OutputBlockCompleted fires.
	var (
		perOutputIDs  [][]common.ID
		perOutputMeta []*backend.BlockMeta
		current       []common.ID
	)

	c := NewCompactor(common.CompactionOptions{
		BlockConfig:     blockConfig,
		OutputBlocks:    2,
		ObjectsCombined: func(_, _ int) {},
		ObjectIDWritten: func(id common.ID) {
			current = append(current, append(common.ID(nil), id...))
		},
		OutputBlockCompleted: func(meta *backend.BlockMeta) {
			perOutputIDs = append(perOutputIDs, current)
			perOutputMeta = append(perOutputMeta, meta)
			current = nil
		},
	})

	newCompactedBlocks, err := c.Compact(context.Background(), log.NewNopLogger(), r, w, []*backend.BlockMeta{meta1, meta2})
	require.NoError(t, err)
	require.Len(t, newCompactedBlocks, 2, "OutputBlocks: 2 with 20 evenly-divisible input records must yield 2 output blocks")
	require.Len(t, perOutputIDs, 2)

	seen := make(map[string]int, idsPerBlock*2)
	for i, meta := range newCompactedBlocks {
		require.Same(t, meta, perOutputMeta[i], "callback meta must be the exact object returned in newCompactedBlocks")

		block := newBackendBlock(meta, r)
		iter, err := block.openTraceIDReader(context.Background())
		require.NoError(t, err)

		var actualIDs []common.ID
		for {
			id, err := iter.Next(context.Background())
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			actualIDs = append(actualIDs, id)
		}
		iter.Close()

		require.ElementsMatch(t, perOutputIDs[i], actualIDs, "captured IDs for output block %d must match its real persisted contents", i)
		require.Len(t, actualIDs, idsPerBlock)

		for _, id := range actualIDs {
			seen[string(id)]++
		}
	}

	require.Len(t, seen, len(allInputIDs))
	for idStr := range allInputIDs {
		require.Equal(t, 1, seen[idStr], "id %x must land in exactly one output block", idStr)
	}
}
