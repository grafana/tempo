package livestore

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/util"
	"github.com/grafana/tempo/v3/pkg/util/test"
)

// A read must not replace live batches with older batches from storage/WAL, even
// when the combiner can append into the live slice's backing array.
//
// With span 1 already in WAL and span 2 live, the buggy query does this:
//
//	                      alias live slice    append WAL batch    sort by time
//	shared array (cap 2)   [2 | unused]     ->  [2 | 1]         ->  [1 | 2]
//	live view (len 1)      [2]                  [2]                 [1]
//	combiner view          [2]                  [2, 1]              [1, 2]
//	                                                                 ^
//	                                                     outside live length
//
// The query returns both spans, but the next flush writes span 1 again and
// loses span 2. With cap 1, append allocates a new array and live stays [2].
func TestInstanceFindByTraceIDPreservesLiveBatches(t *testing.T) {
	i, ls := defaultInstanceAndTmpDir(t)
	defer func() { require.NoError(t, services.StopAndAwaitTerminated(t.Context(), ls)) }()
	traceID := test.ValidTraceID(nil)
	start := uint64(time.Now().UnixNano())

	// This is a trace that has been stored in the WAL or flushed in single binary mode
	stored := test.WrapSpansAsTrace(test.MakeSpanPruningSpan(
		traceID, binary.BigEndian.AppendUint64(nil, 1), nil, "span-1", start, start+500,
	))

	// Live trace
	pending := test.WrapSpansAsTrace(test.MakeSpanPruningSpan(
		traceID, binary.BigEndian.AppendUint64(nil, 2), nil, "span-2", start+1000, start+1500,
	))
	ids := func(tr *tempopb.Trace) []uint64 {
		var result []uint64
		for _, span := range test.AllSpansInTrace(tr) {
			result = append(result, binary.BigEndian.Uint64(span.SpanId))
		}
		return result
	}
	flush := func() {
		drained, err := i.cutIdleTraces(t.Context(), true)
		require.NoError(t, err)
		require.True(t, drained)
	}
	pushTrace(t.Context(), t, i, stored, traceID)
	flush()
	pushTrace(t.Context(), t, i, pending, traceID)

	live := i.liveTraces.Traces[util.HashForTraceID(traceID)]
	require.NotNil(t, live)
	require.Len(t, live.Batches, 1)
	// Leave room for the stored batch so append can reuse the live array.
	batches := make([]*tracev1.ResourceSpans, 1, 2)
	copy(batches, live.Batches)
	// We assign the 2 cap slice to the live traces batches
	live.Batches = batches
	before := ids(&tempopb.Trace{ResourceSpans: live.Batches})
	require.Equal(t, []uint64{2}, before)

	resp, err := i.FindByTraceID(t.Context(), traceID, false)
	require.NoError(t, err)
	require.NotNil(t, resp.Trace)
	assert.ElementsMatch(t, []uint64{1, 2}, ids(resp.Trace), "query returns both spans")
	assert.Equal(t, before, ids(&tempopb.Trace{ResourceSpans: live.Batches}), "query must preserve live span IDs")

	flush()
	blockID, err := i.cutBlocks(t.Context(), true)
	require.NoError(t, err)
	_, err = i.completeBlock(t.Context(), blockID)
	require.NoError(t, err)
	resp, err = i.FindByTraceID(t.Context(), traceID, false)
	require.NoError(t, err)
	require.NotNil(t, resp.Trace)
	assert.ElementsMatch(t, []uint64{1, 2}, ids(resp.Trace), "both spans must survive flush and Parquet completion")
}
