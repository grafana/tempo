package deployments

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

const configSingleBinaryFlush = "config-single-binary-flush.yaml"

func TestSingleBinaryIngestsAndFlushesToBackend(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		DeploymentMode: util.DeploymentModeSingleBinary,
		ConfigOverlay:  configSingleBinaryFlush,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		tempo := h.Services[util.ServiceDistributor]
		require.NoError(t, tempo.WaitSumMetricsWithOptions(
			e2e.GreaterOrEqual(float64(1)),
			[]string{"tempo_live_store_traces_created_total"},
			e2e.WaitMissingMetrics,
		))

		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		require.NoError(t, tempo.WaitSumMetricsWithOptions(
			e2e.GreaterOrEqual(float64(1)),
			[]string{"tempo_live_store_local_blocks_flushed_total"},
			e2e.WaitMissingMetrics,
		))
		h.WaitTracesWrittenToBackend(t, 1)
	})
}
