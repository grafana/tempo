package tracebyid

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestTraceByIDV2Filtering exercises the `q` spanset filter and its keep_hierarchy, match_depth
// and ancestor_depth options on the V2 trace-by-id endpoint end to end, on both the recent-data
// and backend querying paths.
func TestTraceByIDV2Filtering(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// trace shape: A(root) -> B -> C(status_code=500) -> E -> F, and A -> D(status_code=200).
		// only C matches the filter; B and A are its ancestors; E and F are its descendants;
		// D is an unrelated branch.
		traceID := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
		require.NoError(t, h.WriteTempoProtoTraces(buildFilterTestTrace(traceID), ""))
		h.WaitTracesQueryable(t, 1)

		hexID := tempoUtil.TraceIDToHexString(traceID)

		// spans are identified by the last byte of their id:
		//   1=A(root)  2=B  3=C(http.status_code=500)  4=D(http.status_code=200)  5=E  6=F
		// relative to C: ancestors are B at 1 hop then A at 2 hops; descendants are E at 1 hop
		// then F at 2 hops. Relative to D: A is its only ancestor, and it has no descendants.
		const (
			matchC  = "{ .http.status_code = 500 }" // matches C only
			matchCD = "{ .http.status_code > 0 }"   // matches C and D
		)

		// filterCases covers the interaction of keep_hierarchy, match_depth and ancestor_depth.
		// The two depth params default differently when absent — match_depth to 0 (no descendants),
		// ancestor_depth to -1 (unbounded) — so absent and explicit values are both asserted.
		filterCases := []struct {
			name   string
			query  string
			params map[string]string
			expect []byte
			// a filter only reports COMPLETE when it dropped nothing, so this is the exception.
			wantComplete bool
		}{
			// backward-compat baseline: no depth params at all must behave exactly as the endpoint
			// did before match_depth and ancestor_depth existed — the matched span and nothing else.
			{name: "defaults_match_only", query: matchC, expect: []byte{3}},

			// match_depth alone, without keep_hierarchy.
			{name: "match_depth_0", query: matchC, params: map[string]string{"match_depth": "0"}, expect: []byte{3}},
			{name: "match_depth_1", query: matchC, params: map[string]string{"match_depth": "1"}, expect: []byte{3, 5}},
			{name: "match_depth_2", query: matchC, params: map[string]string{"match_depth": "2"}, expect: []byte{3, 5, 6}},
			{name: "match_depth_beyond_subtree", query: matchC, params: map[string]string{"match_depth": "9"}, expect: []byte{3, 5, 6}},
			{name: "match_depth_unbounded", query: matchC, params: map[string]string{"match_depth": "-1"}, expect: []byte{3, 5, 6}},

			// ancestor_depth is inert unless keep_hierarchy is true, at any value.
			{name: "ancestor_depth_unbounded_without_keep_hierarchy", query: matchC, params: map[string]string{"ancestor_depth": "-1"}, expect: []byte{3}},
			{name: "ancestor_depth_2_without_keep_hierarchy", query: matchC, params: map[string]string{"ancestor_depth": "2"}, expect: []byte{3}},
			{name: "ancestor_depth_unbounded_keep_hierarchy_false", query: matchC, params: map[string]string{"keep_hierarchy": "false", "ancestor_depth": "-1"}, expect: []byte{3}},
			{name: "ancestor_depth_inert_but_match_depth_applies", query: matchC, params: map[string]string{"keep_hierarchy": "false", "ancestor_depth": "-1", "match_depth": "1"}, expect: []byte{3, 5}},
			// an unusable ancestor_depth is not read at all without keep_hierarchy, so it neither fails
			// the request nor changes the result. See assertBadDepthParamsRejected for the 400 it earns
			// once keep_hierarchy=true makes it meaningful.
			{name: "ancestor_depth_out_of_range_ignored_without_keep_hierarchy", query: matchC, params: map[string]string{"ancestor_depth": "-2"}, expect: []byte{3}},

			// keep_hierarchy with ancestor_depth, no descendants.
			{name: "keep_hierarchy_default_ancestor_depth", query: matchC, params: map[string]string{"keep_hierarchy": "true"}, expect: []byte{1, 2, 3}},
			{name: "keep_hierarchy_ancestor_depth_unbounded", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "-1"}, expect: []byte{1, 2, 3}},
			{name: "keep_hierarchy_ancestor_depth_0", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "0"}, expect: []byte{3}},
			{name: "keep_hierarchy_ancestor_depth_1", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "1"}, expect: []byte{2, 3}},
			{name: "keep_hierarchy_ancestor_depth_2", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "2"}, expect: []byte{1, 2, 3}},
			{name: "keep_hierarchy_ancestor_depth_past_root", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "9"}, expect: []byte{1, 2, 3}},

			// all three together: ancestors and descendants are bounded independently.
			{name: "hierarchy_ancestor_0_match_0", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "0", "match_depth": "0"}, expect: []byte{3}},
			{name: "hierarchy_ancestor_0_match_1", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "0", "match_depth": "1"}, expect: []byte{3, 5}},
			{name: "hierarchy_ancestor_0_match_unbounded", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "0", "match_depth": "-1"}, expect: []byte{3, 5, 6}},
			{name: "hierarchy_ancestor_1_match_1", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "1", "match_depth": "1"}, expect: []byte{2, 3, 5}},
			{name: "hierarchy_ancestor_1_match_2", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "1", "match_depth": "2"}, expect: []byte{2, 3, 5, 6}},
			{name: "hierarchy_ancestor_1_match_unbounded", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "1", "match_depth": "-1"}, expect: []byte{2, 3, 5, 6}},
			{name: "hierarchy_ancestor_2_match_1", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "2", "match_depth": "1"}, expect: []byte{1, 2, 3, 5}},
			{name: "hierarchy_ancestor_2_match_2", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "2", "match_depth": "2"}, expect: []byte{1, 2, 3, 5, 6}},
			{name: "hierarchy_ancestor_past_root_match_past_leaf", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "9", "match_depth": "9"}, expect: []byte{1, 2, 3, 5, 6}},
			// unbounded in both directions still excludes D: it is on a sibling branch, neither an
			// ancestor nor a descendant of C.
			{name: "hierarchy_both_unbounded_excludes_sibling_branch", query: matchC, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "-1", "match_depth": "-1"}, expect: []byte{1, 2, 3, 5, 6}},

			// two matched spans: depths are counted per match, so the kept set is the union.
			{name: "two_matches_defaults", query: matchCD, expect: []byte{3, 4}},
			// A is 2 hops above C but only 1 above D, so ancestor_depth=1 still pulls it in.
			{name: "two_matches_ancestor_1", query: matchCD, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "1"}, expect: []byte{1, 2, 3, 4}},
			// D is a leaf, so only C contributes a descendant.
			{name: "two_matches_ancestor_0_match_1", query: matchCD, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "0", "match_depth": "1"}, expect: []byte{3, 4, 5}},
			// the union covers every span, so nothing was dropped and the response is not partial.
			{name: "two_matches_both_unbounded_keeps_whole_trace", query: matchCD, params: map[string]string{"keep_hierarchy": "true", "ancestor_depth": "-1", "match_depth": "-1"}, expect: []byte{1, 2, 3, 4, 5, 6}, wantComplete: true},
		}

		assertFiltering := func(t *testing.T, pathName string, apiClient *httpclient.Client) {
			t.Run(pathName, func(t *testing.T) {
				// no params: full trace, all six spans, and a complete (not partial) status.
				full, err := apiClient.QueryTraceV2WithQueryParams(hexID, nil)
				require.NoError(t, err)
				require.ElementsMatch(t, []byte{1, 2, 3, 4, 5, 6}, spanLastBytes(full.Trace))
				require.Equal(t, tempopb.PartialStatus_COMPLETE, full.Status, "an unfiltered trace is complete")

				for _, tc := range filterCases {
					t.Run(tc.name, func(t *testing.T) {
						params := map[string]string{"q": tc.query}
						for k, v := range tc.params {
							params[k] = v
						}

						resp, err := apiClient.QueryTraceV2WithQueryParams(hexID, params)
						require.NoError(t, err)
						require.ElementsMatch(t, tc.expect, spanLastBytes(resp.Trace))

						if tc.wantComplete {
							require.Equal(t, tempopb.PartialStatus_COMPLETE, resp.Status)
							require.Empty(t, resp.Message)
							return
						}
						// a filter that dropped spans marks the response partial so a client knows it is a subset.
						require.Equal(t, tempopb.PartialStatus_PARTIAL, resp.Status)
						require.Contains(t, resp.Message, "only a subset of spans matching the filter")
					})
				}
			})
		}

		apiClient := h.APIClientHTTP("")

		// recent-data path.
		assertFiltering(t, "recent_data", apiClient)
		assertBadFilterRejected(t, apiClient, hexID)
		assertOversizedFilterRejected(t, apiClient, hexID)
		assertBadDepthParamsRejected(t, apiClient, hexID)

		// backend path.
		h.WaitTracesWrittenToBackend(t, 1)
		h.ForceBackendQuerying(t)
		apiClient = h.APIClientHTTP("")
		assertFiltering(t, "backend", apiClient)
		assertBadFilterRejected(t, apiClient, hexID)
		assertOversizedFilterRejected(t, apiClient, hexID)
		assertBadDepthParamsRejected(t, apiClient, hexID)
	})
}

// assertBadFilterRejected confirms a malformed q is rejected with 400, not 500.
func assertBadFilterRejected(t *testing.T, apiClient *httpclient.Client, hexID string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, apiClient.BaseURL+"/api/v2/traces/"+hexID+"?q="+url.QueryEscape("{ .a = }"), nil)
	require.NoError(t, err)
	resp, err := apiClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// assertOversizedFilterRejected confirms a q over the TraceQL size cap is rejected with 400 before parsing.
// The query is valid TraceQL, so only the size check (not the parser) can reject it.
func assertOversizedFilterRejected(t *testing.T, apiClient *httpclient.Client, hexID string) {
	t.Helper()
	oversized := `{ span.foo = "` + strings.Repeat("x", 150*1024) + `" }`
	req, err := http.NewRequest(http.MethodGet, apiClient.BaseURL+"/api/v2/traces/"+hexID+"?q="+url.QueryEscape(oversized), nil)
	require.NoError(t, err)
	resp, err := apiClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// assertBadDepthParamsRejected confirms an out-of-range depth is rejected with 400, not 500, but only
// where the depth is actually read. match_depth always applies once q is set, while ancestor_depth is
// read only under keep_hierarchy=true — so the same -2 that fails there is accepted and ignored
// without it, and must not fail a request it could not have changed.
func assertBadDepthParamsRejected(t *testing.T, apiClient *httpclient.Client, hexID string) {
	t.Helper()

	rejected := []string{
		"match_depth=-2",
		"match_depth=abc",
		"keep_hierarchy=true&ancestor_depth=-2",
		"keep_hierarchy=true&ancestor_depth=abc",
	}
	ignored := []string{
		"ancestor_depth=-2",
		"ancestor_depth=abc",
		"keep_hierarchy=false&ancestor_depth=-2",
	}

	filterQuery := url.QueryEscape("{ .http.status_code = 500 }")

	for _, param := range rejected {
		req, err := http.NewRequest(http.MethodGet, apiClient.BaseURL+"/api/v2/traces/"+hexID+"?q="+filterQuery+"&"+param, nil)
		require.NoError(t, err)
		resp, err := apiClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected 400 for %s", param)
		require.NoError(t, resp.Body.Close())
	}

	for _, param := range ignored {
		req, err := http.NewRequest(http.MethodGet, apiClient.BaseURL+"/api/v2/traces/"+hexID+"?q="+filterQuery+"&"+param, nil)
		require.NoError(t, err)
		resp, err := apiClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 for unread %s", param)
		require.NoError(t, resp.Body.Close())
	}
}

// spanLastBytes returns the last byte of each span id in the trace, used as a compact span identity.
func spanLastBytes(tr *tempopb.Trace) []byte {
	var ids []byte
	for _, rs := range tr.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				ids = append(ids, s.SpanId[len(s.SpanId)-1])
			}
		}
	}
	return ids
}

// buildFilterTestTrace builds the A->B->C(500)->E->F, A->D(200) trace used by
// TestTraceByIDV2Filtering. E and F give C a two-level descendant subtree so match_depth has
// something meaningful to bound.
func buildFilterTestTrace(traceID []byte) *tempopb.Trace {
	start := uint64(time.Now().Add(-3 * time.Second).UnixNano())
	spanID := func(n byte) []byte { return []byte{0, 0, 0, 0, 0, 0, 0, n} }

	span := func(id, parent byte, statusCode int64) *tracev1.Span {
		s := &tracev1.Span{
			TraceId:           traceID,
			SpanId:            spanID(id),
			Name:              "span",
			StartTimeUnixNano: start,
			EndTimeUnixNano:   start + uint64(time.Millisecond),
		}
		if parent != 0 {
			s.ParentSpanId = spanID(parent)
		}
		if statusCode != 0 {
			s.Attributes = []*commonv1.KeyValue{{
				Key:   "http.status_code",
				Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: statusCode}},
			}}
		}
		return s
	}

	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{{
			Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{{
				Key:   "service.name",
				Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "filter-test"}},
			}}},
			ScopeSpans: []*tracev1.ScopeSpans{{
				Spans: []*tracev1.Span{
					span(1, 0, 0),   // A root
					span(2, 1, 0),   // B child of A
					span(3, 2, 500), // C child of B, matches
					span(4, 1, 200), // D child of A, unrelated
					span(5, 3, 0),   // E child of C
					span(6, 5, 0),   // F child of E
				},
			}},
		}},
	}
}
