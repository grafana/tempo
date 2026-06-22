package spanpruning

import (
	"context"
	"hash/fnv"

	"github.com/gogo/protobuf/proto"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	spanpruningprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/tempopb"
)

// PruneTrace runs the upstream span pruning processor against trace and returns the result.
// The processor collapses similar leaf spans (grouped by name, kind, status, and configured
// attribute patterns) when a group exceeds MinSpansToAggregate, replacing them with a single
// summary span annotated with aggregation statistics.
func PruneTrace(cfg *spanpruningprocessor.Config, trace *tempopb.Trace) (*tempopb.Trace, error) {
	td, err := tempopbToTraces(trace)
	if err != nil {
		return nil, err
	}

	sink := &tracesSink{}
	p, err := newProcessor(cfg, sink)
	if err != nil {
		return nil, err
	}

	if err := p.ConsumeTraces(context.Background(), td); err != nil {
		return nil, err
	}

	if !sink.set {
		return trace, nil
	}
	return tracesToTempopb(sink.traces)
}

// SummaryOnlyTrace runs span pruning but keeps all original spans, appending synthetic summary
// spans alongside them. Each summary span carries the same aggregation statistics (span_count,
// duration min/max/avg, etc.) as normal pruning would produce, but the original spans are
// preserved. Summary spans receive a new SpanID to avoid collisions with existing spans.
func SummaryOnlyTrace(cfg *spanpruningprocessor.Config, trace *tempopb.Trace) (*tempopb.Trace, error) {
	pruned, err := PruneTrace(cfg, trace)
	if err != nil {
		return nil, err
	}

	// Collect summary spans from the pruned result, keyed by their parent span ID so we can
	// inject each one next to the ScopeSpans that contains its parent.
	type location struct{ rs, ss int }

	// Build index: spanID → location in the original trace.
	parentToLoc := make(map[string]location)
	for rsIdx, rs := range trace.ResourceSpans {
		for ssIdx, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				parentToLoc[string(span.SpanId)] = location{rsIdx, ssIdx}
			}
		}
	}

	// Extract summary spans from the pruned result and assign synthetic SpanIDs.
	byLoc := make(map[location][]*tracev1.Span)
	var orphans []*tracev1.Span
	for _, rs := range pruned.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if !isSummarySpan(span) {
					continue
				}
				s := proto.Clone(span).(*tracev1.Span)
				s.SpanId = summarySpanID(s.SpanId)
				if loc, ok := parentToLoc[string(s.ParentSpanId)]; ok {
					byLoc[loc] = append(byLoc[loc], s)
				} else {
					orphans = append(orphans, s)
				}
			}
		}
	}

	if len(byLoc) == 0 && len(orphans) == 0 {
		return trace, nil
	}

	// Build the result: a copy of the original trace with summary spans injected into the
	// same ScopeSpans as their parent span.
	out := &tempopb.Trace{}
	for rsIdx, rs := range trace.ResourceSpans {
		newRS := &tracev1.ResourceSpans{Resource: rs.Resource, SchemaUrl: rs.SchemaUrl}
		for ssIdx, ss := range rs.ScopeSpans {
			spans := make([]*tracev1.Span, len(ss.Spans))
			copy(spans, ss.Spans)
			spans = append(spans, byLoc[location{rsIdx, ssIdx}]...)
			newRS.ScopeSpans = append(newRS.ScopeSpans, &tracev1.ScopeSpans{
				Scope:     ss.Scope,
				SchemaUrl: ss.SchemaUrl,
				Spans:     spans,
			})
		}
		out.ResourceSpans = append(out.ResourceSpans, newRS)
	}
	if len(orphans) > 0 {
		out.ResourceSpans = append(out.ResourceSpans, &tracev1.ResourceSpans{
			ScopeSpans: []*tracev1.ScopeSpans{{
				Scope: &commonv1.InstrumentationScope{Name: "tempo.span-pruning"},
				Spans: orphans,
			}},
		})
	}

	return out, nil
}

// isSummarySpan returns true when the span has the aggregation.is_summary=true attribute set by
// the span pruning processor.
func isSummarySpan(span *tracev1.Span) bool {
	for _, kv := range span.Attributes {
		if kv.Key == "aggregation.is_summary" {
			return kv.Value != nil && kv.Value.GetBoolValue()
		}
	}
	return false
}

// summarySpanID derives a synthetic SpanID for a summary span so it does not collide with the
// original span whose ID it would otherwise share. The result is deterministic.
func summarySpanID(original []byte) []byte {
	h := fnv.New64a()
	h.Write(original)
	h.Write([]byte("summary"))
	return h.Sum(nil) // FNV-64a Sum returns exactly 8 bytes
}

func newProcessor(cfg *spanpruningprocessor.Config, sink *tracesSink) (interface {
	ConsumeTraces(context.Context, ptrace.Traces) error
}, error) {
	settings := processor.Settings{
		ID: component.MustNewID("spanpruning"),
		TelemetrySettings: component.TelemetrySettings{
			Logger:         zap.NewNop(),
			MeterProvider:  noopmetric.NewMeterProvider(),
			TracerProvider: nooptrace.NewTracerProvider(),
		},
	}
	return spanpruningprocessor.NewFactory().CreateTraces(context.Background(), settings, cfg, sink)
}

func tempopbToTraces(trace *tempopb.Trace) (ptrace.Traces, error) {
	b, err := trace.Marshal()
	if err != nil {
		return ptrace.Traces{}, err
	}
	return (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(b)
}

func tracesToTempopb(td ptrace.Traces) (*tempopb.Trace, error) {
	b, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(td)
	if err != nil {
		return nil, err
	}
	var result tempopb.Trace
	if err := result.Unmarshal(b); err != nil {
		return nil, err
	}
	return &result, nil
}

type tracesSink struct {
	traces ptrace.Traces
	set    bool
}

func (s *tracesSink) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	s.traces = td
	s.set = true
	return nil
}

func (s *tracesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
