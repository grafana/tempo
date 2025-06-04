package ingest

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestPartitionDownscale(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-partition-downscale.yaml", "config.yaml"))

	// Start dependencies
	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	minio := e2edb.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	// Start Tempo components
	distributor := util.NewTempoDistributor()
	ingester := util.NewTempoIngester(0)
	querier := util.NewTempoQuerier()
	queryFrontend := util.NewTempoQueryFrontend()

	require.NoError(t, s.StartAndWaitReady(distributor, ingester, querier, queryFrontend))

	// Wait until ingester and metrics-generator are active
	isServiceActiveMatcher := func(service string) []*labels.Matcher {
		return []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "name", service),
			labels.MustNewMatcher(labels.MatchEqual, "state", "ACTIVE"),
		}
	}
	require.NoError(t, distributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(isServiceActiveMatcher("ingester")...), e2e.WaitMissingMetrics))

	// Wait until joined to partition ring
	partitionStateMatchers := func(state string) []*labels.Matcher {
		return []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "state", Value: state},
			{Type: labels.MatchEqual, Name: "name", Value: "ingester-partitions"},
		}
	}
	require.NoError(t, distributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(partitionStateMatchers("Active")...)))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerGRPCClient(distributor.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	// Generate and emit initial traces
	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	// Wait for traces to be received
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

	// Create API client
	apiClient := httpclient.New("http://"+queryFrontend.Endpoint(3200), "")

	// Set ingester's partition to INACTIVE state (prepare for downscale)
	req, err := http.NewRequest("POST", "http://"+ingester.Endpoint(3200)+"/ingester/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err := apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)

	// Verify ingester's partition is INACTIVE by checking the timestamp
	req, err = http.NewRequest("GET", "http://"+ingester.Endpoint(3200)+"/ingester/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err = apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)
	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(httpResp.Body).Decode(&result))
	require.Greater(t, result["timestamp"].(float64), float64(0)) // ts > 0 ==> INACTIVE (when it was marked for downscale)

	require.NoError(t, distributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(partitionStateMatchers("Inactive")...)))

	// Start block-builder (it should consume data from the downscaled partition)
	blockbuilder := util.NewTempoBlockBuilder(0)
	require.NoError(t, s.StartAndWaitReady(blockbuilder))

	// Wait for blocks to be flushed from the downscaled partition
	require.NoError(t, blockbuilder.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_block_builder_flushed_blocks"}, e2e.WaitMissingMetrics))
	require.NoError(t, queryFrontend.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempodb_blocklist_length"},
		e2e.WaitMissingMetrics, e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: "single-tenant"})))

	// Verify initial traces can be queried from backend storage
	trace, err := apiClient.QueryTrace(info.HexID())
	require.NoError(t, err)
	require.NotNil(t, trace)

	// Set ingester's partition back to ACTIVE state
	req, err = http.NewRequest("DELETE", "http://"+ingester.Endpoint(3200)+"/ingester/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err = apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)

	// Verify ingester's partition is ACTIVE by checking the timestamp is 0
	req, err = http.NewRequest("GET", "http://"+ingester.Endpoint(3200)+"/ingester/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err = apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)
	require.NoError(t, json.NewDecoder(httpResp.Body).Decode(&result))
	require.Equal(t, float64(0), result["timestamp"].(float64)) // ts == 0 ==> ACTIVE

	require.NoError(t, distributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(partitionStateMatchers("Active")...)))

	// Generate and emit more traces after reactivating the partition
	info2 := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info2.EmitAllBatches(c))

	// Wait for new traces to be received
	expected2, err := info2.ConstructTraceFromEpoch()
	require.NoError(t, err)
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)+util.SpanCount(expected2)), "tempo_distributor_spans_received_total"))

	// Wait for the new traces to be flushed by block-builder
	require.NoError(t, blockbuilder.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(2), []string{"tempo_block_builder_flushed_blocks"}, e2e.WaitMissingMetrics))

	// Verify all traces using trace ID lookup
	trace, err = apiClient.QueryTrace(info.HexID())
	require.NoError(t, err)
	require.NotNil(t, trace)
	require.Equal(t, util.SpanCount(expected), util.SpanCount(trace))

	trace2, err := apiClient.QueryTrace(info2.HexID())
	require.NoError(t, err)
	require.NotNil(t, trace2)
	require.Equal(t, util.SpanCount(expected2), util.SpanCount(trace2))
}
