package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

func TestFromResponseCarriesEveryCounter(t *testing.T) {
	got, err := FromResponse(&tempopb.SearchMetrics{
		InspectedTraces: 3,
		InspectedBytes:  4096,
		InspectedSpans:  120,
		BackendReads:    7,
		BackendBytes:    8192,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricPagesInspected: 12,
			tempopb.AdditionalMetricCacheHits:      5,
		},
	})
	require.NoError(t, err)

	// Named fields arrive under the names the wire format uses, so nothing here
	// needs updating when Tempo adds one.
	require.Equal(t, int64(3), got["inspectedTraces"])
	require.Equal(t, int64(4096), got["inspectedBytes"])
	require.Equal(t, int64(120), got["inspectedSpans"])
	require.Equal(t, int64(7), got["backendReads"])
	require.Equal(t, int64(8192), got["backendBytes"])

	// AdditionalMetrics is flattened to the top level under its own keys.
	require.Equal(t, int64(12), got[tempopb.AdditionalMetricPagesInspected])
	require.Equal(t, int64(5), got[tempopb.AdditionalMetricCacheHits])

	// A field Tempo did not populate is absent, not zero.
	require.NotContains(t, got, "totalJobs")
}

func TestFromResponseWorksForEveryMetricsMessage(t *testing.T) {
	got, err := FromResponse(&tempopb.TraceByIDMetrics{
		InspectedBytes:    2048,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheMisses: 2},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2048), got["inspectedBytes"])
	require.Equal(t, int64(2), got[tempopb.AdditionalMetricCacheMisses])
}

func TestFromResponseEmpty(t *testing.T) {
	got, err := FromResponse(&tempopb.SearchMetrics{})
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = FromResponse(nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestFromEvaluatorIsLossless(t *testing.T) {
	// The metrics path reports a different type, so check it lands under the
	// same keys as the other APIs.
	got, err := FromEvaluator(traceql.EvaluatorMetrics{
		Bytes:             4096,
		SpansTotal:        800,
		SpansDeduped:      12,
		BackendReads:      3,
		BackendBytes:      2048,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricEngineBytes: 64},
	})
	require.NoError(t, err)

	require.Equal(t, int64(4096), got["inspectedBytes"])
	require.Equal(t, int64(800), got["inspectedSpans"])
	require.Equal(t, int64(3), got["backendReads"])
	require.Equal(t, int64(2048), got["backendBytes"])
	require.Equal(t, int64(64), got[tempopb.AdditionalMetricEngineBytes])
	// SpansDeduped has no SearchMetrics field, so it must not be dropped.
	require.Equal(t, int64(12), got[spansDeduped])
}

func TestFromEvaluatorEmpty(t *testing.T) {
	got, err := FromEvaluator(traceql.EvaluatorMetrics{})
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestBytesRead(t *testing.T) {
	// Keyed like the responses, so a reader does not have to know which API
	// produced the row.
	got, err := BytesRead(512)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"inspectedBytes": 512}, got)

	got, err = BytesRead(0)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestAdd(t *testing.T) {
	sum := Add(nil, map[string]int64{"a": 1, "b": 2})
	sum = Add(sum, map[string]int64{"a": 3, "c": 4})
	sum = Add(sum, nil)
	require.Equal(t, map[string]int64{"a": 4, "b": 2, "c": 4}, sum)
}
