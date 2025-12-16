package deployments

import (
	"testing"
	"time"

	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestFailureModes(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
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

		require.Eventually(t, func() bool {
			_, err := apiClient.QueryTraceV2(info.HexID()) // todo: seen this fail with transient errors connecting to the shutdown livestore. are our retry settings well configured?
			t.Logf("query trace v2 error: %v", err)
			return err == nil
		}, 10*time.Second, 100*time.Millisecond)

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
