package e2e

import (
	"context"
	crand "crypto/rand"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"

	"github.com/grafana/e2e"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	configLimits             = "config-limits.yaml"
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
	require.Error(t, c.EmitBatch(context.Background(), batch))

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
	traceIDs := make([][]byte, 4)
	for index := range traceIDs {
		traceID := make([]byte, 16)
		_, err = crand.Read(traceID)
		require.NoError(t, err)
		traceIDs[index] = traceID
	}

	// 3 traces with trace_too_large and 1 with no error
	req := test.MakeReqWithMultipleTraceWithSpanCount([]int{4, 4, 4, 1}, traceIDs)

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
	result, err := client.QueryTrace(tempoUtil.TraceIDToHexString(traceIDs[3]))
	require.NoError(t, err)
	assert.Equal(t, 1, len(result.Batches))

	// test metrics
	// 3 traces with trace_too_large each with 4 spans
	err = tempo.WaitSumMetricsWithOptions(e2e.Equals(3*4),
		[]string{"tempo_discarded_spans_total"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "trace_too_large")),
	)
	require.NoError(t, err)
}
