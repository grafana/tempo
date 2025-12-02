package ingest

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

// jpe - remove

// TestLiveStoreWithHarness demonstrates how to use the new test harness.
// This is a refactored version of TestLiveStore that uses the harness.
func TestLiveStoreWithHarness(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// Use the test harness to set up all components
	util.WithTempoHarness(t, s, util.TestHarnessConfig{
		ConfigOverlay: "config-live-store.yaml",
		// LiveStorePairs defaults to 1 (2 instances: zone-a-0, zone-b-0)
	}, func(h *util.TempoHarness) {
		// Create and emit trace
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, info.EmitAllBatches(h.JaegerExporter))

		expected, err := info.ConstructTraceFromEpoch()
		require.NoError(t, err)

		// Test metrics
		require.NoError(t, h.Distributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

		// Wait for trace to be processed by live store (check first live store in zone-a)
		require.NoError(t, h.LiveStores[0].WaitSumMetrics(e2e.Between(1, 25), "tempo_live_store_records_processed_total"))
	})
}

// TestAPIWithHarness demonstrates testing API endpoints with the harness.
func TestAPIWithHarness(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	util.WithTempoHarness(t, s, util.TestHarnessConfig{
		ConfigOverlay: "config-live-store.yaml",
		// LiveStorePairs defaults to 1 (2 instances)
	}, func(h *util.TempoHarness) {
		// Create and emit trace
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, info.EmitAllBatches(h.JaegerExporter))

		// Wait for trace to be ingested
		time.Sleep(10 * time.Second)

		// Test trace query by ID
		tr, err := h.HTTPClient.QueryTrace(info.HexID())
		require.NoError(t, err)
		require.NotNil(t, tr)
		require.Greater(t, int(util.SpanCount(tr)), 0)

		// Test trace query by ID v2
		resp, err := h.HTTPClient.QueryTraceV2(info.HexID())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Trace)
		require.Greater(t, int(util.SpanCount(resp.Trace)), 0)

		// Test search tags v1
		tagsResp, err := h.HTTPClient.SearchTags()
		require.NoError(t, err)
		require.NotNil(t, tagsResp)
		require.Greater(t, len(tagsResp.TagNames), 0)

		// Test search tag values v1
		tagValuesResp, err := h.HTTPClient.SearchTagValues("service.name")
		require.NoError(t, err)
		require.NotNil(t, tagValuesResp)
		require.Greater(t, len(tagValuesResp.TagValues), 0)

		// Test search tags v2
		tagsV2Resp, err := h.HTTPClient.SearchTagsV2()
		require.NoError(t, err)
		require.NotNil(t, tagsV2Resp)
		total := 0
		for _, sc := range tagsV2Resp.Scopes {
			total += len(sc.Tags)
		}
		require.Greater(t, total, 0)

		// Test search tag values v2
		tagValuesV2Resp, err := h.HTTPClient.SearchTagValuesV2("resource.service.name", "")
		require.NoError(t, err)
		require.NotNil(t, tagValuesV2Resp)
		require.Greater(t, len(tagValuesV2Resp.TagValues), 0)

		// Test metrics query range
		qr, err := h.HTTPClient.MetricsQueryRange("{} | count_over_time()", 0, 0, "", 0)
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

// TestWithCustomLiveStoreConfig demonstrates using custom live store arguments and multiple pairs.
func TestWithCustomLiveStoreConfig(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	util.WithTempoHarness(t, s, util.TestHarnessConfig{
		ConfigOverlay:      "config-live-store.yaml",
		LiveStorePairs:     2, // Start 2 pairs = 4 instances (zone-a-0, zone-b-0, zone-a-1, zone-b-1)
		ExtraLiveStoreArgs: []string{"-live-store.complete-block-timeout=1s"},
	}, func(h *util.TempoHarness) {
		// Your test logic here
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, info.EmitAllBatches(h.JaegerExporter))

		// Verify we have 4 live stores
		require.Equal(t, 4, len(h.LiveStores))

		// Verify trace was processed by first live store
		require.NoError(t, h.LiveStores[0].WaitSumMetrics(e2e.Equals(1), "tempo_live_store_traces_created_total"))
	})
}

// TestWithBlockBuilders demonstrates using block builders.
func TestWithBlockBuilders(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	util.WithTempoHarness(t, s, util.TestHarnessConfig{
		ConfigOverlay:     "config-kafka.yaml",
		BlockBuilderCount: 2, // Start 2 block builders
	}, func(h *util.TempoHarness) {
		// Send trace
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, info.EmitAllBatches(h.JaegerExporter))

		expected, err := info.ConstructTraceFromEpoch()
		require.NoError(t, err)

		// Test distributor metrics
		require.NoError(t, h.Distributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

		// Wait for block builder to flush block (check first block builder)
		require.NoError(t, h.BlockBuilders[0].WaitSumMetricsWithOptions(
			e2e.Equals(1),
			[]string{"tempo_block_builder_flushed_blocks"},
			e2e.WaitMissingMetrics,
		))
	})
}

// TestWithMetricsGenerator demonstrates using metrics generator.
func TestWithMetricsGenerator(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	util.WithTempoHarness(t, s, util.TestHarnessConfig{
		ConfigOverlay:          "config-metrics-generator.yaml",
		EnableMetricsGenerator: true, // Enable metrics generator and Prometheus
	}, func(h *util.TempoHarness) {
		// Send trace
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, info.EmitAllBatches(h.JaegerExporter))

		// Verify metrics generator received spans
		require.NoError(t, h.MetricsGenerator.WaitSumMetrics(
			e2e.Greater(0),
			"tempo_metrics_generator_spans_received_total",
		))

		// Verify Prometheus is running and accessible
		require.NotNil(t, h.Prometheus)
	})
}
