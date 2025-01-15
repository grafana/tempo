package ingest

import (
	"context"
	"fmt"
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

	// query an in-memory trace
	util.QueryAndAssertTrace(t, apiClient, info)

	// wait trace_idle_time and ensure trace is created in ingester
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

	// flush trace to backend
	util.CallFlush(t, tempo)

	// search for trace in backend
	util.SearchAndAssertTrace(t, apiClient, info)
	util.SearchTraceQLAndAssertTrace(t, apiClient, info)

	// sleep
	time.Sleep(10 * time.Second)

	// force clear completed block
	util.CallFlush(t, tempo)

	fmt.Println(tempo.Endpoint(3200))
	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(3), "tempo_query_frontend_queries_total"))

	matchers := []*labels.Matcher{
		{
			Type:  labels.MatchEqual,
			Name:  "receiver",
			Value: "tempo/jaeger_receiver",
		},
		{
			Type:  labels.MatchEqual,
			Name:  "transport",
			Value: "grpc",
		},
	}

	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Greater(1), []string{"tempo_receiver_accepted_spans"}, e2e.WithLabelMatchers(matchers...)))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(0), []string{"tempo_receiver_refused_spans"}, e2e.WithLabelMatchers(matchers...)))

	// query trace - should fetch from backend
	util.QueryAndAssertTrace(t, apiClient, info)

	// search the backend. this works b/c we're passing a start/end AND setting query ingesters within min/max to 0
	now := time.Now()
	util.SearchAndAssertTraceBackend(t, apiClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())

	util.SearchAndAsserTagsBackend(t, apiClient, now.Add(-20*time.Minute).Unix(), now.Unix())

	// find the trace with streaming. using the http server b/c that's what Grafana will do
	grpcClient, err := util.NewSearchGRPCClient(context.Background(), tempo.Endpoint(3200))
	require.NoError(t, err)

	util.SearchStreamAndAssertTrace(t, context.Background(), grpcClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())
}
