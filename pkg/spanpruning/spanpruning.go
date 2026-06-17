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
	td, err := tempopbToTraces(trace)
	if err != nil {
		return nil, err
	}

	sink := &tracesSink{}
	settings := processor.Settings{
		ID: component.MustNewID("spanpruning"),
		TelemetrySettings: component.TelemetrySettings{
			Logger:         zap.NewNop(),
			MeterProvider:  noopmetric.NewMeterProvider(),
			TracerProvider: nooptrace.NewTracerProvider(),
		},
	}

	p, err := spanpruningprocessor.NewFactory().CreateTraces(context.Background(), settings, cfg, sink)
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
