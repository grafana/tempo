package tempo

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc/metadata"

	jaeger "github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	jaeger_spanstore "github.com/jaegertracing/jaeger/storage/spanstore"

	ot_pdata "go.opentelemetry.io/collector/consumer/pdata"
	ot_jaeger "go.opentelemetry.io/collector/translator/trace/jaeger"
)

type Backend struct {
	tempoEndpoint string
}

func New(cfg *Config) *Backend {
	return &Backend{
		tempoEndpoint: "http://" + cfg.Backend + "/api/traces/",
	}
}

func (b *Backend) GetDependencies(endTs time.Time, lookback time.Duration) ([]jaeger.DependencyLink, error) {
	return nil, nil
}
func (b *Backend) GetTrace(ctx context.Context, traceID jaeger.TraceID) (*jaeger.Trace, error) {
	hexID := fmt.Sprintf("%016x%016x", traceID.High, traceID.Low)

	span, _ := opentracing.StartSpanFromContext(ctx, "GetTrace")
	defer span.Finish()

	req, err := http.NewRequestWithContext(ctx, "GET", b.tempoEndpoint+hexID, nil)
	if err != nil {
		return nil, err
	}

	if tracer := opentracing.GlobalTracer(); tracer != nil {
		// this is not really loggable or anything we can react to.  just ignoring this error
		_ = tracer.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
	}

	// currently Jaeger Query will only propagate bearer token to the grpc backend and no other headers
	// so we are going to extract the tenant id from the header, if it exists and use it
	tenantID, found := extractBearerToken(ctx)
	if found {
		req.Header.Set(user.OrgIDHeaderName, tenantID)
	}

	// Set content type to grpc
	req.Header.Set("Accepts", "application/protobuf")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed get to tempo %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, jaeger_spanstore.ErrTraceNotFound
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from tempo: %w", err)
	}
	out := &tempopb.Trace{}
	err = proto.Unmarshal(body, out)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace proto, err: %w. Tempo response body: %s", err, string(body))
	}

	span.LogFields(ot_log.String("msg", "otlp to Jaeger"))
	otTrace := ot_pdata.TracesFromOtlp(out.Batches)
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

func extractBearerToken(ctx context.Context) (string, bool) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(spanstore.BearerTokenKey)
		if len(values) > 0 {
			return values[0], true
		}
	}
	return "", false
}
