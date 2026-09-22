package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestGatherReportsWhatChanged(t *testing.T) {
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

	before, err := Gather(reg)
	require.NoError(t, err)

	counter.WithLabelValues("page").Add(3)
	hist.Observe(5)
	gauge.Set(4)

	after, err := Gather(reg)
	require.NoError(t, err)

	got := after.Since(before)

	// A counter reports what this window added, not its total.
	require.Equal(t, 3.0, got.Deltas[`bench_reads_total{role="page"}`])

	// Histograms give sum, count and buckets, which is the distribution.
	require.Equal(t, 5.0, got.Deltas["bench_duration_seconds_sum"])
	require.Equal(t, 1.0, got.Deltas["bench_duration_seconds_count"])
	require.Equal(t, 1.0, got.Deltas[`bench_duration_seconds_bucket{le="10"}`])
	require.NotContains(t, got.Deltas, `bench_duration_seconds_bucket{le="1"}`)

	// Gauges are values, not deltas.
	require.Equal(t, 4.0, got.Gauges["bench_inflight"])

	// Nothing that stood still is reported, so the result stays readable
	// without keeping a list of metrics we know about.
	require.NotContains(t, got.Deltas, "bench_idle_total")
	require.NotContains(t, got.Gauges, "bench_steady")
}

func TestGatherNilGatherer(t *testing.T) {
	snap, err := Gather(nil)
	require.NoError(t, err)
	require.Empty(t, snap.Since(Snapshot{}).Deltas)
}

func TestMetricKeyIsStable(t *testing.T) {
	reg := prometheus.NewRegistry()
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bench_labelled_total",
	}, []string{"zone", "role"})
	reg.MustRegister(c)
	c.WithLabelValues("b", "a").Inc()

	first, err := Gather(reg)
	require.NoError(t, err)
	second, err := Gather(reg)
	require.NoError(t, err)

	require.Contains(t, first.cumulative, `bench_labelled_total{role="a",zone="b"}`)
	require.Equal(t, first.cumulative, second.cumulative)
}
