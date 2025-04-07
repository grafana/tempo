package ingest

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestIngest(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-kafka.yaml", "config.yaml"))

	kafka := NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Wait until joined to partition ring
	matchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: "Active"},
		{Type: labels.MatchEqual, Name: "name", Value: "ingester-partitions"},
	}
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(matchers...)))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	time.Sleep(5 * time.Minute)

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

	// test echo
	util.AssertEcho(t, "http://"+tempo.Endpoint(3200)+"/api/echo")

	apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "")

	// wait until block-builder block is flushed
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_block_builder_flushed_blocks"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Greater(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))

	// wait trace_idle_time and ensure trace is created in ingester
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

	// flush trace to backend
	util.CallFlush(t, tempo)

	// sleep
	time.Sleep(10 * time.Second)

	// force clear completed block
	util.CallFlush(t, tempo)

	// query trace - should fetch from backend
	util.QueryAndAssertTrace(t, apiClient, info)

	// search the backend. this works b/c we're passing a start/end AND setting query ingesters within min/max to 0
	now := time.Now()
	util.SearchAndAssertTraceBackend(t, apiClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())
	util.SearchAndAsserTagsBackend(t, apiClient, now.Add(-20*time.Minute).Unix(), now.Unix())
}
