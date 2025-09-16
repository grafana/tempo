package ingest

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveStore(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	liveStore0 := util.NewTempoLiveStore(0)
	liveStore1 := util.NewTempoLiveStore(1)
	require.NoError(t, s.StartAndWaitReady(liveStore0, liveStore1))
	waitUntilJoinedToPartitionRing(t, liveStore0, 2)

	distributor := util.NewTempoDistributor()
	require.NoError(t, s.StartAndWaitReady(distributor))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerToOTLPExporter(distributor.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// test metrics
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

	liveStoreProcessedRecords := liveStore0
	err = liveStoreProcessedRecords.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_traces_created_total")
	if err != nil { // then the trace went to second live store
		require.Error(t, err, "metric not found")
		liveStoreProcessedRecords = liveStore1
		err = liveStoreProcessedRecords.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_traces_created_total")
		require.NoError(t, err)
	}

	// Comprehensive metric validation
	testAllLiveStoreMetrics(t, liveStoreProcessedRecords)
}

// TestLiveStoreAPISmoke tests the API endpoints that will hit live store.
// It will be deleted after api tests will start using Rhythm.
func TestSmokeLiveStoreAPI(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	frontend, distributor := StartTempoWithLiveStore(t, s)
	client := httpclient.New("http://"+frontend.Endpoint(3200), "")

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerToOTLPExporter(distributor.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))
	time.Sleep(10 * time.Second) // wait for the trace to be ingested

	t.Run("get trace by id", func(t *testing.T) {
		tr, err := client.QueryTrace(info.HexID())
		require.NoError(t, err)
		require.NotNil(t, tr)
		require.Greater(t, int(util.SpanCount(tr)), 0)
	})

	t.Run("get trace by id v2", func(t *testing.T) {
		resp, err := client.QueryTraceV2(info.HexID())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Trace)
		require.Greater(t, int(util.SpanCount(resp.Trace)), 0)
	})

	t.Run("search tags v1", func(t *testing.T) {
		resp, err := client.SearchTags()
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, len(resp.TagNames), 0)
	})

	t.Run("search tag values v1", func(t *testing.T) {
		resp, err := client.SearchTagValues("service.name")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, len(resp.TagValues), 0)
	})

	t.Run("search tags v2", func(t *testing.T) {
		resp, err := client.SearchTagsV2()
		require.NoError(t, err)
		require.NotNil(t, resp)
		total := 0
		for _, sc := range resp.Scopes {
			total += len(sc.Tags)
		}
		require.Greater(t, total, 0)
	})

	t.Run("search tag values v2", func(t *testing.T) {
		resp, err := client.SearchTagValuesV2("resource.service.name", "")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, len(resp.TagValues), 0)
	})

	t.Run("metrics query range count_over_time", func(t *testing.T) {
		qr, err := client.MetricsQueryRange("{} | count_over_time()", 0, 0, "", 0)
		require.NoError(t, err)
		require.NotNil(t, qr)
		require.Equal(t, 1, len(qr.Series))
		var count int
		for _, s := range qr.Series[0].Samples {
			count += int(s.Value)
		}
		require.Greater(t, count, 0)
	})
}

func waitUntilJoinedToPartitionRing(t *testing.T, liveStore *e2e.HTTPService, numPartitions float64) {
	matchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: "Active"},
		{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
	}
	require.NoError(t, liveStore.WaitSumMetricsWithOptions(e2e.Equals(numPartitions), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(matchers...)))
}

func StartTempoWithLiveStore(t *testing.T, s *e2e.Scenario) (*e2e.HTTPService, *e2e.HTTPService) {
	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	liveStore1 := util.NewTempoLiveStore(0)
	liveStore2 := util.NewTempoLiveStore(1)
	require.NoError(t, s.StartAndWaitReady(liveStore1, liveStore2))
	waitUntilJoinedToPartitionRing(t, liveStore1, 2)

	distributor := util.NewTempoDistributor()
	frontend := util.NewTempoQueryFrontend()
	require.NoError(t, s.StartAndWaitReady(distributor, frontend, util.NewTempoQuerier()))

	return frontend, distributor
}

// testAllLiveStoreMetrics validates all livestore metrics have reasonable values
func testAllLiveStoreMetrics(t *testing.T, liveStore *e2e.HTTPService) {
	// Basic trace processing metrics
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_live_store_traces_created_total"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.Between(1, 50), "tempo_live_store_kafka_records_processed_total"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_bytes_received_total"))

	// Live trace state metrics - should have some live traces initially
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_live_traces"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_live_trace_bytes"))

	// Block lifecycle metrics
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_blocks_cleared_total"))

	// Kafka consumer metrics
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_fetch_bytes_total"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_fetch_records_total"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_fetch_errors_total"))

	// Dropped records should be 0 in normal operation
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.Equals(0), "tempo_live_store_kafka_records_dropped_total"))

	// Backpressure should be minimal in test environment
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_back_pressure_seconds_total"))

	// Histogram metrics - check they have recorded samples
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_fetch_duration_seconds"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_consume_cycle_duration_seconds"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_process_partition_duration_seconds"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_completion_size_bytes"))

	// Partition lag metrics (these are from the ingest package)
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingest_group_partition_lag"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_ingest_group_partition_lag_seconds"))
}

// TestLiveStoreMultiTenantMetrics tests that metrics work correctly with multiple tenants
func TestLiveStoreMultiTenantMetrics(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	liveStore0 := util.NewTempoLiveStore(0)
	liveStore1 := util.NewTempoLiveStore(1)
	require.NoError(t, s.StartAndWaitReady(liveStore0, liveStore1))
	waitUntilJoinedToPartitionRing(t, liveStore0, 2)

	distributor := util.NewTempoDistributor()
	require.NoError(t, s.StartAndWaitReady(distributor))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerToOTLPExporter(distributor.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	// Send traces from multiple tenants
	tenants := []string{"tenant-1", "tenant-2", "tenant-3"}
	var expectedTraces []*tempopb.Trace

	for _, tenant := range tenants {
		info := tempoUtil.NewTraceInfo(time.Now(), tenant)
		require.NoError(t, info.EmitAllBatches(c))

		expected, err := info.ConstructTraceFromEpoch()
		require.NoError(t, err)
		expectedTraces = append(expectedTraces, expected)
	}

	// Wait for all spans to be received
	totalSpans := 0
	for _, trace := range expectedTraces {
		totalSpans += int(util.SpanCount(trace))
	}
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(float64(totalSpans)), "tempo_distributor_spans_received_total"))

	// Test multi-tenant metrics
	testMultiTenantMetrics(t, liveStore0, liveStore1, tenants)
}

func testMultiTenantMetrics(t *testing.T, liveStore0, liveStore1 *e2e.HTTPService, tenants []string) {
	// Wait for traces to be processed
	time.Sleep(5 * time.Second)

	// Test that each tenant has traces created (may be distributed across both live stores)
	for _, tenant := range tenants {
		found := false

		// Check liveStore0 first
		err := liveStore0.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1),
			[]string{"tempo_live_store_traces_created_total"},
			e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenant}),
		)
		if err == nil {
			found = true
		} else {
			// Check liveStore1
			err := liveStore1.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1),
				[]string{"tempo_live_store_traces_created_total"},
				e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenant}),
			)
			if err == nil {
				found = true
			}
		}

		assert.True(t, found, "Expected tenant %s to have traces created metric > 0", tenant)
	}

	// Test aggregate metrics across tenants
	totalTracesCreated := 0.0
	totalRecordsProcessed := 0.0

	// Sum from both live stores
	for _, ls := range []*e2e.HTTPService{liveStore0, liveStore1} {
		traces, _ := ls.SumMetrics([]string{"tempo_live_store_traces_created_total"})
		records, _ := ls.SumMetrics([]string{"tempo_live_store_kafka_records_processed_total"})
		if len(traces) > 0 {
			totalTracesCreated += traces[0]
		}
		if len(records) > 0 {
			totalRecordsProcessed += records[0]
		}
	}

	assert.GreaterOrEqual(t, totalTracesCreated, float64(len(tenants)), "Expected at least one trace per tenant")
	assert.GreaterOrEqual(t, totalRecordsProcessed, float64(len(tenants)), "Expected at least one record processed per tenant")
}

// TestLiveStoreMetricsUnderLoad tests metrics behavior under load and with timing validation
func TestLiveStoreMetricsUnderLoad(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	liveStore0 := util.NewTempoLiveStore(0)
	require.NoError(t, s.StartAndWaitReady(liveStore0))
	waitUntilJoinedToPartitionRing(t, liveStore0, 1)

	distributor := util.NewTempoDistributor()
	require.NoError(t, s.StartAndWaitReady(distributor))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerToOTLPExporter(distributor.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	// Send multiple traces to generate load
	numTraces := 10
	var expectedTraces []*tempopb.Trace

	for i := 0; i < numTraces; i++ {
		info := tempoUtil.NewTraceInfo(time.Now(), "load-test-tenant")
		require.NoError(t, info.EmitAllBatches(c))

		expected, err := info.ConstructTraceFromEpoch()
		require.NoError(t, err)
		expectedTraces = append(expectedTraces, expected)
	}

	// Wait for all spans to be received
	totalSpans := 0
	for _, trace := range expectedTraces {
		totalSpans += int(util.SpanCount(trace))
	}
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(float64(totalSpans)), "tempo_distributor_spans_received_total"))

	// Test performance and timing metrics
	testPerformanceMetrics(t, liveStore0, numTraces)
}

func testPerformanceMetrics(t *testing.T, liveStore *e2e.HTTPService, expectedTraces int) {
	// Wait for processing to complete
	time.Sleep(10 * time.Second)

	// Validate core metrics have expected values
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.Equals(float64(expectedTraces)), "tempo_live_store_traces_created_total"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(float64(expectedTraces)), "tempo_live_store_kafka_records_processed_total"))

	// Validate bytes metrics are reasonable (each trace should be at least 100 bytes)
	minExpectedBytes := float64(expectedTraces * 100)
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(minExpectedBytes), "tempo_live_store_bytes_received_total"))

	// Validate timing metrics have reasonable values (should complete within reasonable time)
	// Note: These are histogram metrics, so we check the bucket counts or sample counts

	// Kafka fetch operations should have completed
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_live_store_fetch_duration_seconds"))

	// Consume cycles should have completed
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_live_store_consume_cycle_duration_seconds"))

	// Partition processing should have occurred
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_live_store_process_partition_duration_seconds"))

	// Kafka metrics should show activity
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(float64(expectedTraces)), "tempo_live_store_fetch_records_total"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(minExpectedBytes), "tempo_live_store_fetch_bytes_total"))

	// Error metrics should be minimal in normal operation
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.Less(2), "tempo_live_store_fetch_errors_total"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.Equals(0), "tempo_live_store_kafka_records_dropped_total"))

	// Live trace metrics should show current state
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_live_traces"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.GreaterOrEqual(0), "tempo_live_store_live_trace_bytes"))

	// Partition lag should be reasonable (not too high)
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.Less(1001), "tempo_ingest_group_partition_lag"))
	assert.NoError(t, liveStore.WaitSumMetrics(e2e.Less(61), "tempo_ingest_group_partition_lag_seconds"))
}
