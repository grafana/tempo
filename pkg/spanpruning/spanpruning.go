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

// PruneTrace runs the upstream span pruning processor against trace and returns the result.
// The processor collapses similar leaf spans (grouped by name, kind, status, and configured
// attribute patterns) when a group exceeds MinSpansToAggregate, replacing them with a single
// summary span annotated with aggregation statistics.
func PruneTrace(cfg *spanpruningprocessor.Config, trace *tempopb.Trace) (*tempopb.Trace, error) {
	ctx := context.Background()
	// TODO: this marshal/unmarshal round trip through ptrace is an opt-in-feature-only cost for
	// now; if span pruning becomes hot enough to matter, work on the otel trace format directly.
	td, err := tempopbToTraces(trace)
	if err != nil {
		return nil, err
	}

	sink := &next{}
	p, err := newProcessor(cfg, sink)
	if err != nil {
		return nil, err
	}

	defer func() { _ = p.Shutdown(ctx) }()

	if err := p.ConsumeTraces(ctx, td); err != nil {
		return nil, err
	}

	return tracesToTempopb(sink.traces)
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
