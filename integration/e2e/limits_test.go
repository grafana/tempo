package e2e

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/grafana/e2e"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

const (
	configLimits      = "config-limits.yaml"
	configLimitsQuery = "config-limits-query.yaml"
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
	require.Error(t, c.EmitBatch(context.Background(), batch), "max trace size")

	// push a trace
	require.NoError(t, c.EmitBatch(context.Background(), makeThriftBatchWithSpanCount(1)))

	// should fail b/c this will be too many traces
	batch = makeThriftBatch()
	require.Error(t, c.EmitBatch(context.Background(), batch), "too many traces")

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
