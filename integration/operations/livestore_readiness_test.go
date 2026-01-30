package deployments

import (
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestLiveStoreReadinessDefaultBehavior verifies that with readiness_target_lag=0 (default),
// the LiveStore becomes ready immediately without waiting
func TestLiveStoreReadinessDefaultBehavior(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying,
	}, func(h *util.TempoHarness) {
		liveStoreA := h.Services[util.ServiceLiveStoreZoneA]

		// With default config (readiness_target_lag=0), LiveStore should be ready immediately
		require.NoError(t, liveStoreA.WaitReady())

		// Verify /ready endpoint returns 200
		req, err := http.NewRequest("GET", "http://"+liveStoreA.Endpoint(3200)+"/ready", nil)
		require.NoError(t, err)
		httpResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 200, httpResp.StatusCode)

		// Verify tempo_live_store_ready metric is 1
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_ready"))
	})
}

// TestLiveStoreReadinessWithCatchUp verifies that readiness waiting works correctly
// and metrics are recorded
func TestLiveStoreReadinessWithCatchUp(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: "config-livestore-readiness-enabled.yaml",
		Components:    util.ComponentsRecentDataQuerying,
	}, func(h *util.TempoHarness) {
		liveStoreA := h.Services[util.ServiceLiveStoreZoneA]
		liveStoreB := h.Services[util.ServiceLiveStoreZoneB]

		// Stop liveStoreB to simplify the test
		require.NoError(t, liveStoreB.Stop())

		// Wait for LiveStore to be ready
		h.WaitTracesWritable(t)

		// Write some traces to create Kafka lag
		for i := 0; i < 5; i++ {
			info := tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, h.WriteTraceInfo(info, ""))
		}

		// Wait for traces to be processed
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.GreaterOrEqual(5), "tempo_live_store_traces_created_total"))

		// Stop the LiveStore
		require.NoError(t, liveStoreA.Stop())

		// Write more traces during downtime to create lag
		for i := 0; i < 3; i++ {
			require.NoError(t, h.WriteTraceInfo(tempoUtil.NewTraceInfo(time.Now(), ""), ""))
			time.Sleep(100 * time.Millisecond)
		}

		// Restart LiveStore
		require.NoError(t, liveStoreA.Start(h.TestScenario.NetworkName(), h.TestScenario.SharedDir()))

		// Wait for it to become ready (it should catch up)
		require.NoError(t, liveStoreA.WaitReady())

		// Verify /ready endpoint returns 200
		req, err := http.NewRequest("GET", "http://"+liveStoreA.Endpoint(3200)+"/ready", nil)
		require.NoError(t, err)
		httpResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 200, httpResp.StatusCode)

		// Verify tempo_live_store_ready metric is 1
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_ready"))

		// Verify catch_up_duration metric was recorded
		// The metric should have at least one observation
		metrics, err := liveStoreA.SumMetrics([]string{"tempo_live_store_catch_up_duration_seconds"})
		require.NoError(t, err)
		require.Greater(t, metrics[0], 0.0, "catch_up_duration should have been recorded")
	})
}

// TestLiveStoreReadinessMaxWaitTimeout verifies that LiveStore becomes ready
// after readiness_max_wait even if lag is still high
func TestLiveStoreReadinessMaxWaitTimeout(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: "config-livestore-readiness-timeout.yaml",
		Components:    util.ComponentsRecentDataQuerying,
	}, func(h *util.TempoHarness) {
		liveStoreA := h.Services[util.ServiceLiveStoreZoneA]
		liveStoreB := h.Services[util.ServiceLiveStoreZoneB]

		// Stop liveStoreB to simplify the test
		require.NoError(t, liveStoreB.Stop())

		// Wait for LiveStore to be ready initially
		h.WaitTracesWritable(t)

		// Write some traces
		for i := 0; i < 3; i++ {
			info := tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, h.WriteTraceInfo(info, ""))
		}

		// Wait for traces to be processed
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.GreaterOrEqual(3), "tempo_live_store_traces_created_total"))

		// Stop the LiveStore
		require.NoError(t, liveStoreA.Stop())

		// Write many traces during downtime to create significant lag
		// With readiness_target_lag=100ms and readiness_max_wait=5s,
		// the LiveStore should become ready after 5s even if lag is high
		for i := 0; i < 50; i++ {
			require.NoError(t, h.WriteTraceInfo(tempoUtil.NewTraceInfo(time.Now(), ""), ""))
			time.Sleep(200 * time.Millisecond) // Create lag that exceeds target
		}

		// Restart LiveStore
		startTime := time.Now()
		require.NoError(t, liveStoreA.Start(h.TestScenario.NetworkName(), h.TestScenario.SharedDir()))

		// It should become ready due to max_wait timeout (5s)
		require.NoError(t, liveStoreA.WaitReady())
		elapsed := time.Since(startTime)

		// Should have waited close to max_wait (5s), but not too long
		require.Less(t, elapsed, 15*time.Second, "should become ready within reasonable time")

		// Verify /ready endpoint returns 200
		req, err := http.NewRequest("GET", "http://"+liveStoreA.Endpoint(3200)+"/ready", nil)
		require.NoError(t, err)
		httpResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 200, httpResp.StatusCode)

		// Verify tempo_live_store_ready metric is 1
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_ready"))
	})
}

// TestLiveStoreReadinessRestartWithLag verifies restart scenario with accumulated Kafka lag
func TestLiveStoreReadinessRestartWithLag(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: "config-livestore-readiness-enabled.yaml",
		Components:    util.ComponentsRecentDataQuerying,
	}, func(h *util.TempoHarness) {
		liveStoreA := h.Services[util.ServiceLiveStoreZoneA]
		liveStoreB := h.Services[util.ServiceLiveStoreZoneB]

		// Stop liveStoreB to simplify the test
		require.NoError(t, liveStoreB.Stop())

		// Wait for initial readiness
		h.WaitTracesWritable(t)

		// Write initial traces
		for i := 0; i < 3; i++ {
			info := tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, h.WriteTraceInfo(info, ""))
		}

		// Wait for traces to be processed
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.GreaterOrEqual(3), "tempo_live_store_traces_created_total"))

		// Verify ready state before restart
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_ready"))

		// Stop LiveStore
		require.NoError(t, liveStoreA.Stop())

		// Write traces during downtime to accumulate lag
		for i := 0; i < 10; i++ {
			require.NoError(t, h.WriteTraceInfo(tempoUtil.NewTraceInfo(time.Now(), ""), ""))
			time.Sleep(100 * time.Millisecond)
		}

		// Restart LiveStore
		require.NoError(t, liveStoreA.Start(h.TestScenario.NetworkName(), h.TestScenario.SharedDir()))

		// Initially, LiveStore should not be ready (503) while catching up
		// Note: This check is timing-sensitive and might pass if catch-up is very fast
		req, err := http.NewRequest("GET", "http://"+liveStoreA.Endpoint(3200)+"/ready", nil)
		require.NoError(t, err)
		httpResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		// During catch-up, we might see 503
		if httpResp.StatusCode == 503 {
			t.Log("LiveStore correctly returns 503 during catch-up")
		}

		// Wait for it to become ready after catching up
		require.NoError(t, liveStoreA.WaitReady())

		// Verify /ready endpoint returns 200
		req, err = http.NewRequest("GET", "http://"+liveStoreA.Endpoint(3200)+"/ready", nil)
		require.NoError(t, err)
		httpResp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 200, httpResp.StatusCode)

		// Verify tempo_live_store_ready metric is 1
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_ready"))

		// Verify some traces have been processed
		require.NoError(t, liveStoreA.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_live_store_traces_created_total"))

		// Verify catch_up_duration metric was recorded
		metrics, err := liveStoreA.SumMetrics([]string{"tempo_live_store_catch_up_duration_seconds"})
		require.NoError(t, err)
		require.Greater(t, metrics[0], 0.0, "catch_up_duration should have been recorded")
	})
}
