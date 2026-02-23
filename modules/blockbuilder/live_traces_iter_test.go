package blockbuilder

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

func TestLiveTracesIter_DedupSpans(t *testing.T) {
	// Build a trace with duplicate spans
	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	span1 := &v1.Span{SpanId: spanID, Kind: v1.Span_SPAN_KIND_SERVER}
	span2 := &v1.Span{SpanId: spanID, Kind: v1.Span_SPAN_KIND_SERVER} // duplicate
	uniqueSpanID := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	span3 := &v1.Span{SpanId: uniqueSpanID, Kind: v1.Span_SPAN_KIND_CLIENT}

	tr := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			{
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{span1, span2, span3},
					},
				},
			},
		},
	}

	// Marshal the trace
	trBytes, err := tr.Marshal()
	require.NoError(t, err)

	traceID := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}

	lt := livetraces.New(func(b []byte) uint64 { return uint64(len(b)) }, 0, 0, "test-tenant")
	ok := lt.Push(traceID, trBytes, 0)
	require.True(t, ok)

	iter := newLiveTracesIter(lt)
	ctx := context.Background()

	// Exhaust the iterator
	id, resultTr, err := iter.Next(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	require.NotNil(t, resultTr)

	// Should have 2 unique spans, not 3
	totalSpans := 0
	for _, rs := range resultTr.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			totalSpans += len(ss.Spans)
		}
	}
	require.Equal(t, 2, totalSpans)

	// Next call should return nil (exhausted)
	id2, tr2, err2 := iter.Next(ctx)
	require.NoError(t, err2)
	require.Nil(t, id2)
	require.Nil(t, tr2)

	// Check deduped count
	require.Equal(t, 1, iter.DedupedSpans())
}
