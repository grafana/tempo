package tempo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc/metadata"

	jaeger "github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	jaeger_spanstore "github.com/jaegertracing/jaeger/storage/spanstore"

	ot_jaeger "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
	"go.opentelemetry.io/collector/model/otlp"
)

const (
	AcceptHeaderKey         = "Accept"
	ProtobufTypeHeaderValue = "application/protobuf"
)

const (
	serviceSearchTag     = "service.name"
	operationSearchTag   = "name"
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
	url := fmt.Sprintf("http://%s/api/traces/%s", b.tempoBackend, traceID)

	span, ctx := opentracing.StartSpanFromContext(ctx, "tempo-query.GetTrace")
	defer span.Finish()

	req, err := b.newGetRequest(ctx, url, span)
	if err != nil {
		return nil, err
	}

	// Set content type to GRPC
	req.Header.Set(AcceptHeaderKey, ProtobufTypeHeaderValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed GET to tempo %w", err)
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

	otTrace, err := otlp.NewProtobufTracesUnmarshaler().UnmarshalTraces(body)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling body to otlp trace %v: %w", traceID, err)
	}

	jaegerBatches, err := ot_jaeger.ProtoFromTraces(otTrace)
	if err != nil {
		return nil, fmt.Errorf("error translating to jaegerBatches %v: %w", traceID, err)
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "tempo-query.GetOperations")
	defer span.Finish()

	return b.lookupTagValues(ctx, span, serviceSearchTag)
}

func (b *Backend) GetOperations(ctx context.Context, query jaeger_spanstore.OperationQueryParameters) ([]jaeger_spanstore.Operation, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "tempo-query.GetOperations")
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "tempo-query.FindTraces")
	defer span.Finish()

	traceIDs, err := b.FindTraceIDs(ctx, query)
	if err != nil {
		return nil, err
	}

	span.LogFields(ot_log.String("msg", fmt.Sprintf("Found %d trace IDs", len(traceIDs))))

	// for every traceID, get the full trace
	var jaegerTraces []*jaeger.Trace
	for _, traceID := range traceIDs {
		trace, err := b.GetTrace(ctx, traceID)
		if err != nil {
			// TODO this seems to be an internal inconsistency error, ignore so we can still show the rest
			span.LogFields(ot_log.Error(fmt.Errorf("could not get trace for traceID %v: %w", traceID, err)))
			continue
		}

		jaegerTraces = append(jaegerTraces, trace)
	}

	span.LogFields(ot_log.String("msg", fmt.Sprintf("Returning %d traces", len(jaegerTraces))))

	return jaegerTraces, nil
}

func (b *Backend) FindTraceIDs(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]jaeger.TraceID, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "tempo-query.FindTraceIDs")
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

	req, err := b.newGetRequest(ctx, url.String(), span)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed GET to tempo %w", err)
	}
	defer resp.Body.Close()

	// if search endpoint returns 404, search is most likely not enabled
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response from Tempo: got %s", resp.Status)
		}
		return nil, fmt.Errorf("%s", body)
	}

	var searchResponse tempopb.SearchResponse
	err = jsonpb.Unmarshal(resp.Body, &searchResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Tempo response: %w", err)
	}

	jaegerTraceIDs := make([]jaeger.TraceID, len(searchResponse.Traces))

	for i, traceMetadata := range searchResponse.Traces {
		jaegerTraceID, err := jaeger.TraceIDFromString(traceMetadata.TraceID)
		if err != nil {
			return nil, fmt.Errorf("could not convert traceID into Jaeger's traceID %w", err)
		}
		jaegerTraceIDs[i] = jaegerTraceID
	}

	return jaegerTraceIDs, nil
}

func (b *Backend) lookupTagValues(ctx context.Context, span opentracing.Span, tagName string) ([]string, error) {
	url := fmt.Sprintf("http://%s/api/search/tag/%s/values", b.tempoBackend, tagName)

	req, err := b.newGetRequest(ctx, url, span)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed GET to tempo %w", err)
	}
	defer resp.Body.Close()

	// if search endpoint returns 404, search is most likely not enabled
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response from Tempo: got %s", resp.Status)
		}
		return nil, fmt.Errorf("%s", body)
	}

	var searchLookupResponse tempopb.SearchTagValuesResponse
	err = jsonpb.Unmarshal(resp.Body, &searchLookupResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Tempo response: %w", err)
	}

	return searchLookupResponse.TagValues, nil
}

func (b *Backend) WriteSpan(ctx context.Context, span *jaeger.Span) error {
	return nil
}

func (b *Backend) newGetRequest(ctx context.Context, url string, span opentracing.Span) (*http.Request, error) {
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
		values := md.Get(shared.BearerTokenKey)
		if len(values) > 0 {
			return values[0], true
		}
	}
	return "", false
}
