package distributor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/grafana/tempo/pkg/tempopb"
)

// PushTraces implements shim.TracesPusher, it is called by the OpenTelemetry Collector receiver
func (d *Distributor) PushTraces(ctx context.Context, traces ptrace.Traces) (*tempopb.PushResponse, error) {
	return d.pushTracesInternal(ctx, pushRequestFromOTLPTrace(traces))
}

// PushTempopbTraces implements tempopb.Distributor, it is called by the gRPC server
func (d *Distributor) PushTempopbTraces(ctx context.Context, trace *tempopb.Trace) (*tempopb.PushResponse, error) {
	return d.pushTracesInternal(ctx, pushRequestFromTempopbTrace(trace))
}

// pushRequest bundles data needed to handle a request and allows to lazy-load expensive operations.
type pushRequest struct {
	spanCount    int
	size         int
	tempopbTrace func() (*tempopb.Trace, error)
	otlpTrace    func() (ptrace.Traces, error)
}

func pushRequestFromOTLPTrace(trace ptrace.Traces) pushRequest {
	return pushRequest{
		spanCount: trace.SpanCount(),
		size:      (&ptrace.ProtoMarshaler{}).TracesSize(trace),
		tempopbTrace: func() (*tempopb.Trace, error) {
			return tempopb.ConvertFromOTLP(trace)
		},
		otlpTrace: func() (ptrace.Traces, error) {
			return trace, nil
		},
	}
}

func pushRequestFromTempopbTrace(trace *tempopb.Trace) pushRequest {
	return pushRequest{
		spanCount: trace.SpanCount(),
		size:      trace.Size(),
		tempopbTrace: func() (*tempopb.Trace, error) {
			return trace, nil
		},
		otlpTrace: func() (ptrace.Traces, error) {
			return trace.ConvertToOTLP()
		},
	}
}
