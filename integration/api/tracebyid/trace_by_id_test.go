package tracebyid

import (
	"testing"
	"time"

	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestTraceByIDandTraceQL(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
		Backends:   util.BackendObjectStorageAll, // runs basic querying against all 3 object storage backends. no need to replicate for every test.
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		countTraces := 10
		infos := tempoUtil.NewTraceInfos(time.Now(), countTraces, "")
		for _, info := range infos {
			require.NoError(t, h.WriteTraceInfo(info, ""))
		}

		h.WaitTracesQueryable(t, countTraces)

		grpcClient, ctx, err := h.APIClientGRPC("")
		require.NoError(t, err)
		apiClient := h.APIClientHTTP("")

		now := time.Now()
		for _, i := range infos {
			util.QueryAndAssertTrace(t, apiClient, i)
			util.SearchTraceQLAndAssertTraceWithRange(t, apiClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
			util.SearchStreamAndAssertTrace(t, ctx, grpcClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
		}

		h.WaitTracesWrittenToBackend(t, countTraces)
		h.ForceBackendQuerying(t)

		grpcClient, ctx, err = h.APIClientGRPC("")
		require.NoError(t, err)
		apiClient = h.APIClientHTTP("")

		// Assert tags on storage backend
		for _, i := range infos {
			util.QueryAndAssertTrace(t, apiClient, i)
			util.SearchTraceQLAndAssertTraceWithRange(t, apiClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
			util.SearchStreamAndAssertTrace(t, ctx, grpcClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
		}
	})
}
