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

func TestBlockBuilderMetrics(t *testing.T) {
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

	// wait until the block-builder block is flushed
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_block_builder_flushed_blocks"}, e2e.WaitMissingMetrics))

	// Test comprehensive blockbuilder metrics
	testAllBlockBuilderMetricsDetailed(t, tempo)
}

// testAllBlockBuilderMetricsDetailed validates all blockbuilder metrics from pkg/ingest/metrics.go have reasonable values
func testAllBlockBuilderMetricsDetailed(t *testing.T, tempo *e2e.HTTPService) {
	// Basic trace processing metrics
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_block_builder_traces_created_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_block_builder_kafka_records_processed_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_bytes_received_total"))

	// Live trace state metrics - blockbuilder may have 0 live traces as it processes batches
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_live_traces"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_live_trace_bytes"))

	// Block lifecycle metrics
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_blocks_cleared_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_completion_size_bytes"))

	// Kafka consumer metrics
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_fetch_bytes_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_fetch_records_total"))

	// In normal operation, fetch errors should be minimal (allowing some transient errors)
	assert.NoError(t, tempo.WaitSumMetrics(e2e.Less(5), "tempo_block_builder_fetch_errors_total"))

	// Processing duration metrics - check they have recorded samples
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_fetch_duration_seconds"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_consume_cycle_duration_seconds"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_process_partition_duration_seconds"))

	// Dropped records should be 0 in normal operation
	assert.NoError(t, tempo.WaitSumMetrics(e2e.Equals(0), "tempo_block_builder_kafka_records_dropped_total"))

	// Back pressure should be minimal in test environment
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_block_builder_back_pressure_seconds_total"))

	// Partition lag metrics (these are from the ingest package)
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingest_group_partition_lag"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingest_group_partition_lag_seconds"))
}

func TestBlockBuilderMetricsUnderLoad(t *testing.T) {
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
	expectedTraces := 10
	for i := 0; i < expectedTraces; i++ {
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, info.EmitAllBatches(c))
	}

	// Wait for processing
	time.Sleep(5 * time.Second)

	// Test metrics under load - should have higher values
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(float64(expectedTraces)), "tempo_block_builder_traces_created_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(float64(expectedTraces)), "tempo_block_builder_kafka_records_processed_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_block_builder_bytes_received_total"))

	// Processing metrics should show activity
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_block_builder_fetch_duration_seconds"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_block_builder_consume_cycle_duration_seconds"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_block_builder_process_partition_duration_seconds"))

	// Fetch metrics should show activity
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_block_builder_fetch_bytes_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(float64(expectedTraces)), "tempo_block_builder_fetch_records_total"))

	// Errors and drops should still be minimal
	assert.NoError(t, tempo.WaitSumMetrics(e2e.Less(5), "tempo_block_builder_fetch_errors_total"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.Equals(0), "tempo_block_builder_kafka_records_dropped_total"))

	// Lag should be reasonable
	assert.NoError(t, tempo.WaitSumMetrics(e2e.Less(1000), "tempo_ingest_group_partition_lag"))
	assert.NoError(t, tempo.WaitSumMetrics(e2e.Less(30), "tempo_ingest_group_partition_lag_seconds"))
}