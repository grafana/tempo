package vparquet2

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"github.com/go-kit/log"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/parquet-go/parquet-go"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	meta := createTestBlock(b, ctx, cfg, r, w, traceCount, batchCount, spanCount)

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
	meta := createTestBlock(b, ctx, cfg, r, w, 10, 1000, 1000)
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
func createTestBlock(t testing.TB, ctx context.Context, cfg *common.BlockConfig, r backend.Reader, w backend.Writer, traceCount, batchCount, spanCount int) *backend.BlockMeta {
	inMeta := &backend.BlockMeta{
		TenantID:     tenantID,
		BlockID:      backend.NewUUID(),
		TotalObjects: int64(traceCount),
	}

	sb := newStreamingBlock(ctx, cfg, inMeta, r, w, tempo_io.NewBufferedWriter)

	for i := 0; i < traceCount; i++ {
		id := make([]byte, 16)
		binary.LittleEndian.PutUint64(id, uint64(i))

		tr := test.MakeTraceWithSpanCount(batchCount, spanCount, id)
		trp := traceToParquet(id, tr, nil)

		err := sb.Add(trp, 0, 0)
		require.NoError(t, err)
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
	trp := traceToParquet(traceID, tr, nil)
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

type slowWriter struct {
	backend.Writer
	wait chan struct{}
}

func (w *slowWriter) WriteBlockMeta(ctx context.Context, meta *backend.BlockMeta) error {
	w.wait <- struct{}{} // send a signal to a goroutine
	<-w.wait             // wait for the signal from the goroutine
	return w.Writer.WriteBlockMeta(ctx, meta)
}

func TestWriteBlockMetaWithNoCompactFlag(t *testing.T) {
	tempDir := t.TempDir()

	r, w, _, err := local.New(&local.Config{
		Path: tempDir,
	})
	require.NoError(t, err)

	for _, withNoCompactFlag := range []bool{true, false} {
		t.Run(fmt.Sprintf("withNoCompactFlag=%t", withNoCompactFlag), func(t *testing.T) {
			ctx := t.Context()
			meta := &backend.BlockMeta{
				BlockID:  backend.NewUUID(),
				TenantID: "test-tenant",
			}

			waitChan := make(chan struct{})
			reader := backend.NewReader(r)
			writer := &slowWriter{
				Writer: backend.NewWriter(w),
				wait:   waitChan,
			}

			// Create a streamingBlock with nocompact flag enabled
			cfg := &common.BlockConfig{
				BloomFP:                 0.01,
				BloomShardSizeBytes:     1024,
				CreateWithNoCompactFlag: withNoCompactFlag,
			}
			streamingBlock := newStreamingBlock(ctx, cfg, meta, reader, writer, tempo_io.NewBufferedWriter)

			go func() {
				<-waitChan // writing block meta started, stopping it to emulate a slow write
				hasFlag, err := reader.HasNoCompactFlag(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
				require.NoError(t, err)
				assert.Equal(t, withNoCompactFlag, hasFlag, fmt.Sprintf("nocompact flag should be %t in the middle of writing", withNoCompactFlag))
				waitChan <- struct{}{} // we checked flag, proceed with writing
			}()

			// Complete the streamingBlock - this should write nocompact flag first, then all block data
			_, err = streamingBlock.Complete()
			require.NoError(t, err)

			// Verify nocompact flag remains after successful Complete (flag removal is done at higher level)
			hasFlag, err := reader.HasNoCompactFlag(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
			require.NoError(t, err)
			assert.Equal(t, withNoCompactFlag, hasFlag, fmt.Sprintf("nocompact flag should be %t after successful Complete", withNoCompactFlag))

			// Verify meta.json was written
			blockMeta, err := reader.BlockMeta(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
			require.NoError(t, err)
			assert.Equal(t, meta.BlockID, blockMeta.BlockID)
			assert.Equal(t, meta.TenantID, blockMeta.TenantID)
		})
	}
}
