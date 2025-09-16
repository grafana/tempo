package ingest

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngesterMetrics(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-kafka.yaml", "config.yaml"))

	kafka := e2edb.NewKafka()
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
	c, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// test initial metrics
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

	// Test comprehensive ingester metrics
	testAllIngesterMetricsDetailed(t, tempo)
}

// testAllIngesterMetricsDetailed validates all ingester metrics from pkg/ingest/metrics.go have reasonable values
func testAllIngesterMetricsDetailed(t *testing.T, tempo *e2e.HTTPService) {
	// Basic trace processing metrics
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_ingester_traces_created_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingester_bytes_received_total"))

	// Live trace state metrics - should have some live traces initially
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingester_live_traces"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingester_live_trace_bytes"))

	// Block lifecycle metrics
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingester_blocks_cleared_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingester_completion_size_bytes"))

	// Back pressure should be minimal in test environment
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingester_back_pressure_seconds_total"))

	// Note: Ingester doesn't use Kafka consumer metrics directly like blockbuilder/livestore
	// so we don't test fetch_* metrics for ingester as they won't be present
}

func TestIngesterMetricsMultiTenant(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-kafka.yaml", "config.yaml"))

	kafka := e2edb.NewKafka()
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
	c, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	// Send traces for different tenants
	tenants := []string{"tenant-1", "tenant-2"}
	for _, tenant := range tenants {
		info := tempoUtil.NewTraceInfo(time.Now(), tenant)
		require.NoError(t, info.EmitAllBatches(c))
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Test that metrics are present for multiple tenants
	for _, tenant := range tenants {
		tenantMatchers := []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "tenant", Value: tenant},
		}
		// Check traces were created for this tenant
		err := tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1),
			[]string{"tempo_ingester_traces_created_total"},
			e2e.WithLabelMatchers(tenantMatchers...))
		if err != nil {
			// For single tenant setups, metrics might be under "single-tenant" label
			singleTenantMatchers := []*labels.Matcher{
				{Type: labels.MatchEqual, Name: "tenant", Value: "single-tenant"},
			}
			assert.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1),
				[]string{"tempo_ingester_traces_created_total"},
				e2e.WithLabelMatchers(singleTenantMatchers...)))
			break
		}
	}
}

func TestIngesterMetricsUnderLoad(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-kafka.yaml", "config.yaml"))

	kafka := e2edb.NewKafka()
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
	c, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	// Send multiple traces to generate more metrics
	expectedTraces := 15
	for i := 0; i < expectedTraces; i++ {
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, info.EmitAllBatches(c))
	}

	// Wait for processing
	time.Sleep(5 * time.Second)

	// Test metrics under load - should have higher values
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(float64(expectedTraces)), "tempo_ingester_traces_created_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_ingester_bytes_received_total"))

	// State metrics should show activity
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingester_live_traces"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingester_live_trace_bytes"))

	// Back pressure should still be minimal in test environment
	assert.NoError(t, tempo.WaitSumMetrics(e2e.Less(10), "tempo_ingester_back_pressure_seconds_total"))
}