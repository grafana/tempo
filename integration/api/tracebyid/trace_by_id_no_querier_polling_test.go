package tracebyid

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/integration/util"
	tempoquerier "github.com/grafana/tempo/v3/modules/querier"
	tempoUtil "github.com/grafana/tempo/v3/pkg/util"
)

// TestTraceByIDWithoutQuerierPolling verifies that queriers find traces in backend blocks
// using the blocks sent by the query-frontend, without polling the blocklist themselves.
func TestTraceByIDWithoutQuerierPolling(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: "config-querier-no-polling.yaml",
		Components:    util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		countTraces := 5
		infos := tempoUtil.NewTraceInfos(time.Now(), countTraces, "")
		for _, info := range infos {
			require.NoError(t, h.WriteTraceInfo(info, ""))
		}

		h.WaitTracesQueryable(t, countTraces)
		h.WaitTracesWrittenToBackend(t, countTraces)

		querier := h.Services[util.ServiceQuerier]

		// the querier never polled, so it can only find backend traces through the blocks the frontend sends
		polls, err := querier.SumMetrics([]string{"tempodb_blocklist_poll_duration_seconds"}, e2e.WithMetricCount)
		require.NoError(t, err)
		require.Equal(t, float64(0), polls[0])

		// wait until the live-stores dropped the traces, so a found trace must come from the backend
		for _, info := range infos {
			require.Eventually(t, func() bool {
				resp, err := http.Get("http://" + querier.HTTPEndpoint() + "/querier/api/traces/" + info.HexID() + "?mode=ingesters")
				if err != nil {
					return false
				}
				defer resp.Body.Close()
				return resp.StatusCode == http.StatusNotFound
			}, time.Minute, time.Second, "trace %s still in live-stores", info.HexID())
		}

		apiClient := h.APIClientHTTP("")
		for _, info := range infos {
			util.QueryAndAssertTrace(t, apiClient, info)
		}

		// a job without blocks, as sent by an older query-frontend, is rejected instead of silently finding nothing
		resp, err := http.Get("http://" + querier.HTTPEndpoint() + "/querier/api/traces/" + infos[0].HexID() + "?mode=blocks")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), tempoquerier.ErrTraceByIDBlocksRequired.Error())
	})
}
