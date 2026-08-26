// Package spanpruning collapses similar leaf spans in a trace into a single summary span.
//
// EXPERIMENTAL: this package is not yet a stable API; config, behavior, and the shape of the
// summary spans it produces may change in future releases.
package spanpruning

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/grafana/tempo/pkg/tempopb"
	spanpruningprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"
)

// Status classifies whether, and where, a trace was pruned, so callers can tell lossy
// pruning from best-effort pruning that leaves the original spans intact in storage.
type Status int64

const (
	// StatusNotPruned means the trace was not modified by pruning (e.g. nothing met the
	// aggregation threshold).
	StatusNotPruned Status = 0
	// StatusPrunedOnWrite means the trace already contained a pruning summary span when it
	// reached Tempo, i.e. an upstream collector pruned it before ingestion. This is lossy: the
	// original spans were never stored, so this is all Tempo has.
	StatusPrunedOnWrite Status = 1
	// StatusPrunedOnRead means Tempo pruned the trace while serving this query. The original,
	// unpruned spans remain in storage and can be retrieved by re-querying with pruning disabled.
	StatusPrunedOnRead Status = 2
)

// PruneTrace runs the upstream span pruning processor against trace and returns the result
// along with a Status describing whether/where pruning happened. The processor collapses
// similar leaf spans (grouped by name, kind, status, and configured attribute patterns) when a
// group exceeds MinSpansToAggregate, replacing them with a single summary span annotated with
// aggregation statistics.
//
// If trace already contains a summary span from a prior pruning pass (e.g. an upstream
// collector ran this same processor before the trace reached Tempo), pruning is skipped
// entirely: aggregating summary spans again would produce summaries-of-summaries and corrupt
// the stats they carry.
func PruneTrace(cfg *spanpruningprocessor.Config, trace *tempopb.Trace) (*tempopb.Trace, Status, error) {
	if alreadyPruned(cfg, trace) {
		return trace, StatusPrunedOnWrite, nil
	}

	ctx := context.Background()
	// TODO: this marshal/unmarshal round trip through ptrace is an opt-in-feature-only cost for
	// now; if span pruning becomes hot enough to matter, work on the otel trace format directly.
	td, err := tempopbToTraces(trace)
	if err != nil {
		return nil, StatusNotPruned, err
	}

	sink := &next{}
	p, err := newProcessor(cfg, sink)
	if err != nil {
		return nil, StatusNotPruned, err
	}

	defer func() { _ = p.Shutdown(ctx) }()

	if err := p.ConsumeTraces(ctx, td); err != nil {
		return nil, StatusNotPruned, err
	}

	result, err := tracesToTempopb(sink.traces)
	if err != nil {
		return nil, StatusNotPruned, err
	}

	status := StatusNotPruned
	if alreadyPruned(cfg, result) {
		status = StatusPrunedOnRead
	}
	return result, status, nil
}

// alreadyPruned reports whether trace already contains a summary span produced by a previous
// run of the span pruning processor, identified by its aggregation marker attribute (e.g.
// "aggregation.is_summary" for the default AggregationAttributePrefix).
func alreadyPruned(cfg *spanpruningprocessor.Config, trace *tempopb.Trace) bool {
	summaryKey := cfg.AggregationAttributePrefix + "is_summary"
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				for _, kv := range span.Attributes {
					if kv.Key == summaryKey && kv.Value.GetBoolValue() {
						return true
					}
				}
			}
		}
	}
	return false
}

// tracesProcessor is the subset of the OTel Collector traces processor interface this package
// drives directly: consuming traces and, if present, shutting down cleanly.
type tracesProcessor interface {
	component.Component
	ConsumeTraces(context.Context, ptrace.Traces) error
}

func newProcessor(cfg *spanpruningprocessor.Config, next *next) (tracesProcessor, error) {
	settings := processor.Settings{
		ID: component.MustNewID("spanpruning"),
		TelemetrySettings: component.TelemetrySettings{
			Logger:         zap.NewNop(),
			MeterProvider:  noopmetric.NewMeterProvider(),
			TracerProvider: nooptrace.NewTracerProvider(),
		},
	}
	return spanpruningprocessor.NewFactory().CreateTraces(context.Background(), settings, cfg, next)
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

type next struct {
	traces ptrace.Traces
}

func (s *next) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	s.traces = td
	return nil
}

func (s *next) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
