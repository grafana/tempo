package blockbuilder

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestLiveTracesIter_DedupSpans(t *testing.T) {
	const spanCount = 5

	traceID := generateTraceID(t)
	tr := test.MakeTraceWithSpanCount(1, spanCount, traceID)

	trBytes, err := tr.Marshal()
	require.NoError(t, err)

	// Push the same trace bytes twice to simulate replicated writes
	lt := livetraces.New(func(b []byte) uint64 { return uint64(len(b)) }, 0, 0, "test-tenant")
	lt.Push(traceID, trBytes, 0)
	lt.Push(traceID, trBytes, 0)

	iter := newLiveTracesIter(lt)
	ctx := context.Background()

	id, resultTr, err := iter.Next(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	require.NotNil(t, resultTr)

	// Exhaust the iterator
	_, _, err = iter.Next(ctx)
	require.NoError(t, err)

	// Duplicate push should be fully deduped - only the original spans remain
	total := 0
	for _, rs := range resultTr.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			total += len(ss.Spans)
		}
	}
	require.Equal(t, spanCount, total)
	n, err := iter.DedupedSpans()
	require.NoError(t, err)
	require.Equal(t, uint32(spanCount), n)
}

func TestLiveTracesIter_DedupedSpans_ErrorWhenNotExhausted(t *testing.T) {
	// Push 11 traces so iter() produces 2 chunks (chunk size is 10).
	// Without reading from the channel the buffer fills after the first chunk,
	// causing iter() to block on the second send. Cancelling via Close() makes
	// it exit early without setting exhausted, so DedupedSpans returns an error.
	lt := livetraces.New(func(b []byte) uint64 { return uint64(len(b)) }, 0, 0, "test-tenant")
	for j := 0; j < 11; j++ {
		traceID := generateTraceID(t)
		tr := test.MakeTraceWithSpanCount(1, 1, traceID)
		trBytes, err := tr.Marshal()
		require.NoError(t, err)
		lt.Push(traceID, trBytes, 0)
	}

	iter := newLiveTracesIter(lt)
	iter.Close() // cancel before exhausting

	_, err := iter.DedupedSpans() // blocks until iter() exits, then returns error
	require.Error(t, err)
}
