package e2e

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/e2e"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	configLimits             = "config-limits.yaml"
	configLimitsQuery        = "config-limits-query.yaml"
	configLimitsPartialError = "config-limits-partial-success.yaml"
)

func TestLimits(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configLimits, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Get port for the otlp receiver endpoint
	c, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	// should fail b/c the trace is too large. each batch should be ~70 bytes
	batch := makeThriftBatchWithSpanCount(2)
	require.NoError(t, c.EmitBatch(context.Background(), batch), "max trace size")

	// push a trace
	require.NoError(t, c.EmitBatch(context.Background(), makeThriftBatchWithSpanCount(1)))

	// should fail b/c this will be too many traces
	batch = makeThriftBatch()
	require.NoError(t, c.EmitBatch(context.Background(), batch), "too many traces")

	// should fail b/c due to ingestion rate limit
	batch = makeThriftBatchWithSpanCount(10)
	err = c.EmitBatch(context.Background(), batch)
	require.Error(t, err)

	// this error must have a retryinfo as expected in otel collector code: https://github.com/open-telemetry/opentelemetry-collector/blob/d7b49df5d9e922df6ce56ad4b64ee1c79f9dbdbe/exporter/otlpexporter/otlp.go#L172
	st, ok := status.FromError(err)
	require.True(t, ok)
	foundRetryInfo := false
	for _, detail := range st.Details() {
		if _, ok := detail.(*errdetails.RetryInfo); ok {
			foundRetryInfo = true
			break
		}
	}
	require.True(t, foundRetryInfo)

	// test limit metrics
	err = tempo.WaitSumMetricsWithOptions(e2e.Equals(2),
		[]string{"tempo_discarded_spans_total"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "trace_too_large")),
	)
	require.NoError(t, err)
	err = tempo.WaitSumMetricsWithOptions(e2e.Equals(1),
		[]string{"tempo_discarded_spans_total"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "live_traces_exceeded")),
	)
	require.NoError(t, err)
	err = tempo.WaitSumMetricsWithOptions(e2e.Equals(10),
		[]string{"tempo_discarded_spans_total"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "rate_limited")),
	)
	require.NoError(t, err)
}

func TestOTLPLimits(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configLimits, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	protoSpans := test.MakeProtoSpans(100)

	// gRPC
	grpcClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(tempo.Endpoint(4317)),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{Enabled: false}),
	)
	require.NoError(t, grpcClient.Start(context.Background()))

	grpcErr := grpcClient.UploadTraces(context.Background(), protoSpans)
	assert.Error(t, grpcErr)
	require.Equal(t, codes.ResourceExhausted, status.Code(grpcErr))

	// HTTP
	httpClient := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(tempo.Endpoint(4318)),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{Enabled: false}),
	)
	require.NoError(t, httpClient.Start(context.Background()))

	httpErr := httpClient.UploadTraces(context.Background(), protoSpans)
	assert.Error(t, httpErr)
	require.Contains(t, httpErr.Error(), http.StatusText(http.StatusInternalServerError)) // Should be 429
}

func TestQueryLimits(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configLimitsQuery, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Get port for the otlp receiver endpoint
	c, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	// make a trace with 10 spans and push them one at a time, flush in between each one to force different blocks
	batch := makeThriftBatchWithSpanCount(5)
	allSpans := batch.Spans
	for i := range batch.Spans {
		batch.Spans = allSpans[i : i+1]
		require.NoError(t, c.EmitBatch(context.Background(), batch))
		callFlush(t, tempo)
		time.Sleep(2 * time.Second) // trace idle and flush time are both 1ms
	}

	// calc trace id
	traceID := [16]byte{}
	binary.BigEndian.PutUint64(traceID[:8], uint64(batch.Spans[0].TraceIdHigh))
	binary.BigEndian.PutUint64(traceID[8:], uint64(batch.Spans[0].TraceIdLow))

	// now try to query it back. this should fail b/c the trace is too large
	client := httpclient.New("http://"+tempo.Endpoint(3200), tempoUtil.FakeTenantID)
	querierClient := httpclient.New("http://"+tempo.Endpoint(3200)+"/querier", tempoUtil.FakeTenantID)

	_, err = client.QueryTrace(tempoUtil.TraceIDToHexString(traceID[:]))
	require.ErrorContains(t, err, "trace exceeds max size")
	require.ErrorContains(t, err, "failed with response: 500") // confirm frontend returns 500

	_, err = querierClient.QueryTrace(tempoUtil.TraceIDToHexString(traceID[:]))
	require.ErrorContains(t, err, "trace exceeds max size")
	require.ErrorContains(t, err, "failed with response: 500") // todo: this should return 400 ideally so the frontend does not retry, but does not currently

	// complete block timeout  is 10 seconds
	time.Sleep(15 * time.Second)
	_, err = client.QueryTrace(tempoUtil.TraceIDToHexString(traceID[:]))
	require.ErrorContains(t, err, "trace exceeds max size")
	require.ErrorContains(t, err, "failed with response: 500") // confirm frontend returns 500

	_, err = querierClient.QueryTrace(tempoUtil.TraceIDToHexString(traceID[:]))
	require.ErrorContains(t, err, "trace exceeds max size")
	require.ErrorContains(t, err, "failed with response: 400") // confirm querier returns 400
}

func TestLimitsPartialSuccess(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, util.CopyFileToSharedDir(s, configLimitsPartialError, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// otel grpc exporter
	exporter, err := util.NewOtelGRPCExporter(tempo.Endpoint(4317))
	require.NoError(t, err)

	err = exporter.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// make request
	traceIDs := make([][]byte, 6)
	for index := range traceIDs {
		traceID := make([]byte, 16)
		_, err = crand.Read(traceID)
		require.NoError(t, err)
		traceIDs[index] = traceID
	}

	// 3 traces with trace_too_large and 3 with no error
	spanCountsByTrace := []int{1, 4, 1, 5, 6, 1}
	req := test.MakeReqWithMultipleTraceWithSpanCount(spanCountsByTrace, traceIDs)

	b, err := req.Marshal()
	require.NoError(t, err)

	// unmarshal into otlp proto
	traces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(b)
	require.NoError(t, err)
	require.NotNil(t, traces)

	ctx := user.InjectOrgID(context.Background(), tempoUtil.FakeTenantID)
	ctx, err = user.InjectIntoGRPCRequest(ctx)
	require.NoError(t, err)

	// send traces to tempo
	// partial success = no error
	err = exporter.ConsumeTraces(ctx, traces)
	require.NoError(t, err)

	// shutdown to ensure traces are flushed
	require.NoError(t, exporter.Shutdown(context.Background()))

	// query for the one trace that didn't trigger an error
	client := httpclient.New("http://"+tempo.Endpoint(3200), tempoUtil.FakeTenantID)
	for i, count := range spanCountsByTrace {
		if count == 1 {
			result, err := client.QueryTrace(tempoUtil.TraceIDToHexString(traceIDs[i]))
			require.NoError(t, err)
			assert.Equal(t, 1, len(result.Batches))
		}
	}

	// test metrics
	// 3 traces with trace_too_large each with 4+5+6 spans
	err = tempo.WaitSumMetricsWithOptions(e2e.Equals(15),
		[]string{"tempo_discarded_spans_total"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "trace_too_large")),
	)
	require.NoError(t, err)

	// this metric should never exist
	err = tempo.WaitSumMetricsWithOptions(e2e.Equals(0),
		[]string{"tempo_discarded_spans_total"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "unknown_error")),
	)
	require.NoError(t, err)
}
