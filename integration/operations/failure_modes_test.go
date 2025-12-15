package deployments

import (
	"testing"
	"time"

	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

const queryBackendConfig = "config-failure-modes.yaml"

func TestFailureModes(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		Components:    util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
		ConfigOverlay: queryBackendConfig,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		apiClient := h.APIClientHTTP("")
		util.QueryAndAssertTrace(t, apiClient, info)

		// stop one live store. should still be queryable
		liveStoreA := h.Services[util.ServiceLiveStoreZoneA]
		err := liveStoreA.Stop()
		require.NoError(t, err)
		util.QueryAndAssertTrace(t, apiClient, info) // todo: seen this fail with transient errors. are our retry settings well configured?

		// stop the second live store. querying should fail
		liveStoreB := h.Services[util.ServiceLiveStoreZoneB]
		err = liveStoreB.Stop()
		require.NoError(t, err)
		_, err = apiClient.QueryTraceV2(info.HexID())
		require.Error(t, err)

		h.WaitTracesWrittenToBackend(t, 1)

		// stop the block builder now that the trace is in the backend. ingestion should still work
		blockBuilder := h.Services[util.ServiceBlockBuilder]
		err = blockBuilder.Stop()
		require.NoError(t, err)
		require.NoError(t, h.WriteTraceInfo(tempoUtil.NewTraceInfo(time.Now(), ""), ""))

		// restart the query frontend to do backend querying only. querying should work again
		h.ForceBackendQuerying(t)
		apiClient = h.APIClientHTTP("")
		_, err = apiClient.SearchTraceQLWithRange("{}", time.Now().Add(-time.Hour).Unix(), time.Now().Unix())
		require.NoError(t, err)
	})
}
