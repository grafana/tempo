package trace

import (
	"github.com/grafana/tempo/v3/pkg/tempopb"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

type serviceGraphEdgeKey struct {
	parent, child string
	kind          int32
}

func edgeKey(edge *tempopb.ServiceGraphEdge) serviceGraphEdgeKey {
	return serviceGraphEdgeKey{string(edge.ParentSpanId), string(edge.ChildSpanId), edge.ChildKind}
}

// ReconcileServiceGraphEdges resolves records against the spans visible in this
// response. It also removes edge metadata for filtered children and scrubs
// parent metadata when redaction, filtering, or pruning removes a parent.
// It never creates collapsed relationships or additional records.
func ReconcileServiceGraphEdges(trace *tempopb.Trace) {
	if trace == nil || len(trace.ServiceGraphEdges) == 0 {
		return
	}
	type endpoint struct {
		service string
		kind    int32
	}
	type spanKey struct {
		id   string
		kind int32
	}
	spans := make(map[spanKey]endpoint)
	parents := make(map[string]endpoint)
	for _, rs := range trace.ResourceSpans {
		if rs == nil {
			continue
		}
		service := ""
		if rs.Resource != nil {
			for _, attr := range rs.Resource.Attributes {
				if attr != nil && attr.Key == "service.name" {
					service = attr.Value.GetStringValue()
					break
				}
			}
		}
		for _, ss := range rs.ScopeSpans {
			if ss == nil {
				continue
			}
			for _, span := range ss.Spans {
				if span != nil {
					node := endpoint{service, int32(span.Kind)}
					spans[spanKey{string(span.SpanId), int32(span.Kind)}] = node
					existing, exists := parents[string(span.SpanId)]
					if !exists || (existing.kind == int32(tracev1.Span_SPAN_KIND_CLIENT) && span.Kind == tracev1.Span_SPAN_KIND_SERVER) {
						parents[string(span.SpanId)] = node
					}
				}
			}
		}
	}
	output := trace.ServiceGraphEdges[:0]
	for _, edge := range trace.ServiceGraphEdges {
		if edge == nil {
			continue
		}
		child, exists := spans[spanKey{string(edge.ChildSpanId), edge.ChildKind}]
		if !exists {
			continue
		}
		parent, exists := parents[string(edge.ParentSpanId)]
		edge.ChildService = child.service
		edge.ChildKind = child.kind
		// Preserve an absent optional label until compaction resolves it on disk.
		// Expanding a long parent name into every unresolved edge during a query
		// could bypass the response-size limit. IDs and returned spans still join it.
		if !exists || edge.ParentService != "" {
			edge.ParentService = parent.service
		}
		// Span kind is a bounded enum, so resolving it cannot inflate a label.
		// Zero is SPAN_KIND_UNSPECIFIED and must not stand in for missing data.
		edge.ParentKind = parent.kind
		edge.ParentMissing = !exists
		output = append(output, edge)
	}
	clear(trace.ServiceGraphEdges[len(output):])
	trace.ServiceGraphEdges = output
}

func (c *Combiner) consumeServiceGraphEdges(trace *tempopb.Trace) {
	if len(trace.ServiceGraphEdges) == 0 {
		return
	}
	if c.edgeKeys == nil {
		c.edgeKeys = make(map[serviceGraphEdgeKey]*tempopb.ServiceGraphEdge)
		for _, edge := range c.result.ServiceGraphEdges {
			if edge != nil {
				c.edgeKeys[edgeKey(edge)] = edge
			}
		}
	}
	for _, edge := range trace.ServiceGraphEdges {
		if edge == nil {
			continue
		}
		key := edgeKey(edge)
		if existing, found := c.edgeKeys[key]; found {
			if existing.ParentMissing && !edge.ParentMissing {
				*existing = *edge
			}
			continue
		}
		c.edgeKeys[key] = edge
		c.result.ServiceGraphEdges = append(c.result.ServiceGraphEdges, edge)
	}
}
