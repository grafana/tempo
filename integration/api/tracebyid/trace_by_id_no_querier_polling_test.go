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
		Components:    util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying | util.ComponentsBackendWork,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// two batches flushed separately give at least two blocks for the backend-worker to compact
		countTraces := 5
		var infos []*tempoUtil.TraceInfo
		for batch := 1; batch <= 2; batch++ {
			for _, info := range tempoUtil.NewTraceInfos(time.Now(), countTraces, "") {
				require.NoError(t, h.WriteTraceInfo(info, ""))
				infos = append(infos, info)
			}
			h.WaitTracesQueryable(t, batch*countTraces)
			h.WaitTracesWrittenToBackend(t, batch*countTraces)
		}

		querier := h.Services[util.ServiceQuerier]
		worker := h.Services[util.ServiceBackendWorker]
		apiClient := h.APIClientHTTP("")

		inLiveStores := func(info *tempoUtil.TraceInfo) bool {
			resp, err := http.Get("http://" + querier.HTTPEndpoint() + "/querier/api/traces/" + info.HexID() + "?mode=ingesters")
			require.NoError(t, err)
			defer resp.Body.Close()
			return resp.StatusCode != http.StatusNotFound
		}

		// query through the handoff from live-stores to the backend and through compaction, the frontend's
		// blocklist view lags both, so any gap between the views shows up as a missing trace here
		var settledAt time.Time
		deadline := time.Now().Add(2 * time.Minute)
		for {
			for _, info := range infos {
				util.QueryAndAssertTrace(t, apiClient, info)
			}

			if settledAt.IsZero() {
				compacted, err := worker.SumMetrics([]string{"tempodb_compaction_blocks_total"}, e2e.SkipMissingMetrics)
				require.NoError(t, err)
				dropped := true
				for _, info := range infos {
					if inLiveStores(info) {
						dropped = false
						break
					}
				}
				if dropped && compacted[0] > 0 {
					settledAt = time.Now()
				}
			}
			// keep querying for a few polls after settling, so the frontend's view catches up with compaction
			if !settledAt.IsZero() && time.Since(settledAt) > 10*time.Second {
				break
			}

			require.True(t, time.Now().Before(deadline), "traces did not leave the live-stores or blocks were not compacted")
			time.Sleep(500 * time.Millisecond)
		}

		// the querier never polled, so it can only find backend traces through the blocks the frontend sends
		polls, err := querier.SumMetrics([]string{"tempodb_blocklist_poll_duration_seconds"}, e2e.WithMetricCount)
		require.NoError(t, err)
		require.Equal(t, float64(0), polls[0])

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
