package blockbuilder

import (
	"context"
	"testing"

	"github.com/grafana/tempo/v3/pkg/livetraces"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1_common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/util/test"
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
			require.NotEmpty(t, ss.Spans)
			total += len(ss.Spans)
		}
	}
	require.Len(t, resultTr.ResourceSpans, len(tr.ResourceSpans))
	require.Equal(t, spanCount, total)
	n, err := iter.DedupedSpans()
	require.NoError(t, err)
	require.Equal(t, uint32(spanCount), n)
}

func TestLiveTracesIter_DedupDropsEmptiedScopes(t *testing.T) {
	traceID := generateTraceID(t)
	span := func(id byte) *v1_trace.Span {
		return &v1_trace.Span{TraceId: traceID, SpanId: []byte{0, 0, 0, 0, 0, 0, 0, id}, StartTimeUnixNano: 1, EndTimeUnixNano: 2}
	}
	scope := func(attrs ...string) *v1_common.InstrumentationScope {
		sc := &v1_common.InstrumentationScope{Name: "scope"}
		for _, a := range attrs {
			sc.Attributes = append(sc.Attributes, &v1_common.KeyValue{Key: a, Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}})
		}
		return sc
	}

	// A duplicate of span 2 arrives under a scope with an extra attribute, so it
	// is kept apart from the original scope and dedup leaves it with no spans.
	original := &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{{
		ScopeSpans: []*v1_trace.ScopeSpans{{Scope: scope(), Spans: []*v1_trace.Span{span(1), span(2)}}},
	}}}
	resent := &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{{
		ScopeSpans: []*v1_trace.ScopeSpans{{Scope: scope("tailsampling.cached_decision"), Spans: []*v1_trace.Span{span(2)}}},
	}}}

	lt := livetraces.New(func(b []byte) uint64 { return uint64(len(b)) }, 0, 0, "test-tenant")
	for _, tr := range []*tempopb.Trace{original, resent} {
		b, err := tr.Marshal()
		require.NoError(t, err)
		lt.Push(traceID, b, 0)
	}

	iter := newLiveTracesIter(lt)
	ctx := context.Background()

	_, resultTr, err := iter.Next(ctx)
	require.NoError(t, err)
	_, _, err = iter.Next(ctx)
	require.NoError(t, err)

	require.Len(t, resultTr.ResourceSpans, 1)
	require.Len(t, resultTr.ResourceSpans[0].ScopeSpans, 1)
	require.Empty(t, resultTr.ResourceSpans[0].ScopeSpans[0].Scope.Attributes)
	require.Len(t, resultTr.ResourceSpans[0].ScopeSpans[0].Spans, 2)
	n, err := iter.DedupedSpans()
	require.NoError(t, err)
	require.Equal(t, uint32(1), n)
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
