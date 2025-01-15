package tempo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logfmt/logfmt"
	"github.com/gogo/protobuf/jsonpb"
	tlsCfg "github.com/grafana/dskit/crypto/tls"
	"github.com/grafana/dskit/user"
	jaeger "github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	jaeger_spanstore "github.com/jaegertracing/jaeger/storage/spanstore"
	ot_jaeger "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	AcceptHeaderKey         = "Accept"
	ProtobufTypeHeaderValue = "application/protobuf"
)

const (
	tagsSearchTag        = "tags"
	serviceSearchTag     = "service.name"
	operationSearchTag   = "name"
	minDurationSearchTag = "minDuration"
	maxDurationSearchTag = "maxDuration"
	startTimeMaxTag      = "end"
	startTimeMinTag      = "start"
	numTracesSearchTag   = "limit"
)

var tlsVersions = map[string]uint16{
	"VersionTLS10": tls.VersionTLS10,
	"VersionTLS11": tls.VersionTLS11,
	"VersionTLS12": tls.VersionTLS12,
	"VersionTLS13": tls.VersionTLS13,
}

var (
	_ storage_v1.SpanReaderPluginServer         = (*Backend)(nil)
	_ storage_v1.DependenciesReaderPluginServer = (*Backend)(nil)
	_ storage_v1.SpanWriterPluginServer         = (*Backend)(nil)
)

var tracer = otel.Tracer("cmd/tempo-query/tempo")

type Backend struct {
	logger                       *zap.Logger
	tempoBackend                 string
	tlsEnabled                   bool
	tls                          tlsCfg.ClientConfig
	httpClient                   *http.Client
	tenantHeaderKey              string
	QueryServicesDuration        *time.Duration
	findTracesConcurrentRequests int
}

func New(logger *zap.Logger, cfg *Config) (*Backend, error) {
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
		return http.DefaultClient, nil
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
	return &http.Client{Transport: transport}, nil
}

func mapCipherNamesToIDs(cipherSuiteNames []string) ([]uint16, error) {
	cipherSuites := []uint16{}
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

func (b *Backend) GetDependencies(context.Context, *storage_v1.GetDependenciesRequest) (*storage_v1.GetDependenciesResponse, error) {
	return &storage_v1.GetDependenciesResponse{}, nil
}

func (b *Backend) apiSchema() string {
	if b.tlsEnabled {
		return "https"
	}
	return "http"
}

func (b *Backend) GetTrace(req *storage_v1.GetTraceRequest, stream storage_v1.SpanReaderPlugin_GetTraceServer) error {
	jt, err := b.getTrace(stream.Context(), req.TraceID)
	if err != nil {
		return err
	}

	spans := make([]jaeger.Span, len(jt.Spans))
	for i, span := range jt.Spans {
		spans[i] = *span
	}

	return stream.Send(&storage_v1.SpansResponseChunk{Spans: spans})
}

func (b *Backend) getTrace(ctx context.Context, traceID jaeger.TraceID) (*jaeger.Trace, error) {
	url := fmt.Sprintf("%s://%s/api/traces/%s", b.apiSchema(), b.tempoBackend, traceID)

	ctx, span := tracer.Start(ctx, "tempo-query.GetTrace")
	defer span.End()

	req, err := b.newGetRequest(ctx, url)
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
		return nil, jaeger_spanstore.ErrTraceNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from tempo: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", body)
	}

	otTrace, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(body)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling body to otlp trace %v: %w", traceID, err)
	}

	jaegerBatches := ot_jaeger.ProtoFromTraces(otTrace)

	jaegerTrace := &jaeger.Trace{
		Spans:      []*jaeger.Span{},
		ProcessMap: []jaeger.Trace_ProcessMapping{},
	}

	span.AddEvent("build process map")
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

func (b *Backend) calculateTimeRange() (int64, int64) {
	now := time.Now()
	start := now.Add(*b.QueryServicesDuration * -1)
	return start.Unix(), now.Unix()
}

func (b *Backend) GetServices(ctx context.Context, _ *storage_v1.GetServicesRequest) (*storage_v1.GetServicesResponse, error) {
	ctx, span := tracer.Start(ctx, "tempo-query.GetOperations")
	defer span.End()

	services, err := b.lookupTagValues(ctx, serviceSearchTag)
	if err != nil {
		return nil, err
	}
	return &storage_v1.GetServicesResponse{Services: services}, nil
}

func (b *Backend) GetOperations(ctx context.Context, _ *storage_v1.GetOperationsRequest) (*storage_v1.GetOperationsResponse, error) {
	ctx, span := tracer.Start(ctx, "tempo-query.GetOperations")
	defer span.End()

	tagValues, err := b.lookupTagValues(ctx, operationSearchTag)
	if err != nil {
		return nil, err
	}

	var operations []*storage_v1.Operation
	for _, value := range tagValues {
		operations = append(operations, &storage_v1.Operation{
			Name:     value,
			SpanKind: "",
		})
	}

	return &storage_v1.GetOperationsResponse{
		OperationNames: tagValues,
		Operations:     operations,
	}, nil
}

type job struct {
	ctx     context.Context
	traceID jaeger.TraceID
}

type jobResult struct {
	traceID jaeger.TraceID
	trace   *jaeger.Trace
	err     error
}

func worker(b *Backend, jobs <-chan job, results chan<- jobResult) {
	for job := range jobs {
		jaegerTrace, err := b.getTrace(job.ctx, job.traceID)
		results <- jobResult{
			traceID: job.traceID,
			trace:   jaegerTrace,
			err:     err,
		}
	}
}

func (b *Backend) FindTraces(req *storage_v1.FindTracesRequest, stream storage_v1.SpanReaderPlugin_FindTracesServer) error {
	ctx, span := tracer.Start(stream.Context(), "tempo-query.FindTraces")
	defer span.End()

	resp, err := b.FindTraceIDs(ctx, &storage_v1.FindTraceIDsRequest{Query: req.Query})
	if err != nil {
		return err
	}

	span.AddEvent(fmt.Sprintf("Found %d trace IDs", len(resp.TraceIDs)))
	b.logger.Info("FindTraces: fetching traces", zap.Int("traceids", len(resp.TraceIDs)))

	numWorkers := b.findTracesConcurrentRequests
	jobs := make(chan job, len(resp.TraceIDs))
	results := make(chan jobResult, len(resp.TraceIDs))
	var workersDone sync.WaitGroup
	// Start workers
	for w := 0; w < numWorkers; w++ {
		workersDone.Add(1)
		go func() { defer workersDone.Done(); worker(b, jobs, results) }()
	}

	// for every traceID, get the full trace
	var jaegerTraces []*jaeger.Trace
	for _, traceID := range resp.TraceIDs {
		jobs <- job{
			ctx:     ctx,
			traceID: traceID,
		}
	}
	close(jobs)
	workersDone.Wait()

	var failedTraces []jobResult
	// Collecting results
	for i := 0; i < len(resp.TraceIDs); i++ {
		result := <-results
		if result.err != nil {
			// TODO this seems to be an internal inconsistency error, ignore so we can still show the rest
			b.logger.Info("failed to get a trace", zap.Error(err), zap.String("traceid", result.traceID.String()))
			span.AddEvent(fmt.Sprintf("could not get trace for traceID %v", result.traceID))
			span.RecordError(err)
			failedTraces = append(failedTraces, result)
		} else {
			jaegerTraces = append(jaegerTraces, result.trace)
		}
	}
	close(results)
	if len(failedTraces) > 0 {
		b.logger.Info("FindTraces: failed to find traces, getTrace failed", zap.Int32("limit", req.Query.NumTraces), zap.Int("failed", len(failedTraces)))
	}

	span.AddEvent(fmt.Sprintf("Returning %d traces", len(jaegerTraces)))

	for _, jt := range jaegerTraces {
		spans := make([]jaeger.Span, len(jt.Spans))
		for i, span := range jt.Spans {
			spans[i] = *span
		}
		if err := stream.Send(&storage_v1.SpansResponseChunk{Spans: spans}); err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) FindTraceIDs(ctx context.Context, r *storage_v1.FindTraceIDsRequest) (*storage_v1.FindTraceIDsResponse, error) {
	ctx, span := tracer.Start(ctx, "tempo-query.FindTraceIDs")
	defer span.End()

	url := url.URL{
		Scheme: b.apiSchema(),
		Host:   b.tempoBackend,
		Path:   "api/search",
	}
	urlQuery := url.Query()
	urlQuery.Set(minDurationSearchTag, r.Query.DurationMin.String())
	urlQuery.Set(maxDurationSearchTag, r.Query.DurationMax.String())
	urlQuery.Set(numTracesSearchTag, strconv.Itoa(int(r.Query.GetNumTraces())))
	urlQuery.Set(startTimeMaxTag, fmt.Sprintf("%d", r.Query.StartTimeMax.Unix()))
	urlQuery.Set(startTimeMinTag, fmt.Sprintf("%d", r.Query.StartTimeMin.Unix()))

	queryParam, err := createTagsQueryParam(
		r.Query.ServiceName,
		r.Query.OperationName,
		r.Query.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tags query parameter: %w", err)
	}
	urlQuery.Set(tagsSearchTag, queryParam)

	url.RawQuery = urlQuery.Encode()

	req, err := b.newGetRequest(ctx, url.String())
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

	var searchResponse tempopb.SearchResponse
	err = jsonpb.Unmarshal(resp.Body, &searchResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Tempo response: %w", err)
	}

	jaegerTraceIDs := make([]jaeger.TraceID, len(searchResponse.Traces))

	for i, traceMetadata := range searchResponse.Traces {
		jaegerTraceID, err := jaeger.TraceIDFromString(traceMetadata.TraceID)
		if err != nil {
			return nil, fmt.Errorf("could not convert traceID into Jaeger's traceID: %w", err)
		}
		jaegerTraceIDs[i] = jaegerTraceID
	}

	return &storage_v1.FindTraceIDsResponse{TraceIDs: jaegerTraceIDs}, nil
}

func createTagsQueryParam(service string, operation string, tags map[string]string) (string, error) {
	tagsBuilder := &strings.Builder{}
	tagsEncoder := logfmt.NewEncoder(tagsBuilder)
	err := tagsEncoder.EncodeKeyval(serviceSearchTag, service)
	if err != nil {
		return "", err
	}
	if operation != "" {
		err := tagsEncoder.EncodeKeyval(operationSearchTag, operation)
		if err != nil {
			return "", err
		}
	}
	for k, v := range tags {
		err := tagsEncoder.EncodeKeyval(k, v)
		if err != nil {
			return "", err
		}
	}
	return tagsBuilder.String(), nil
}

func (b *Backend) lookupTagValues(ctx context.Context, tagName string) ([]string, error) {
	var url string

	if b.QueryServicesDuration == nil {
		url = fmt.Sprintf("%s://%s/api/search/tag/%s/values", b.apiSchema(), b.tempoBackend, tagName)
	} else {
		startTime, endTime := b.calculateTimeRange()
		url = fmt.Sprintf("%s://%s/api/search/tag/%s/values?start=%d&end=%d", b.apiSchema(), b.tempoBackend, tagName, startTime, endTime)
	}

	req, err := b.newGetRequest(ctx, url)
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

	var searchLookupResponse tempopb.SearchTagValuesResponse
	err = jsonpb.Unmarshal(resp.Body, &searchLookupResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Tempo response: %w", err)
	}

	return searchLookupResponse.TagValues, nil
}

func (b *Backend) WriteSpan(context.Context, *storage_v1.WriteSpanRequest) (*storage_v1.WriteSpanResponse, error) {
	return nil, nil
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
