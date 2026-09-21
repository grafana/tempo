package deployments

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

// capped by the e2e library's hardcoded `docker stop --time=30`, floored by block-builder finishing its consume cycle
const gracefulStopBudget = 20 * time.Second

// read path first, memberlist gossip seeds last, so nothing stops after its gossip peers are gone
var shutdownOrder = []string{
	util.ServiceQueryFrontend,
	util.ServiceQuerier,
	util.ServiceMetricsGenerator,
	util.ServiceDistributor,
	util.ServiceBackendWorker,
	util.ServiceBackendScheduler,
	util.ServiceBlockBuilder,
	util.ServiceLiveStoreZoneB,
	util.ServiceLiveStoreZoneA,
}

func TestReadPathShutsDownGracefully(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		// serving a query leaves a tenant queue behind on the frontend, which is the state
		// that made it ignore SIGTERM until docker killed it
		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		// querier first, so the frontend then has to drain with no querier available
		for _, name := range []string{util.ServiceQuerier, util.ServiceQueryFrontend} {
			start := time.Now()
			err := h.Services[name].Stop()
			elapsed := time.Since(start)

			// Stop returns the container's wait status, so exit status 137 here means SIGKILL
			require.NoError(t, err, "%s did not exit cleanly on SIGTERM", name)
			require.Less(t, elapsed, gracefulStopBudget, "%s took %s to stop", name, elapsed)
		}
	})
}

// single binary is the one mode where neither frontend nor querier enables blocklist polling
func TestSingleBinaryShutsDownGracefully(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		DeploymentMode: util.DeploymentModeSingleBinary,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		// every service name maps to the same container here
		tempo := h.Services[util.ServiceQueryFrontend]
		require.NoError(t, tempo.WaitSumMetricsWithOptions(
			e2e.GreaterOrEqual(float64(1)),
			[]string{"tempo_live_store_traces_created_total"},
			e2e.WaitMissingMetrics,
		))

		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		start := time.Now()
		err := tempo.Stop()
		elapsed := time.Since(start)

		require.NoError(t, err, "single binary did not exit cleanly on SIGTERM")
		require.Less(t, elapsed, gracefulStopBudget, "single binary took %s to stop", elapsed)
	})
}

// TestAllComponentsShutDownGracefully guards the bug class, not just the two deadlocks that prompted it.
func TestAllComponentsShutDownGracefully(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying |
			util.ComponentsMetricsGeneration | util.ComponentsBackendWork,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		// shut down components that have live state, since an idle process can stop cleanly
		// while a working one deadlocks
		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		type stopResult struct {
			name    string
			elapsed time.Duration
			err     error
		}

		// stop everything before asserting, so the first bad component does not hide the rest
		results := make([]stopResult, 0, len(shutdownOrder))
		for _, name := range shutdownOrder {
			svc := h.Services[name]
			require.NotNil(t, svc, "%s was not started, so this test covers less than it claims", name)

			start := time.Now()
			err := svc.Stop()
			results = append(results, stopResult{name: name, elapsed: time.Since(start), err: err})
		}

		for _, r := range results {
			t.Logf("shutdown %-22s %6.2fs err=%v", r.name, r.elapsed.Seconds(), r.err)
		}

		for _, r := range results {
			// Stop returns the container's wait status, so exit status 137 here means SIGKILL
			require.NoError(t, r.err, "%s did not exit cleanly on SIGTERM", r.name)
			require.Less(t, r.elapsed, gracefulStopBudget, "%s took %s to stop", r.name, r.elapsed)
		}
	})
}
