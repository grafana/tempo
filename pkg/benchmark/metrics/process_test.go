package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestCollectorReportsWhatChanged(t *testing.T) {
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

	c := NewCollector(nil, reg)
	c.BeginCase()

	// Two executions, each moving the metrics by a different amount, so the
	// distribution has something to describe.
	c.BeginExecution()
	counter.WithLabelValues("page").Add(1)
	hist.Observe(5)
	gauge.Set(4)
	c.EndExecution(100, nil)

	c.BeginExecution()
	counter.WithLabelValues("page").Add(3)
	hist.Observe(5)
	gauge.Set(6)
	c.EndExecution(300, nil)

	got := c.EndCase()

	// A counter's total is what the case added; its summary describes the
	// per-execution deltas.
	reads := got[PrefixProcess+`bench_reads_total{role="page"}`]
	require.Equal(t, Counter, reads.Kind)
	require.Equal(t, 4.0, reads.Total)
	require.Equal(t, 2, reads.Summary.Count)
	require.Equal(t, 1.0, reads.Summary.Min)
	require.Equal(t, 3.0, reads.Summary.Max)

	// Histograms give sum, count and buckets, so the distribution comes along.
	require.Equal(t, 10.0, got[PrefixProcess+"bench_duration_seconds_sum"].Total)
	require.Equal(t, 2.0, got[PrefixProcess+"bench_duration_seconds_count"].Total)
	require.Equal(t, 2.0, got[PrefixProcess+`bench_duration_seconds_bucket{le="10"}`].Total)
	require.NotContains(t, got, PrefixProcess+`bench_duration_seconds_bucket{le="1"}`)

	// A gauge's total is the value it was left at, marked so it is not summed.
	inflight := got[PrefixProcess+"bench_inflight"]
	require.Equal(t, Gauge, inflight.Kind)
	require.Equal(t, 6.0, inflight.Total)
	require.Equal(t, 4.0, inflight.Summary.Min)

	// The harness measures its own wall clock the same way.
	wall := got[KeyWallNs]
	require.Equal(t, 400.0, wall.Total)
	require.Equal(t, 100.0, wall.Summary.Min)
	require.Equal(t, 300.0, wall.Summary.Max)

	// Nothing that stood still is reported, so the result stays readable
	// without keeping a list of metrics we know about.
	require.NotContains(t, got, PrefixProcess+"bench_idle_total")
	require.NotContains(t, got, PrefixProcess+"bench_steady")
}

func TestCollectorKeepsMeasuredZeros(t *testing.T) {
	// A reader that is never used: its counters were measured, and they are
	// zero. Only the registry's untouched counters are dropped.
	c := NewCollector(&CountingReader{}, nil)
	c.BeginCase()
	c.BeginExecution()
	c.EndExecution(0, nil)

	got := c.EndCase()
	for _, key := range []string{KeyBackendReads, KeyBackendBytes, KeyBackendTimeNs, KeyWallNs} {
		require.Contains(t, got, key, "a measured zero must not read as absent")
		require.Zero(t, got[key].Total, key)
		require.Equal(t, 1, got[key].Summary.Count, key)
	}
}

func TestCollectorNilGatherer(t *testing.T) {
	c := NewCollector(nil, nil)
	c.BeginCase()
	c.BeginExecution()
	c.EndExecution(50, map[string]int64{"inspectedBytes": 7})

	got := c.EndCase()
	require.Equal(t, 50.0, got[KeyWallNs].Total)
	require.Equal(t, 7.0, got[PrefixResponse+"inspectedBytes"].Total)
	for key := range got {
		require.NotContains(t, key, PrefixProcess)
	}
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
