package tempo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/golang/protobuf/jsonpb"
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

const (
	AcceptHeaderKey         = "Accept"
	ProtobufTypeHeaderValue = "application/protobuf"
)

const (
	serviceSearchTag     = "root.service.name"
	operationSearchTag   = "root.name"
	minDurationSearchTag = "minDuration"
	maxDurationSearchTag = "maxDuration"
	numTracesSearchTag   = "limit"
)

type Backend struct {
	tempoBackend string
}

func New(cfg *Config) *Backend {
	return &Backend{
		tempoBackend: cfg.Backend,
	}
}

func (b *Backend) GetDependencies(ctx context.Context, endTs time.Time, lookback time.Duration) ([]jaeger.DependencyLink, error) {
	return nil, nil
}

func (b *Backend) GetTrace(ctx context.Context, traceID jaeger.TraceID) (*jaeger.Trace, error) {
	hexID := fmt.Sprintf("%016x%016x", traceID.High, traceID.Low)
	url := fmt.Sprintf("http://%s/api/traces/%s", b.tempoBackend, hexID)

	span, ctx := opentracing.StartSpanFromContext(ctx, "GetTrace")
	defer span.Finish()

	req, err := b.NewGetRequest(ctx, url, span)
	if err != nil {
		return nil, err
	}

	// Set content type to grpc
	req.Header.Set(AcceptHeaderKey, ProtobufTypeHeaderValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed get to tempo %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, jaeger_spanstore.ErrTraceNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from tempo: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", body)
	}

	otTrace := ot_pdata.NewTraces()
	err = otTrace.FromOtlpProtoBytes(body)
	if err != nil {
		return nil, fmt.Errorf("Error converting tempo response to Otlp: %w", err)
	}

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
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetOperations")
	defer span.Finish()

	return b.lookupTagValues(ctx, span, serviceSearchTag)
}

func (b *Backend) GetOperations(ctx context.Context, query jaeger_spanstore.OperationQueryParameters) ([]jaeger_spanstore.Operation, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetOperations")
	defer span.Finish()

	tagValues, err := b.lookupTagValues(ctx, span, operationSearchTag)
	if err != nil {
		return nil, err
	}

	var operations []jaeger_spanstore.Operation
	for _, value := range tagValues {
		operations = append(operations, jaeger_spanstore.Operation{
			Name:     value,
			SpanKind: "",
		})
	}

	return operations, nil

}

func (b *Backend) FindTraces(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]*jaeger.Trace, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "FindTraces")
	defer span.Finish()

	traceIDs, err := b.FindTraceIDs(ctx, query)
	if err != nil {
		return nil, err
	}

	// for every traceID, get the full trace
	var jaegerTraces []*jaeger.Trace
	for _, traceID := range traceIDs {
		trace, err := b.GetTrace(ctx, traceID)
		if err != nil {
			return nil, fmt.Errorf("could not get trace for traceID %v: %w", traceID, err)
		}

		jaegerTraces = append(jaegerTraces, trace)
	}

	return jaegerTraces, nil
}

func (b *Backend) FindTraceIDs(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]jaeger.TraceID, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "FindTraceIDs")
	defer span.Finish()

	url := url.URL{
		Scheme: "http",
		Host:   b.tempoBackend,
		Path:   "api/search",
	}
	urlQuery := url.Query()
	urlQuery.Set(serviceSearchTag, query.ServiceName)
	urlQuery.Set(operationSearchTag, query.OperationName)
	urlQuery.Set(minDurationSearchTag, query.DurationMin.String())
	urlQuery.Set(maxDurationSearchTag, query.DurationMax.String())
	urlQuery.Set(numTracesSearchTag, strconv.Itoa(query.NumTraces))
	for k, v := range query.Tags {
		urlQuery.Set(k, v)
	}
	url.RawQuery = urlQuery.Encode()

	req, err := b.NewGetRequest(ctx, url.String(), span)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed get to tempo %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected 200 OK, got %v", resp.StatusCode)
	}

	var searchResponse tempopb.SearchResponse

	unmarshaler := jsonpb.Unmarshaler{}
	err = unmarshaler.Unmarshal(resp.Body, &searchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response %w", err)
	}

	jaegerTraceIDs := make([]jaeger.TraceID, len(searchResponse.Traces))

	for i, traceMetadata := range searchResponse.Traces {
		jaegerTraceID, err := jaeger.TraceIDFromBytes(traceMetadata.TraceID)
		if err != nil {
			return nil, fmt.Errorf("could not convert traceID into jaeger traceID %w", err)
		}
		jaegerTraceIDs[i] = jaegerTraceID
	}

	return jaegerTraceIDs, nil
}

func (b *Backend) lookupTagValues(ctx context.Context, span opentracing.Span, tagName string) ([]string, error) {
	url := fmt.Sprintf("http://%s/api/search/tag/%s/values", b.tempoBackend, tagName)

	req, err := b.NewGetRequest(ctx, url, span)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed get to tempo %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected 200 OK, got %v", resp.StatusCode)
	}

	var searchLookupResponse tempopb.SearchTagValuesResponse

	unmarshaler := jsonpb.Unmarshaler{}
	err = unmarshaler.Unmarshal(resp.Body, &searchLookupResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response %w", err)
	}

	return searchLookupResponse.TagValues, nil
}

func (b *Backend) WriteSpan(ctx context.Context, span *jaeger.Span) error {
	return nil
}

func (b *Backend) NewGetRequest(ctx context.Context, url string, span opentracing.Span) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	return req, nil
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
