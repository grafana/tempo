package tempo

import (
	"context"
	"fmt"
	"github.com/grafana/tempo/pkg/util"
	"google.golang.org/grpc"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"

	jaeger "github.com/jaegertracing/jaeger/model"
	jaeger_spanstore "github.com/jaegertracing/jaeger/storage/spanstore"

	ot_pdata "go.opentelemetry.io/collector/consumer/pdata"
	ot_jaeger "go.opentelemetry.io/collector/translator/trace/jaeger"
)

type Backend struct {
	client tempopb.QuerierClient
}

func New(cfg *Config) (*Backend, error) {
	conn, err := grpc.Dial(cfg.Backend, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client := tempopb.NewQuerierClient(conn)

	return &Backend{
		client: client,
	}, nil
}

func (b *Backend) GetDependencies(endTs time.Time, lookback time.Duration) ([]jaeger.DependencyLink, error) {
	return nil, nil
}
func (b *Backend) GetTrace(ctx context.Context, traceID jaeger.TraceID) (*jaeger.Trace, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "GetTrace")
	defer span.Finish()

	if tracer := opentracing.GlobalTracer(); tracer != nil {
		ctx = derivedCtx
	}

	// create TraceByIDRequest
	hexID := fmt.Sprintf("%016x%016x", traceID.High, traceID.Low)
	idBytes, _ := util.HexStringToTraceID(hexID)
	req := &tempopb.TraceByIDRequest{
		TraceID: idBytes,
	}

	// Call querier
	resp, err := b.client.FindTraceByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error reading response from tempo: %w", err)
	}

	// convert from otlp to jaeger format
	span.LogFields(ot_log.String("msg", "otlp to Jaeger"))
	otTrace := ot_pdata.TracesFromOtlp(resp.Trace.Batches)
	jaegerBatches, err := ot_jaeger.InternalTracesToJaegerProto(otTrace)

	if err != nil {
		return nil, fmt.Errorf("error translating to jaegerBatches %v: %w", hexID, err)
	}

	jaegerTrace := &jaeger.Trace{
		Spans:      []*jaeger.Span{},
		ProcessMap: []jaeger.Trace_ProcessMapping{},
	}

	span.LogFields(ot_log.String("msg", "build process map"))
	// otel proto conversion doesn't set jaeger processes
	for _, batch := range jaegerBatches {
		for _, s := range batch.Spans {
			s.Process = batch.Process
		}

		jaegerTrace.Spans = append(jaegerTrace.Spans, batch.Spans...)
		jaegerTrace.ProcessMap = append(jaegerTrace.ProcessMap, jaeger.Trace_ProcessMapping{
			Process:   *batch.Process,
			ProcessID: batch.Process.ServiceName,
		})
	}

	return jaegerTrace, nil
}

func (b *Backend) GetServices(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (b *Backend) GetOperations(ctx context.Context, query jaeger_spanstore.OperationQueryParameters) ([]jaeger_spanstore.Operation, error) {
	return nil, nil
}
func (b *Backend) FindTraces(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]*jaeger.Trace, error) {
	return nil, nil
}
func (b *Backend) FindTraceIDs(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]jaeger.TraceID, error) {
	return nil, nil
}
func (b *Backend) WriteSpan(span *jaeger.Span) error {
	return nil
}
