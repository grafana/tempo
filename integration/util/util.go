package util

// Collection of utilities to share between our various load tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/e2e"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	mnoop "go.opentelemetry.io/otel/metric/noop"
	tnoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

const (
	image           = "tempo:latest"
	debugImage      = "tempo-debug:latest"
	queryImage      = "tempo-query:latest"
	jaegerImage     = "jaegertracing/jaeger-query:1.64.0"
	prometheusImage = "prom/prometheus:latest"
)

// GetExtraArgs returns the extra args to pass to the Docker command used to run Tempo.
func GetExtraArgs() []string {
	// Get extra args from the TEMPO_EXTRA_ARGS env variable
	// falling back to an empty list
	if os.Getenv("TEMPO_EXTRA_ARGS") != "" {
		return strings.Fields(os.Getenv("TEMPO_EXTRA_ARGS"))
	}

	return nil
}

func buildArgsWithExtra(args, extraArgs []string) []string {
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	if envExtraArgs := GetExtraArgs(); len(envExtraArgs) > 0 {
		args = append(args, envExtraArgs...)
	}

	return args
}

func NewTempoAllInOne(extraArgs ...string) *e2e.HTTPService {
	return NewTempoAllInOneWithReadinessProbe(e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299), extraArgs...)
}

func NewTempoAllInOneDebug(extraArgs ...string) *e2e.HTTPService {
	rp := e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299)
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml")}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		"tempo",
		debugImage,
		e2e.NewCommand("", args...),
		rp,
		3200,  // http all things
		3201,  // http all things
		9095,  // grpc tempo
		14250, // jaeger grpc ingest
		9411,  // zipkin ingest (used by load)
		4317,  // otlp grpc
		4318,  // OTLP HTTP
		2345,  // delve port
	)
	env := map[string]string{
		"DEBUG_BLOCK": "1",
	}
	s.SetEnvVars(env)

	s.SetBackoff(TempoBackoff())
	return s
}

func NewTempoAllInOneWithReadinessProbe(rp e2e.ReadinessProbe, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml")}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		"tempo",
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		rp,
		3200,  // http all things
		3201,  // http all things
		9095,  // grpc tempo
		14250, // jaeger grpc ingest
		9411,  // zipkin ingest (used by load)
		4317,  // otlp grpc
		4318,  // OTLP HTTP
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoDistributor(extraArgs ...string) *e2e.HTTPService {
	return NewNamedTempoDistributor("distributor", extraArgs...)
}

func NewNamedTempoDistributor(name string, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=distributor"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		name,
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
		14250,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoIngester(replica int, extraArgs ...string) *e2e.HTTPService {
	return NewNamedTempoIngester("ingester", replica, extraArgs...)
}

func NewNamedTempoIngester(name string, replica int, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=ingester"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		name+"-"+strconv.Itoa(replica),
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoMetricsGenerator(extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=metrics-generator"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		"metrics-generator",
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoQueryFrontend(extraArgs ...string) *e2e.HTTPService {
	return NewNamedTempoQueryFrontend("query-frontend", extraArgs...)
}

func NewNamedTempoQueryFrontend(name string, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=query-frontend"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		name,
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoQuerier(extraArgs ...string) *e2e.HTTPService {
	return NewNamedTempoQuerier("querier", extraArgs...)
}

func NewNamedTempoQuerier(name string, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=querier"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		name,
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoScalableSingleBinary(replica int, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=scalable-single-binary", "-querier.frontend-address=tempo-" + strconv.Itoa(replica) + ":9095"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		"tempo-"+strconv.Itoa(replica),
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,  // http all things
		14250, // jaeger grpc ingest
		// 9411,  // zipkin ingest (used by load)
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoQuery() *e2e.HTTPService {
	args := []string{
		"-config=" + filepath.Join(e2e.ContainerSharedDir, "config-tempo-query.yaml"),
	}

	s := e2e.NewHTTPService(
		"tempo-query",
		queryImage,
		e2e.NewCommandWithoutEntrypoint("/tempo-query", args...),
		e2e.NewTCPReadinessProbe(7777),
		7777,
	)

	s.SetBackoff(TempoBackoff())
	return s
}

func NewTempoTarget(target string, configFile string) *e2e.HTTPService {
	args := []string{
		"-config.file=" + filepath.Join(e2e.ContainerSharedDir, configFile),
		"-target=" + target,
	}

	s := e2e.NewHTTPService(
		target,
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewJaegerQuery() *e2e.HTTPService {
	args := []string{
		"--grpc-storage.server=tempo-query:7777",
		"--span-storage.type=grpc",
	}

	s := e2e.NewHTTPService(
		"jaeger-query",
		jaegerImage,
		e2e.NewCommandWithoutEntrypoint("/go/bin/query-linux", args...),
		e2e.NewHTTPReadinessProbe(16686, "/", 200, 299),
		16686,
	)

	s.SetBackoff(TempoBackoff())
	return s
}

func CopyFileToSharedDir(s *e2e.Scenario, src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("unable to read local file %s: %w", src, err)
	}

	_, err = writeFileToSharedDir(s, dst, content)
	return err
}

func CopyTemplateToSharedDir(s *e2e.Scenario, src, dst string, data any) (string, error) {
	tmpl, err := template.ParseFiles(src)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return writeFileToSharedDir(s, dst, buf.Bytes())
}

func writeFileToSharedDir(s *e2e.Scenario, dst string, content []byte) (string, error) {
	dst = filepath.Join(s.SharedDir(), dst)

	// NOTE: since the integration tests are setup outside of the container
	// before container execution, the permissions within the container must be
	// able to read the configuration.

	// Ensure the entire path of directories exists
	err := os.MkdirAll(filepath.Dir(dst), os.ModePerm)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(dst, content, os.ModePerm)
	if err != nil {
		return "", err
	}

	return dst, nil
}

func TempoBackoff() backoff.Config {
	return backoff.Config{
		MinBackoff: 500 * time.Millisecond,
		MaxBackoff: time.Second,
		MaxRetries: 300, // Sometimes the CI is slow ¯\_(ツ)_/¯
	}
}

func NewOtelGRPCExporter(endpoint string) (exporter.Traces, error) {
	factory := otlpexporter.NewFactory()
	exporterCfg := factory.CreateDefaultConfig()
	otlpCfg := exporterCfg.(*otlpexporter.Config)
	otlpCfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: endpoint,
		TLSSetting: configtls.ClientConfig{
			Insecure: true,
		},
	}
	logger, _ := zap.NewDevelopment()
	return factory.CreateTraces(
		context.Background(),
		exporter.Settings{
			TelemetrySettings: component.TelemetrySettings{
				Logger:         logger,
				TracerProvider: tnoop.NewTracerProvider(),
				MeterProvider:  mnoop.NewMeterProvider(),
			},
			BuildInfo: component.NewDefaultBuildInfo(),
		},
		otlpCfg,
	)
}

func NewJaegerGRPCClient(endpoint string) (*jaeger_grpc.Reporter, error) {
	// new jaeger grpc exporter
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return jaeger_grpc.NewReporter(conn, nil, logger), err
}

func NewSearchGRPCClient(ctx context.Context, endpoint string) (tempopb.StreamingQuerierClient, error) {
	return NewSearchGRPCClientWithCredentials(ctx, endpoint, insecure.NewCredentials())
}

func NewSearchGRPCClientWithCredentials(ctx context.Context, endpoint string, creds credentials.TransportCredentials) (tempopb.StreamingQuerierClient, error) {
	clientConn, err := grpc.DialContext(ctx, endpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return tempopb.NewStreamingQuerierClient(clientConn), nil
}

func SearchAndAssertTrace(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)

	// NOTE: SearchTags doesn't include live traces anymore
	// so don't check SearchTags

	// verify attribute value is present in tag values
	tagValuesResp, err := client.SearchTagValues(attr.Key)
	require.NoError(t, err)
	require.Contains(t, tagValuesResp.TagValues, attr.GetValue().GetStringValue())

	// verify trace can be found using attribute
	resp, err := client.Search(attr.GetKey() + "=" + attr.GetValue().GetStringValue())
	require.NoError(t, err)

	require.True(t, traceIDInResults(t, info.HexID(), resp))
}

func SearchTraceQLAndAssertTrace(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)
	query := fmt.Sprintf(`{ .%s = "%s"}`, attr.GetKey(), attr.GetValue().GetStringValue())

	resp, err := client.SearchTraceQL(query)
	require.NoError(t, err)

	require.True(t, traceIDInResults(t, info.HexID(), resp))
}

// SearchStreamAndAssertTrace will search and assert that the trace is present in the streamed results.
// nolint: revive
func SearchStreamAndAssertTrace(t *testing.T, ctx context.Context, client tempopb.StreamingQuerierClient, info *tempoUtil.TraceInfo, start, end int64) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)
	query := fmt.Sprintf(`{ .%s = "%s"}`, attr.GetKey(), attr.GetValue().GetStringValue())

	// -- assert search
	resp, err := client.Search(ctx, &tempopb.SearchRequest{
		Query: query,
		Start: uint32(start),
		End:   uint32(end),
	})
	require.NoError(t, err)

	// drain the stream until everything is returned while watching for the trace in question
	found := false
	for {
		resp, err := resp.Recv()
		if resp != nil {
			found = traceIDInResults(t, info.HexID(), resp)
			if found {
				break
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.True(t, found)
}

// by passing a time range and using a query_ingesters_until/backend_after of 0 we can force the queriers
// to look in the backend blocks
func SearchAndAssertTraceBackend(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo, start, end int64) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)

	// verify trace can be found using attribute and time range
	resp, err := client.SearchWithRange(attr.GetKey()+"="+attr.GetValue().GetStringValue(), start, end)
	require.NoError(t, err)

	require.True(t, traceIDInResults(t, info.HexID(), resp))
}

// by passing a time range and using a query_ingesters_until/backend_after of 0 we can force the queriers
// to look in the backend blocks
func SearchAndAsserTagsBackend(t *testing.T, client *httpclient.Client, start, end int64) {
	// There are no tags in recent data
	resp, err := client.SearchTags()
	require.NoError(t, err)
	require.Equal(t, len(resp.TagNames), 0)

	// There are additional tags in the backend
	resp, err = client.SearchTagsWithRange(start, end)
	require.NoError(t, err)
	require.True(t, len(resp.TagNames) > 0)
}

func traceIDInResults(t *testing.T, hexID string, resp *tempopb.SearchResponse) bool {
	for _, s := range resp.Traces {
		equal, err := tempoUtil.EqualHexStringTraceIDs(s.TraceID, hexID)
		require.NoError(t, err)
		if equal {
			return true
		}
	}

	return false
}

func MakeThriftBatch() *thrift.Batch {
	return MakeThriftBatchWithSpanCount(1)
}

func MakeThriftBatchWithSpanCount(n int) *thrift.Batch {
	return MakeThriftBatchWithSpanCountAttributeAndName(n, "my operation", "", "y", "xx", "x")
}

func MakeThriftBatchWithSpanCountAttributeAndName(n int, name, resourceValue, spanValue, resourceTag, spanTag string) *thrift.Batch {
	var spans []*thrift.Span

	traceIDLow := rand.Int63()
	traceIDHigh := rand.Int63()
	for i := 0; i < n; i++ {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        rand.Int63(),
			ParentSpanId:  0,
			OperationName: name,
			References:    nil,
			Flags:         0,
			StartTime:     time.Now().UnixNano() / 1000, // microsecconds
			Duration:      1,
			Tags: []*thrift.Tag{
				{
					Key:  spanTag,
					VStr: &spanValue,
				},
			},
			Logs: nil,
		})
	}

	return &thrift.Batch{
		Process: &thrift.Process{
			ServiceName: "my-service",
			Tags: []*thrift.Tag{
				{
					Key:   resourceTag,
					VType: thrift.TagType_STRING,
					VStr:  &resourceValue,
				},
			},
		},
		Spans: spans,
	}
}

func CallFlush(t *testing.T, ingester *e2e.HTTPService) {
	fmt.Printf("Calling /flush on %s\n", ingester.Name())
	res, err := e2e.DoGet("http://" + ingester.Endpoint(3200) + "/flush")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
}

func CallIngesterRing(t *testing.T, svc *e2e.HTTPService) {
	endpoint := "/ingester/ring"
	fmt.Printf("Calling %s on %s\n", endpoint, svc.Name())
	res, err := e2e.DoGet("http://" + svc.Endpoint(3200) + endpoint)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func CallCompactorRing(t *testing.T, svc *e2e.HTTPService) {
	endpoint := "/compactor/ring"
	fmt.Printf("Calling %s on %s\n", endpoint, svc.Name())
	res, err := e2e.DoGet("http://" + svc.Endpoint(3200) + endpoint)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func CallStatus(t *testing.T, svc *e2e.HTTPService) {
	endpoint := "/status/endpoints"
	fmt.Printf("Calling %s on %s\n", endpoint, svc.Name())
	res, err := e2e.DoGet("http://" + svc.Endpoint(3200) + endpoint)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func CallBuildinfo(t *testing.T, svc *e2e.HTTPService) {
	endpoint := "/api/status/buildinfo"
	fmt.Printf("Calling %s on %s\n", endpoint, svc.Name())
	res, err := e2e.DoGet("http://" + svc.Endpoint(3200) + endpoint)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Check that the actual JSON response contains all the expected keys (we disregard the values)
	var jsonResponse map[string]any
	keys := []string{"version", "revision", "branch", "buildDate", "buildUser", "goVersion"}
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &jsonResponse)
	require.NoError(t, err)
	for _, key := range keys {
		_, ok := jsonResponse[key]
		require.True(t, ok)
	}

	version, ok := jsonResponse["version"].(string)
	require.True(t, ok)
	semverRegex := `^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	require.Regexp(t, semverRegex, version)

	defer res.Body.Close()
}

func AssertEcho(t *testing.T, url string) {
	res, err := e2e.DoGet(url)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	defer func() { require.NoError(t, res.Body.Close()) }()
}

func QueryAndAssertTrace(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	resp, err := client.QueryTrace(info.HexID())
	require.NoError(t, err)

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	AssertEqualTrace(t, resp, expected)
}

func AssertEqualTrace(t *testing.T, a, b *tempopb.Trace) {
	t.Helper()
	trace.SortTraceAndAttributes(a)
	trace.SortTraceAndAttributes(b)

	assert.Equal(t, a, b)
}

func SpanCount(a *tempopb.Trace) float64 {
	count := 0
	for _, batch := range a.ResourceSpans {
		for _, spans := range batch.ScopeSpans {
			count += len(spans.Spans)
		}
	}

	return float64(count)
}

func NewPrometheus() *e2e.HTTPService {
	return e2e.NewHTTPService(
		"prometheus",
		prometheusImage,
		e2e.NewCommandWithoutEntrypoint("/bin/prometheus", "--config.file=/etc/prometheus/prometheus.yml", "--web.enable-remote-write-receiver"),
		e2e.NewHTTPReadinessProbe(9090, "/-/ready", 200, 299),
		9090,
	)
}
