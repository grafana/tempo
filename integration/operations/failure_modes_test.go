package deployments

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestFailureModes(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components:         util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
		DisableParallelism: true, // for unknown reasons this test is flakey when run in parallel. unsure if there's some kind of cross test interference i'm unaware of? or a timing thing?
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		apiClient := h.APIClientHTTP("")
		util.QueryAndAssertTrace(t, apiClient, info)

		// stop one live store. data should still be queryable
		liveStoreB := h.Services[util.ServiceLiveStoreZoneB]
		err := liveStoreB.Stop()
		require.NoError(t, err)

		// get /live-store/ring on querier and dump to stdout
		ring, err := http.Get("http://" + h.Services[util.ServiceQuerier].Endpoint(3200) + "/live-store/ring")
		require.NoError(t, err)
		body, err := io.ReadAll(ring.Body)
		require.NoError(t, err)
		fmt.Println(string(body))

		_, err = apiClient.QueryTraceV2(info.HexID()) // todo: occassional failures here when running in parallel. disabled above
		require.NoError(t, err)

		// stop the second live store. querying should fail
		liveStoreA := h.Services[util.ServiceLiveStoreZoneA]
		err = liveStoreA.Stop()
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
