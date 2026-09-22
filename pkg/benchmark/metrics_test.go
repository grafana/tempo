package benchmark

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

func TestResponseMetricsCarriesEveryCounter(t *testing.T) {
	m := &tempopb.SearchMetrics{
		InspectedTraces: 3,
		InspectedBytes:  4096,
		InspectedSpans:  120,
		BackendReads:    7,
		BackendBytes:    8192,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricPagesInspected: 12,
			tempopb.AdditionalMetricCacheHits:      5,
		},
	}

	got, err := responseMetrics(m)
	require.NoError(t, err)

	// Named fields arrive under the names the wire format uses, so nothing here
	// has to be updated when Tempo adds one.
	require.Equal(t, int64(3), got["inspectedTraces"])
	require.Equal(t, int64(4096), got["inspectedBytes"])
	require.Equal(t, int64(120), got["inspectedSpans"])
	require.Equal(t, int64(7), got["backendReads"])
	require.Equal(t, int64(8192), got["backendBytes"])

	// AdditionalMetrics is flattened to the top level under its own keys.
	require.Equal(t, int64(12), got[tempopb.AdditionalMetricPagesInspected])
	require.Equal(t, int64(5), got[tempopb.AdditionalMetricCacheHits])

	// A field Tempo did not populate is absent, not zero: the two are
	// different claims, and a reader must be able to tell them apart.
	require.NotContains(t, got, "totalJobs")
}

func TestResponseMetricsWorksForEveryMetricsMessage(t *testing.T) {
	got, err := responseMetrics(&tempopb.TraceByIDMetrics{
		InspectedBytes:    2048,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheMisses: 2},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2048), got["inspectedBytes"])
	require.Equal(t, int64(2), got[tempopb.AdditionalMetricCacheMisses])
}

func TestResponseMetricsEmpty(t *testing.T) {
	got, err := responseMetrics(&tempopb.SearchMetrics{})
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = responseMetrics(nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestAddMetrics(t *testing.T) {
	sum := addMetrics(nil, map[string]int64{"a": 1, "b": 2})
	sum = addMetrics(sum, map[string]int64{"a": 3, "c": 4})
	sum = addMetrics(sum, nil)
	require.Equal(t, map[string]int64{"a": 4, "b": 2, "c": 4}, sum)
}

func TestEvaluatorResponseMetricsIsLossless(t *testing.T) {
	// The metrics path reports a different type, so check it lands under the
	// same keys as the other APIs rather than the evaluator's field names.
	got, err := evaluatorResponseMetrics(traceql.EvaluatorMetrics{
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
	require.Equal(t, int64(12), got[metricSpansDeduped])
}

func TestEvaluatorResponseMetricsEmpty(t *testing.T) {
	got, err := evaluatorResponseMetrics(traceql.EvaluatorMetrics{})
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestGatherMetricsReportsWhatChanged(t *testing.T) {
	reg := prometheus.NewRegistry()

	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bench_reads_total",
	}, []string{"role"})
	hist := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "bench_duration_seconds",
		Buckets: []float64{1, 10},
	})
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: "bench_inflight"})
	idle := prometheus.NewCounter(prometheus.CounterOpts{Name: "bench_idle_total"})
	steady := prometheus.NewGauge(prometheus.GaugeOpts{Name: "bench_steady"})
	reg.MustRegister(counter, hist, gauge, idle, steady)

	counter.WithLabelValues("page").Add(2)
	gauge.Set(1)
	steady.Set(9)
	idle.Add(5)

	before, err := gatherMetrics(reg)
	require.NoError(t, err)

	counter.WithLabelValues("page").Add(3)
	hist.Observe(5)
	gauge.Set(4)

	after, err := gatherMetrics(reg)
	require.NoError(t, err)

	got := after.since(before)

	// A counter reports what this window added, not its cumulative total.
	require.Equal(t, 3.0, got.Deltas[`bench_reads_total{role="page"}`])

	// Histograms give sum, count and buckets, which is the distribution.
	require.Equal(t, 5.0, got.Deltas["bench_duration_seconds_sum"])
	require.Equal(t, 1.0, got.Deltas["bench_duration_seconds_count"])
	require.Equal(t, 1.0, got.Deltas[`bench_duration_seconds_bucket{le="10"}`])
	require.NotContains(t, got.Deltas, `bench_duration_seconds_bucket{le="1"}`)

	// Gauges are snapshots, because a difference of a gauge means nothing.
	require.Equal(t, 4.0, got.Gauges["bench_inflight"])

	// Nothing that stood still is reported, so the result stays readable
	// without the runner keeping a list of metrics it knows about.
	require.NotContains(t, got.Deltas, "bench_idle_total")
	require.NotContains(t, got.Gauges, "bench_steady")
}

func TestGatherMetricsNilGatherer(t *testing.T) {
	snap, err := gatherMetrics(nil)
	require.NoError(t, err)
	require.Empty(t, snap.since(metricSnapshot{}).Deltas)
}

func TestMetricKeyIsStable(t *testing.T) {
	reg := prometheus.NewRegistry()
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bench_labelled_total",
	}, []string{"zone", "role"})
	reg.MustRegister(c)
	c.WithLabelValues("b", "a").Inc()

	first, err := gatherMetrics(reg)
	require.NoError(t, err)
	second, err := gatherMetrics(reg)
	require.NoError(t, err)

	require.Contains(t, first.cumulative, `bench_labelled_total{role="a",zone="b"}`)
	require.Equal(t, first.cumulative, second.cumulative)
}
