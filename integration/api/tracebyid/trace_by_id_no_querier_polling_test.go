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

// TestTraceByIDWithoutQuerierPolling checks that queriers find backend traces using only the blocks the query-frontend sends.
func TestTraceByIDWithoutQuerierPolling(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: "config-querier-no-polling.yaml",
		Components:    util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying | util.ComponentsBackendWork,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// separate flushes give the backend-worker blocks to compact
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

		// the frontend's blocklist lags flushes and compaction, so a gap between views shows up as a missing trace
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
			// give the frontend's view a few polls to catch up with compaction
			if !settledAt.IsZero() && time.Since(settledAt) > 10*time.Second {
				break
			}

			require.True(t, time.Now().Before(deadline), "traces did not leave the live-stores or blocks were not compacted")
			time.Sleep(500 * time.Millisecond)
		}

		// without its own blocklist, the querier can only have used the frontend's blocks
		polls, err := querier.SumMetrics([]string{"tempodb_blocklist_poll_duration_seconds"}, e2e.WithMetricCount)
		require.NoError(t, err)
		require.Equal(t, float64(0), polls[0])

		// an older query-frontend sends no blocks, which must fail instead of finding nothing
		resp, err := http.Get("http://" + querier.HTTPEndpoint() + "/querier/api/traces/" + infos[0].HexID() + "?mode=blocks")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), tempoquerier.ErrTraceByIDBlocksRequired.Error())
	})
}
