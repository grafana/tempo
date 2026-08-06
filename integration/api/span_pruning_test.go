package api

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

// buildSpanPruningTrace returns a trace with a single root span and n identical "leaf-op"
// leaf children (same name, kind and status), optionally tagged with attr so that
// span_pruning_group_by can be exercised to split groups by attribute value. leafIDPrefix
// must be unique per call when combining multiple leaf groups into one trace, so that the
// resulting leaf span IDs don't collide (and get silently deduped) across groups.
func buildSpanPruningTrace(traceID []byte, n int, leafIDPrefix byte, attr *commonv1.KeyValue) *tempopb.Trace {
	root := test.MakeSpanPruningSpan(traceID, test.MakeSpanPruningSpanID(1, 0), nil, "root", 0, 1_000_000)
	spans := []*tracev1.Span{root}
	for i := 0; i < n; i++ {
		var attrs []*commonv1.KeyValue
		if attr != nil {
			attrs = []*commonv1.KeyValue{attr}
		}
		spans = append(spans, test.MakeSpanPruningSpan(traceID, test.MakeSpanPruningSpanID(leafIDPrefix, byte(i)), root.SpanId, "leaf-op",
			uint64(i*1_000), uint64(i*1_000+100), attrs...))
	}
	return test.WrapSpansAsTrace(spans...)
}

// TestSpanPruning verifies that query_frontend.trace_by_id.span_pruning_enabled gates the
// span_pruning* request params on /api/v2/traces/{id}, and that those params control whether
// and how duplicate leaf spans get collapsed into aggregated summary spans.
func TestSpanPruning(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: "config-span-pruning.yaml",
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// 6 identical leaves >= the default min_spans_to_aggregate (5): aggregates as a whole.
		basicTraceID := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00}
		require.NoError(t, h.WriteTempoProtoTraces(buildSpanPruningTrace(basicTraceID, 6, 2, nil), ""))

		// 3 mysql + 3 postgres leaves (distinct leaf-ID prefixes so they don't collide): below
		// the default threshold per-group, but splits into 2 summaries when grouped by
		// db.system with a lowered min_spans.
		groupedTraceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
		grouped := buildSpanPruningTrace(groupedTraceID, 3, 2, test.MakeAttribute("db.system", "mysql"))
		postgres := buildSpanPruningTrace(groupedTraceID, 3, 3, test.MakeAttribute("db.system", "postgres"))
		grouped.ResourceSpans[0].ScopeSpans[0].Spans = append(grouped.ResourceSpans[0].ScopeSpans[0].Spans, postgres.ResourceSpans[0].ScopeSpans[0].Spans[1:]...)
		require.NoError(t, h.WriteTempoProtoTraces(grouped, ""))

		h.WaitTracesQueryable(t, 2)

		client := h.APIClientHTTP("")
		basicHexID := hex.EncodeToString(basicTraceID)
		groupedHexID := hex.EncodeToString(groupedTraceID)

		t.Run("without span_pruning param the trace is returned untouched", func(t *testing.T) {
			resp, err := client.QueryTraceV2(basicHexID)
			require.NoError(t, err)
			require.Len(t, test.AllSpansInTrace(resp.Trace), 7) // root + 6 leaves
			require.Empty(t, test.SpanPruningSummaries(resp.Trace))
		})

		t.Run("span_pruning=true aggregates the leaves into a summary span", func(t *testing.T) {
			resp, err := client.QueryTraceV2WithQueryParams(basicHexID, map[string]string{"span_pruning": "true"})
			require.NoError(t, err)

			require.Len(t, test.AllSpansInTrace(resp.Trace), 2) // root + 1 summary

			summaries := test.SpanPruningSummaries(resp.Trace)
			require.Len(t, summaries, 1)
			require.Equal(t, int64(6), test.SpanAttrInt(summaries[0], "aggregation.span_count"))
			require.Equal(t, "leaf-op", summaries[0].Name)
		})

		t.Run("span_pruning_min_spans raises the threshold above the group size", func(t *testing.T) {
			resp, err := client.QueryTraceV2WithQueryParams(basicHexID, map[string]string{
				"span_pruning":           "true",
				"span_pruning_min_spans": "10",
			})
			require.NoError(t, err)
			require.Len(t, test.AllSpansInTrace(resp.Trace), 7) // 6 < 10, nothing aggregated
			require.Empty(t, test.SpanPruningSummaries(resp.Trace))
		})

		t.Run("span_pruning_group_by splits aggregation per attribute value", func(t *testing.T) {
			resp, err := client.QueryTraceV2WithQueryParams(groupedHexID, map[string]string{
				"span_pruning":           "true",
				"span_pruning_group_by":  "db.system",
				"span_pruning_min_spans": "3",
			})
			require.NoError(t, err)

			require.Len(t, test.AllSpansInTrace(resp.Trace), 3) // root + mysql summary + postgres summary

			summaries := test.SpanPruningSummaries(resp.Trace)
			require.Len(t, summaries, 2)
			byDB := map[string]int64{}
			for _, s := range summaries {
				byDB[test.SpanAttrString(s, "db.system")] = test.SpanAttrInt(s, "aggregation.span_count")
			}
			require.Equal(t, int64(3), byDB["mysql"])
			require.Equal(t, int64(3), byDB["postgres"])
		})

		t.Run("invalid span_pruning_min_spans is rejected with a 400", func(t *testing.T) {
			_, err := client.QueryTraceV2WithQueryParams(basicHexID, map[string]string{
				"span_pruning":           "true",
				"span_pruning_min_spans": "not-a-number",
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "span_pruning_min_spans")
		})

		t.Run("invalid span_pruning value is rejected with a 400", func(t *testing.T) {
			_, err := client.QueryTraceV2WithQueryParams(basicHexID, map[string]string{"span_pruning": "maybe"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "span_pruning")
		})
	})
}

// TestSpanPruningEnabledByDefault verifies that
// query_frontend.trace_by_id.span_pruning_enabled_by_default makes pruning the default for
// requests that don't set their own span_pruning param, while an explicit span_pruning value in
// the request (true or false) still takes precedence.
func TestSpanPruningEnabledByDefault(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: "config-span-pruning-enabled-by-default.yaml",
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		basicTraceID := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00}
		require.NoError(t, h.WriteTempoProtoTraces(buildSpanPruningTrace(basicTraceID, 6, 2, nil), ""))
		h.WaitTracesQueryable(t, 1)

		client := h.APIClientHTTP("")
		basicHexID := hex.EncodeToString(basicTraceID)

		t.Run("without span_pruning param the trace is pruned by default", func(t *testing.T) {
			resp, err := client.QueryTraceV2(basicHexID)
			require.NoError(t, err)
			require.Len(t, test.AllSpansInTrace(resp.Trace), 2) // root + 1 summary

			summaries := test.SpanPruningSummaries(resp.Trace)
			require.Len(t, summaries, 1)
			require.Equal(t, int64(6), test.SpanAttrInt(summaries[0], "aggregation.span_count"))
		})

		t.Run("explicit span_pruning=false is respected", func(t *testing.T) {
			resp, err := client.QueryTraceV2WithQueryParams(basicHexID, map[string]string{"span_pruning": "false"})
			require.NoError(t, err)
			require.Len(t, test.AllSpansInTrace(resp.Trace), 7) // param respected, trace unpruned
			require.Empty(t, test.SpanPruningSummaries(resp.Trace))
		})

		t.Run("request-supplied span_pruning_min_spans is honored without span_pruning=true", func(t *testing.T) {
			resp, err := client.QueryTraceV2WithQueryParams(basicHexID, map[string]string{
				"span_pruning_min_spans": "10",
			})
			require.NoError(t, err)
			require.Len(t, test.AllSpansInTrace(resp.Trace), 7) // 6 < 10, nothing aggregated
			require.Empty(t, test.SpanPruningSummaries(resp.Trace))
		})
	})
}

// TestSpanPruningDisabledByConfig verifies that when span_pruning_enabled is left at its
// default (false), span_pruning* request params are silently ignored and the trace is
// returned unpruned - the config flag is the master switch, not just a request-side hint.
func TestSpanPruningDisabledByConfig(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		traceID := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00}
		require.NoError(t, h.WriteTempoProtoTraces(buildSpanPruningTrace(traceID, 6, 2, nil), ""))
		h.WaitTracesQueryable(t, 1)

		client := h.APIClientHTTP("")
		hexID := hex.EncodeToString(traceID)

		resp, err := client.QueryTraceV2WithQueryParams(hexID, map[string]string{"span_pruning": "true"})
		require.NoError(t, err)

		require.Len(t, test.AllSpansInTrace(resp.Trace), 7) // param ignored, trace unpruned
		require.Empty(t, test.SpanPruningSummaries(resp.Trace))
	})
}
