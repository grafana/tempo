package vparquet5

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// buildTraceIDProjectionTestBlock writes a block containing exactly the
// given raw trace IDs -- some deliberately shorter than 16 bytes, in the
// style of 64-bit Jaeger IDs -- through the real write path
// (traceToParquet -> streamingBlock.Add), so the projection reader under
// test is exercised against the padding schema.go actually produces on
// disk, not a test-local reimplementation of it. Returns the block and the
// expected canonical (left-zero-padded) 16-byte IDs, sorted ascending:
// FindTraceByID (used below as an independent trusted cross-check)
// binary-searches row-group bounds and requires the TraceID column to have
// been written in ascending order.
func buildTraceIDProjectionTestBlock(t *testing.T, rawIDs [][]byte) (*backendBlock, [][]byte) {
	rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
	require.NoError(t, err)

	var (
		ctx = context.Background()
		r   = backend.NewReader(rawR)
		w   = backend.NewWriter(rawW)
		cfg = &common.BlockConfig{
			BloomFP:             0.01,
			BloomShardSizeBytes: 100 * 1024,
		}
	)

	type idPair struct {
		raw    []byte
		padded []byte
	}
	pairs := make([]idPair, len(rawIDs))
	for i, id := range rawIDs {
		pairs[i] = idPair{raw: id, padded: util.PadTraceIDTo16Bytes(id)}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return bytes.Compare(pairs[i].padded, pairs[j].padded) < 0
	})

	meta := backend.NewBlockMeta("test-tenant", uuid.New(), VersionString)
	meta.TotalObjects = int64(len(pairs))

	s, newMeta := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)

	buffer := &Trace{}
	wantPadded := make([][]byte, len(pairs))
	for i, p := range pairs {
		tr := test.MakeTrace(2, append([]byte{}, p.raw...))
		// traceToParquet derives buffer.TraceID from p.raw via
		// util.PadTraceIDTo16Bytes -- the real write-side padding step,
		// exercised here with genuinely short raw IDs.
		traceToParquet(newMeta, p.raw, tr, buffer)
		require.NoError(t, s.Add(buffer, 0, 0))
		wantPadded[i] = p.padded

		if i%5 == 0 {
			_, err := s.Flush()
			require.NoError(t, err)
		}
	}
	_, err = s.Complete()
	require.NoError(t, err)

	return newBackendBlock(newMeta, r), wantPadded
}

func randomTraceID(t *testing.T) []byte {
	id := make([]byte, 16)
	_, err := rand.Read(id)
	require.NoError(t, err)
	return id
}

// TestTraceIDProjection enumerates a known small synthetic block's trace
// IDs and cross-checks the exact set against FindTraceByID, an existing,
// independently trusted read path.
func TestTraceIDProjection(t *testing.T) {
	var rawIDs [][]byte
	for i := 0; i < 20; i++ {
		rawIDs = append(rawIDs, randomTraceID(t))
	}
	// 64-bit Jaeger-style IDs, shorter than the canonical 16 bytes.
	// DESIGN.md's canonical-hashing invariant (§ Design constraints)
	// depends on every reader producing the identical padded form; assert
	// the projection returns these already padded, not merely passed
	// through unchanged.
	rawIDs = append(
		rawIDs,
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	)

	block, wantPadded := buildTraceIDProjectionTestBlock(t, rawIDs)
	ctx := context.Background()

	iter, err := block.openTraceIDReader(ctx)
	require.NoError(t, err)

	var n int
	got := make(map[string][]byte, len(wantPadded))
	for {
		id, err := iter.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		require.Len(t, id, 16, "trace ID projection must already be the canonical 16-byte padded form")
		n++
		got[string(id)] = append([]byte{}, id...)
	}
	iter.Close()
	iter.Close() // Close must be safe to call more than once.

	assert.Equal(t, len(wantPadded), n, "projection produced a different number of rows than were written")
	require.Len(t, got, len(wantPadded), "projection returned a different number of distinct trace IDs than were written")

	for _, padded := range wantPadded {
		gotID, ok := got[string(padded)]
		if assert.True(t, ok, "projection is missing trace ID %s", util.TraceIDToHexString(padded)) {
			assert.Equal(t, padded, gotID)
		}

		// Cross-check against FindTraceByID, an independent, already-
		// trusted read path: every ID the projection reports must
		// actually be present in the block.
		resp, err := block.FindTraceByID(ctx, padded, common.DefaultSearchOptions())
		require.NoError(t, err)
		if assert.NotNil(t, resp, "FindTraceByID could not confirm trace ID %s, but the projection returned it", util.TraceIDToHexString(padded)) {
			assert.NotNil(t, resp.Trace)
		}
	}
}

// TestTraceIDProjection_CloseIsIdempotent asserts Close is safe to call
// more than once, independent of whatever the underlying pq.SyncIterator
// happens to do on repeated Close (its span.End() etc.).
func TestTraceIDProjection_CloseIsIdempotent(t *testing.T) {
	block, _ := buildTraceIDProjectionTestBlock(t, [][]byte{randomTraceID(t)})

	iter, err := block.openTraceIDReader(context.Background())
	require.NoError(t, err)

	require.NotPanics(t, func() {
		iter.Close()
		iter.Close()
	})
}

// TestTraceIDProjection_ContextCancellation asserts that a cancelled
// context is honored promptly: Next never returns a row (nor blocks) once
// its ctx argument is done, whether that happens before the first call or
// partway through iteration.
func TestTraceIDProjection_ContextCancellation(t *testing.T) {
	var rawIDs [][]byte
	for i := 0; i < 50; i++ {
		rawIDs = append(rawIDs, randomTraceID(t))
	}
	block, _ := buildTraceIDProjectionTestBlock(t, rawIDs)

	t.Run("cancelled before the first call", func(t *testing.T) {
		iter, err := block.openTraceIDReader(context.Background())
		require.NoError(t, err)
		defer iter.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = iter.Next(ctx)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("cancelled mid-iteration", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		iter, err := block.openTraceIDReader(ctx)
		require.NoError(t, err)
		defer iter.Close()

		for i := 0; i < 5; i++ {
			_, err := iter.Next(ctx)
			require.NoError(t, err)
		}

		cancel()

		_, err = iter.Next(ctx)
		require.ErrorIs(t, err, context.Canceled)
	})
}
