package search

import (
	"testing"
	"time"

	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestSearchTraceQL is the basic end-to-end TraceQL search test: write a trace, then find it
// with a TraceQL query. Add new plain TraceQL search tests here.
func TestSearchTraceQL(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		apiClient := h.APIClientHTTP("")
		util.SearchTraceQLAndAssertTrace(t, apiClient, info)
	})
}
