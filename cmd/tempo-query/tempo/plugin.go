package tempo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	tlsCfg "github.com/grafana/dskit/crypto/tls"
	"github.com/grafana/dskit/user"
	jaeger "github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	otlptracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	storagev2 "github.com/grafana/tempo/pkg/jaegerpb/storage/v2"
	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	AcceptHeaderKey         = "Accept"
	ProtobufTypeHeaderValue = "application/protobuf"
)

const (
	serviceSearchTag   = "service.name"
	operationSearchTag = "name"
	startTimeMaxTag    = "end"
	startTimeMinTag    = "start"
	numTracesSearchTag = "limit"
)

var tlsVersions = map[string]uint16{
	"VersionTLS10": tls.VersionTLS10,
	"VersionTLS11": tls.VersionTLS11,
	"VersionTLS12": tls.VersionTLS12,
	"VersionTLS13": tls.VersionTLS13,
}

var tracer = otel.Tracer("cmd/tempo-query/tempo")

// ErrTraceNotFound is returned by Reader's GetTrace if no data is found for given trace ID.
var ErrTraceNotFound = errors.New("trace not found")

type Backend struct {
	logger                       *zap.Logger
	tempoBackend                 string
	tlsEnabled                   bool
	tls                          tlsCfg.ClientConfig
	httpClient                   *http.Client
	tenantHeaderKey              string
	QueryServicesDuration        *time.Duration
	findTracesConcurrentRequests int

	storagev2.UnimplementedTraceReaderServer
	storagev2.UnimplementedDependencyReaderServer
}

func New(logger *zap.Logger, cfg *Config) (*Backend, error) {
	_, span := tracer.Start(context.Background(), "tempo.New")
	defer span.End()
	httpClient, err := createHTTPClient(cfg)
	if err != nil {
		return nil, err
	}

	var queryServiceDuration *time.Duration
	if cfg.QueryServicesDuration != "" {
		queryDuration, err := time.ParseDuration(cfg.QueryServicesDuration)
		if err != nil {
			return nil, err
		}
		queryServiceDuration = &queryDuration

	}

	return &Backend{
		logger:                       logger,
		tempoBackend:                 cfg.Backend,
		tlsEnabled:                   cfg.TLSEnabled,
		tls:                          cfg.TLS,
		httpClient:                   httpClient,
		tenantHeaderKey:              cfg.TenantHeaderKey,
		QueryServicesDuration:        queryServiceDuration,
		findTracesConcurrentRequests: cfg.FindTracesConcurrentRequests,
	}, nil
}

func createHTTPClient(cfg *Config) (*http.Client, error) {
	if !cfg.TLSEnabled {
		return otelhttp.DefaultClient, nil
	}
	config := &tls.Config{
		InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		ServerName:         cfg.TLS.ServerName,
	}

	// read ca certificates
	if cfg.TLS.CAPath != "" {
		caCert, err := os.ReadFile(cfg.TLS.CAPath)
		if err != nil {
			return nil, fmt.Errorf("error opening %s CA", cfg.TLS.CAPath)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		config.RootCAs = caCertPool
	}
	// read client certificate
	if cfg.TLS.CertPath != "" || cfg.TLS.KeyPath != "" {
		clientCert, err := tls.LoadX509KeyPair(cfg.TLS.CertPath, cfg.TLS.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("error opening %s , %s cert", cfg.TLS.CertPath, cfg.TLS.KeyPath)
		}
		config.Certificates = []tls.Certificate{clientCert}
	}

	if cfg.TLS.MinVersion != "" {
		minVersion, ok := tlsVersions[cfg.TLS.MinVersion]
		if !ok {
			return nil, fmt.Errorf("unknown minimum TLS version: %q", cfg.TLS.MinVersion)
		}
		config.MinVersion = minVersion
	}

	if cfg.TLS.CipherSuites != "" {
		cleanedCipherSuiteNames := strings.ReplaceAll(cfg.TLS.CipherSuites, " ", "")
		cipherSuitesNames := strings.Split(cleanedCipherSuiteNames, ",")
		cipherSuites, err := mapCipherNamesToIDs(cipherSuitesNames)
		if err != nil {
			return nil, err
		}
		config.CipherSuites = cipherSuites
	}
	transport := &http.Transport{TLSClientConfig: config}
	return &http.Client{Transport: otelhttp.NewTransport(transport)}, nil
}

func mapCipherNamesToIDs(cipherSuiteNames []string) ([]uint16, error) {
	var cipherSuites []uint16
	allCipherSuites := tlsCipherSuites()

	for _, name := range cipherSuiteNames {
		id, ok := allCipherSuites[name]
		if !ok {
			return nil, fmt.Errorf("unsupported cipher suite: %q", name)
		}
		cipherSuites = append(cipherSuites, id)
	}

	return cipherSuites, nil
}

func tlsCipherSuites() map[string]uint16 {
	cipherSuites := map[string]uint16{}

	for _, suite := range tls.CipherSuites() {
		cipherSuites[suite.Name] = suite.ID
	}
	for _, suite := range tls.InsecureCipherSuites() {
		cipherSuites[suite.Name] = suite.ID
	}

	return cipherSuites
}

func (b *Backend) GetDependencies(context.Context, *storagev2.GetDependenciesRequest) (*storagev2.GetDependenciesResponse, error) {
	return &storagev2.GetDependenciesResponse{}, nil
}

func (b *Backend) apiSchema() string {
	if b.tlsEnabled {
		return "https"
	}
	return "http"
}

func (b *Backend) getTrace(ctx context.Context, traceID string, start, end *types.Timestamp) (*tempopb.TraceByIDResponse, error) {
	ctx, span := tracer.Start(ctx, "tempo-query.getTrace")
	defer span.End()
	endpoint := fmt.Sprintf("%s://%s/api/v2/traces/%s", b.apiSchema(), b.tempoBackend, traceID)
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing Tempo URL: %w", err)
	}

	values := url.Values{}
	if start != nil && start.Seconds > 0 {
		values.Set("start", fmt.Sprintf("%d", start.Seconds))
	}
	if end != nil && end.Seconds > 0 {
		values.Set("end", fmt.Sprintf("%d", end.Seconds))
	}
	u.RawQuery = values.Encode()

	req, err := b.newGetRequest(ctx, u.String())
	if err != nil {
		return nil, err
	}

	// Set content type to GRPC
	req.Header.Set(AcceptHeaderKey, ProtobufTypeHeaderValue)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed GET to tempo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrTraceNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from tempo: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", body)
	}

	var data = tempopb.TraceByIDResponse{}
	if err := proto.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("error unmarshaling Tempo response: %w", err)
	}
	return &data, nil
}

func (b *Backend) calculateTimeRange() (int64, int64) {
	now := time.Now()
	start := now.Add(*b.QueryServicesDuration * -1)
	return start.Unix(), now.Unix()
}

func (b *Backend) GetServices(ctx context.Context, _ *storagev2.GetServicesRequest) (*storagev2.GetServicesResponse, error) {
	ctx, span := tracer.Start(ctx, "tempo-query.GetServices")
	defer span.End()

	services, err := b.lookupTagValues(ctx, "resource."+serviceSearchTag, "")
	if err != nil {
		return nil, err
	}
	return &storagev2.GetServicesResponse{Services: services}, nil
}

func (b *Backend) GetOperations(ctx context.Context, r *storagev2.GetOperationsRequest) (*storagev2.GetOperationsResponse, error) {
	ctx, span := tracer.Start(ctx, "tempo-query.GetOperations")
	defer span.End()

	tagValues, err := b.lookupTagValues(ctx, operationSearchTag,
		fmt.Sprintf("{resource.service.name=%q}", strings.Replace(r.Service, "\"", "\\\"", -1)),
	)
	if err != nil {
		b.logger.Error("failed to lookup tag values", zap.Error(err))
		return nil, err
	}

	var operations []*storagev2.Operation
	for _, value := range tagValues {
		operations = append(operations, &storagev2.Operation{
			Name:     value,
			SpanKind: "",
		})
	}

	return &storagev2.GetOperationsResponse{
		Operations: operations,
	}, nil
}

type job struct {
	ctx     context.Context
	traceID jaeger.TraceID
}

type jobResult struct {
	traceID string
	trace   *otlptracev1.TracesData
	err     error
}

func worker(b *Backend, jobs <-chan job, results chan<- jobResult, start, end *types.Timestamp) {
	for job := range jobs {
		td, err := b.getTrace(job.ctx, job.traceID.String(), start, end)
		results <- jobResult{
			traceID: job.traceID.String(),
			trace:   getOtlpTraceData(td),
			err:     err,
		}
	}
}

func stringValue(s string) *storagev2.AnyValue {
	return &storagev2.AnyValue{
		Value: &storagev2.AnyValue_StringValue{
			StringValue: s,
		},
	}
}

func (b *Backend) FindTraces(r *storagev2.FindTracesRequest, s storagev2.TraceReader_FindTracesServer) error {
	ctx, span := tracer.Start(s.Context(), "tempo-query.FindTraces")
	defer span.End()

	endpoint := url.URL{
		Scheme: b.apiSchema(),
		Host:   b.tempoBackend,
		Path:   "api/search",
	}
	urlQuery := endpoint.Query()
	urlQuery.Set(numTracesSearchTag, strconv.Itoa(int(r.Query.SearchDepth)))

	if r.Query.StartTimeMin != nil {
		urlQuery.Set(startTimeMinTag, fmt.Sprintf("%d", r.Query.StartTimeMin.Seconds))
	}
	if r.Query.StartTimeMax != nil {
		urlQuery.Set(startTimeMaxTag, fmt.Sprintf("%d", r.Query.StartTimeMax.Seconds))
	}

	tags := make(map[string]*storagev2.AnyValue)
	if r.Query.ServiceName != "" {
		// I don't think it's possible to _not_ have a ServiceName - but let's cover this case as well.
		tags["resource."+serviceSearchTag] = stringValue(r.Query.ServiceName)
	}
	if r.Query.OperationName != "" {
		tags[operationSearchTag] = stringValue(r.Query.OperationName)
	}

	for _, v := range r.Query.Attributes {
		switch v.Key {
		case "error":
			// Handle error=true / error=false which is special in Jaeger
			if strings.EqualFold(v.Value.GetStringValue(), "true") {
				tags["status"] = stringValue("error")
			} else if strings.EqualFold(v.Value.GetStringValue(), "false") {
				tags["status"] = stringValue("ok")
			} else {
				tags["span."+v.Key] = v.Value
			}
		default:
			tags["span."+v.Key] = v.Value
		}
	}

	tql, err := BuildTQL(tags)
	if err != nil {
		return fmt.Errorf("failed to build TQL: %w", err)
	}

	tqlStringQ := fmt.Sprintf("{ %s }", ConditionsToString(tql, r.Query.DurationMin, r.Query.DurationMax))
	urlQuery.Set("q", tqlStringQ)
	b.logger.Debug("Tempo Query", zap.String("query", fmt.Sprintf("%v", urlQuery)))
	endpoint.RawQuery = urlQuery.Encode()

	req, err := b.newGetRequest(ctx, endpoint.String())
	if err != nil {
		return err
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed GET to tempo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		b.logger.Info("Tempo search endpoint not found (404), perhaps search is not enabled.")
		return s.Send(&otlptracev1.TracesData{ResourceSpans: make([]*otlptracev1.ResourceSpans, 0)})
	}

	if resp.StatusCode != http.StatusOK {
		body, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			return fmt.Errorf("error reading response body from Tempo: %w (original status: %s)", errRead, resp.Status)
		}
		return fmt.Errorf("tempo server returned error: %s - %s", resp.Status, string(body))
	}

	var searchResponse tempopb.SearchResponse
	err = jsonpb.Unmarshal(resp.Body, &searchResponse)
	if err != nil {
		return fmt.Errorf("error unmarshaling Tempo search response: %w", err)
	}

	if len(searchResponse.Traces) == 0 {
		b.logger.Info("No traces found by initial search.")
		return nil
	}

	jaegerTraceIDs := make([]jaeger.TraceID, 0, len(searchResponse.Traces))
	for _, traceMetadata := range searchResponse.Traces {
		jaegerTraceID, errConv := jaeger.TraceIDFromString(traceMetadata.TraceID)
		if errConv != nil {
			b.logger.Warn("Could not convert traceID from search result into Jaeger's traceID", zap.Error(errConv), zap.String("traceID", traceMetadata.TraceID))
			continue
		}
		jaegerTraceIDs = append(jaegerTraceIDs, jaegerTraceID)
	}

	if len(jaegerTraceIDs) == 0 {
		b.logger.Info("No valid traceIDs after parsing search results.")
		return s.Send(&otlptracev1.TracesData{ResourceSpans: make([]*otlptracev1.ResourceSpans, 0)})
	}

	numJobs := len(jaegerTraceIDs)
	jobs := make(chan job, numJobs)
	results := make(chan jobResult, numJobs)

	numWorkers := b.findTracesConcurrentRequests
	if numWorkers <= 0 {
		numWorkers = 4 // Default
	}
	if numWorkers > numJobs {
		numWorkers = numJobs
	}

	b.logger.Debug("Starting workers to fetch traces", zap.Int("num_workers", numWorkers), zap.Int("num_traces_to_fetch", numJobs))
	for i := 0; i < numWorkers; i++ {
		go worker(b, jobs, results, r.Query.StartTimeMin, r.Query.StartTimeMax)
	}

	for _, traceID := range jaegerTraceIDs {
		jobs <- job{ctx: ctx, traceID: traceID}
	}
	close(jobs)

	var td []*otlptracev1.TracesData
	for i := 0; i < numJobs; i++ {
		result := <-results
		if result.err != nil {
			if errors.Is(result.err, ErrTraceNotFound) {
				b.logger.Debug("Trace not found by worker", zap.String("traceID", result.traceID))
			} else {
				b.logger.Warn("Failed to fetch trace via worker", zap.String("traceID", result.traceID), zap.Error(result.err))
			}
			continue
		}
		if result.trace != nil {
			td = append(td, result.trace)
		}
	}

	if len(td) == 0 {
		b.logger.Info("No traces were successfully fetched after search.", zap.Int("search_results_count", len(searchResponse.Traces)))
		return s.Send(&otlptracev1.TracesData{ResourceSpans: make([]*otlptracev1.ResourceSpans, 0)})
	}
	b.logger.Info("Successfully fetched traces, preparing to send.", zap.Int("num_traces_to_send", len(td)))

	for _, traceData := range td {
		if traceData == nil || len(traceData.ResourceSpans) == 0 {
			b.logger.Debug("Skipping nil or empty traceData during send.")
			continue
		}
		// Each traceData is an *otlptracev1.TracesData, send it directly
		err = s.Send(traceData)
		if err != nil {
			b.logger.Error("Failed to send OTLP trace data", zap.Error(err))
			return fmt.Errorf("failed to send OTLP trace data: %w", err)
		}
	}

	b.logger.Info("Successfully sent all OTLP traces to client.")
	return nil
}

func (b *Backend) GetTraces(req *storagev2.GetTracesRequest, srv storagev2.TraceReader_GetTracesServer) error {
	ctx, span := tracer.Start(srv.Context(), "tempo-query.GetTraces")
	defer span.End()

	for _, q := range req.Query {
		if q == nil {
			return fmt.Errorf("q is nil")
		}
		traceId, err := jaeger.TraceIDFromBytes(q.TraceId)
		if err != nil {
			return fmt.Errorf("error converting traceId to jaeger TraceID: %w", err)
		}
		trace, err := b.getTrace(ctx, traceId.String(), q.StartTime, q.EndTime)
		if err != nil {
			if errors.Is(err, ErrTraceNotFound) {
				b.logger.Debug("Trace not found", zap.String("traceID", traceId.String()))
				continue
			}
			return fmt.Errorf("error getting trace: %w", err)
		}
		if trace == nil {
			return fmt.Errorf("got a nil trace")
		}
		convertedTrace := getOtlpTraceData(trace)
		if convertedTrace == nil {
			return fmt.Errorf("failed to convert trace to OTLP format")
		}
		if err := srv.Send(convertedTrace); err != nil {
			return fmt.Errorf("error sending trace: %w", err)
		}
	}

	return nil
}

func (b *Backend) lookupTagValues(ctx context.Context, tagName string, query string) ([]string, error) {
	q := url.Values{}
	if b.QueryServicesDuration != nil {
		startTime, endTime := b.calculateTimeRange()
		q.Set("start", fmt.Sprintf("%d", startTime))
		q.Set("end", fmt.Sprintf("%d", endTime))
	}
	if query != "" {
		q.Set("q", query)
	}
	u, err := url.Parse(fmt.Sprintf("%s://%s/api/v2/search/tag/%s/values", b.apiSchema(), b.tempoBackend, tagName))
	if err != nil {
		return nil, fmt.Errorf("error parsing Tempo URL: %w", err)
	}
	u.RawQuery = q.Encode()

	req, err := b.newGetRequest(ctx, u.String())
	if err != nil {
		return nil, err
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed GET to tempo: %w", err)
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

	var searchLookupResponse tempopb.SearchTagValuesV2Response
	err = jsonpb.Unmarshal(resp.Body, &searchLookupResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Tempo response: %w", err)
	}

	tagValues := make([]string, len(searchLookupResponse.TagValues))
	for i, tagValue := range searchLookupResponse.TagValues {
		tagValues[i] = tagValue.Value
	}
	return tagValues, nil
}

func (b *Backend) WriteSpan(context.Context, *storage_v1.WriteSpanRequest) (*storage_v1.WriteSpanResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Backend) Close(context.Context, *storage_v1.CloseWriterRequest) (*storage_v1.CloseWriterResponse, error) {
	return nil, nil
}

func (b *Backend) newGetRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// currently Jaeger Query will only propagate bearer token to the grpc backend and no other headers
	// so we are going to extract the tenant id from the header, if it exists and use it
	tenantID, found := extractBearerToken(ctx, b.tenantHeaderKey)
	if found {
		req.Header.Set(user.OrgIDHeaderName, tenantID)
	}

	return req, nil
}

func extractBearerToken(ctx context.Context, tenantHeader string) (string, bool) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(tenantHeader)
		if len(values) > 0 {
			return values[0], true
		}

	}
	return "", false
}
