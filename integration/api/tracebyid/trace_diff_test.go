package tracebyid

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafana/tempo/v3/integration/util"
	"github.com/grafana/tempo/v3/pkg/model/tracediff"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/v3/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestTraceDiff(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		infos := tempoUtil.NewTraceInfos(time.Now(), 2, "")
		for _, info := range infos {
			require.NoError(t, h.WriteTraceInfo(info, ""))
		}

		h.WaitTracesQueryable(t, len(infos))
		callTraceDiffAndAssert(t, h, infos[0], infos[1])

		h.WaitTracesWrittenToBackend(t, len(infos))
		h.ForceBackendQuerying(t)
		callTraceDiffAndAssert(t, h, infos[0], infos[1])
	})
}

func callTraceDiffAndAssert(t *testing.T, h *util.TempoHarness, base, compare *tempoUtil.TraceInfo) {
	t.Helper()

	// Default request (no format): the bare trace-patch-v0 document.
	respBody := postTraceDiff(t, h, fmt.Sprintf(`{"base":{"traceId":"%s"},"compare":{"traceId":"%s"}}`, base.HexID(), compare.HexID()))

	var patch tracediff.Result
	require.NoError(t, json.Unmarshal(respBody, &patch))
	assertTraceDiffPatch(t, &patch, base, compare)

	// Explicit composed request: the summary with the full patch embedded
	// because the vulture traces fit the patch budget.
	respBody = postTraceDiff(t, h, fmt.Sprintf(`{"base":{"traceId":"%s"},"compare":{"traceId":"%s"},"format":"trace-summary-v0-composed"}`, base.HexID(), compare.HexID()))

	var composed tracediff.ComposedResult
	require.NoError(t, json.Unmarshal(respBody, &composed))
	require.Equal(t, tracediff.VersionTraceSummaryV0Composed, composed.Version)
	require.NotNil(t, composed.Summary)
	require.Equal(t, tracediff.VersionTraceSummaryV0Native, composed.Summary.Version)
	require.Nil(t, composed.PatchOmitted)

	var embeddedPatch tracediff.Result
	require.NoError(t, json.Unmarshal(composed.Patch, &embeddedPatch))
	assertTraceDiffPatch(t, &embeddedPatch, base, compare)
}

func postTraceDiff(t *testing.T, h *util.TempoHarness, body string) []byte {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, h.BaseURL()+"/api/v2/traces/diff", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))
	return respBody
}

func assertTraceDiffPatch(t *testing.T, result *tracediff.Result, base, compare *tempoUtil.TraceInfo) {
	t.Helper()

	require.Equal(t, tracediff.VersionTracePatchV0, result.Version)
	require.Equal(t, base.HexID(), result.Base.TraceID)
	require.Equal(t, compare.HexID(), result.Compare.TraceID)

	baseTrace, err := base.ConstructTraceFromEpoch()
	require.NoError(t, err)
	compareTrace, err := compare.ConstructTraceFromEpoch()
	require.NoError(t, err)
	require.Equal(t, countTraceDiffSpans(baseTrace), result.Base.SpanCount)
	require.Equal(t, countTraceDiffSpans(compareTrace), result.Compare.SpanCount)
	require.Equal(t, result.Base.SpanCount, result.Stats.SpanCountA)
	require.Equal(t, result.Compare.SpanCount, result.Stats.SpanCountB)
}

func countTraceDiffSpans(trace *tempopb.Trace) int {
	if trace == nil {
		return 0
	}
	var count int
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			count += len(ss.Spans)
		}
	}
	return count
}
