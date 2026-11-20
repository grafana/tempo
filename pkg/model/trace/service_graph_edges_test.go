package trace

import (
	"testing"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

func edgeTestSpan(service string, id byte) *tracev1.ResourceSpans {
	return &tracev1.ResourceSpans{
		Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", service)}},
		ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{SpanId: []byte{id}, Kind: tracev1.Span_SPAN_KIND_INTERNAL}}}},
	}
}

func TestServiceGraphEdgesCombineFragments(t *testing.T) {
	child := &tempopb.Trace{
		ResourceSpans:     []*tracev1.ResourceSpans{edgeTestSpan("gateway", 2)},
		ServiceGraphEdges: []*tempopb.ServiceGraphEdge{{ParentSpanId: []byte{1}, ChildSpanId: []byte{2}, ChildService: "gateway", ChildKind: 1, ParentMissing: true}},
	}
	parent := &tempopb.Trace{ResourceSpans: []*tracev1.ResourceSpans{edgeTestSpan("A", 1)}}
	c := NewCombiner(0, false)
	_, err := c.Consume(child)
	require.NoError(t, err)
	_, err = c.Consume(parent)
	require.NoError(t, err)
	// Replay a durable record from another block without duplicating the edge.
	_, err = c.Consume(&tempopb.Trace{ServiceGraphEdges: []*tempopb.ServiceGraphEdge{{ParentSpanId: []byte{1}, ChildSpanId: []byte{2}, ChildKind: 1, ParentMissing: true}}})
	require.NoError(t, err)
	result, _ := c.Result()
	require.Len(t, result.ServiceGraphEdges, 1)
	require.Empty(t, result.ServiceGraphEdges[0].ParentService, "query combination must not expand optional labels beyond the size budget")
	require.Equal(t, "gateway", result.ServiceGraphEdges[0].ChildService)
	require.False(t, result.ServiceGraphEdges[0].ParentMissing)
	require.Equal(t, int32(tracev1.Span_SPAN_KIND_INTERNAL), result.ServiceGraphEdges[0].ParentKind)
}

func TestServiceGraphEdgesRespectVisibleSpans(t *testing.T) {
	for _, tc := range []struct {
		name          string
		parent, child bool
		count         int
	}{
		{"both visible", true, true, 1}, {"parent hidden", false, true, 1}, {"child hidden", true, false, 0}, {"all hidden", false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := &tempopb.Trace{ServiceGraphEdges: []*tempopb.ServiceGraphEdge{{ParentSpanId: []byte{1}, ChildSpanId: []byte{2}, ParentService: "secret-parent", ChildService: "secret-child", ChildKind: 1}}}
			if tc.parent {
				tr.ResourceSpans = append(tr.ResourceSpans, edgeTestSpan("visible-parent", 1))
			}
			if tc.child {
				tr.ResourceSpans = append(tr.ResourceSpans, edgeTestSpan("visible-child", 2))
			}
			ReconcileServiceGraphEdges(tr)
			require.Len(t, tr.ServiceGraphEdges, tc.count)
			if tc.count == 0 {
				return
			}
			edge := tr.ServiceGraphEdges[0]
			require.Equal(t, "visible-child", edge.ChildService)
			require.Equal(t, !tc.parent, edge.ParentMissing)
			if tc.parent {
				require.Equal(t, "visible-parent", edge.ParentService)
			} else {
				require.Empty(t, edge.ParentService)
				require.Zero(t, edge.ParentKind)
			}
		})
	}
}

// Kind zero is a real span kind, not a missing-value sentinel. Every visible
// parent must supply its current kind, while a hidden parent must be scrubbed.
func TestServiceGraphEdgesResolveParentKind(t *testing.T) {
	for _, kind := range []tracev1.Span_SpanKind{tracev1.Span_SPAN_KIND_UNSPECIFIED, tracev1.Span_SPAN_KIND_INTERNAL, tracev1.Span_SPAN_KIND_SERVER} {
		t.Run(kind.String(), func(t *testing.T) {
			parent := edgeTestSpan("A", 1)
			parent.ScopeSpans[0].Spans[0].Kind = kind
			tr := &tempopb.Trace{ResourceSpans: []*tracev1.ResourceSpans{parent, edgeTestSpan("B", 2)}, ServiceGraphEdges: []*tempopb.ServiceGraphEdge{{ParentSpanId: []byte{1}, ChildSpanId: []byte{2}, ChildKind: 1, ParentMissing: true}}}
			ReconcileServiceGraphEdges(tr)
			require.False(t, tr.ServiceGraphEdges[0].ParentMissing)
			require.Equal(t, int32(kind), tr.ServiceGraphEdges[0].ParentKind)
			require.Empty(t, tr.ServiceGraphEdges[0].ParentService, "do not inflate optional labels during query combination")
			tr.ResourceSpans = tr.ResourceSpans[1:]
			ReconcileServiceGraphEdges(tr)
			require.True(t, tr.ServiceGraphEdges[0].ParentMissing)
			require.Zero(t, tr.ServiceGraphEdges[0].ParentKind)
		})
	}
}
