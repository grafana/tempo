package compare

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestParseRunArg(t *testing.T) {
	tests := []struct {
		arg, name, file string
	}{
		{"base=out/a.json", "base", "out/a.json"},
		// A bare path is left for New to name.
		{"out/base.json", "", "out/base.json"},
		{"base.json", "", "base.json"},
		{"a=b=c.json", "a", "b=c.json"},
		// A separator before the = means it is part of a path, not a name.
		{"out/x=1.json", "", "out/x=1.json"},
		{"=a.json", "", "=a.json"},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			name, file := parseRunArg(tt.arg)
			require.Equal(t, tt.name, name)
			require.Equal(t, tt.file, file)
		})
	}
}

func TestLoadRun(t *testing.T) {
	file := filepath.Join(t.TempDir(), "base.json")
	f, err := os.Create(file)
	require.NoError(t, err)
	require.NoError(t, newResult("host", caseResult("traceid/present", 10, nil)).Write(f))
	require.NoError(t, f.Close())

	run, err := LoadRun(file)
	require.NoError(t, err)
	require.Empty(t, run.Name)
	require.Equal(t, file, run.File)
	require.Len(t, run.Result.Cases, 1)

	run, err = LoadRun("named=" + file)
	require.NoError(t, err)
	require.Equal(t, "named", run.Name)
	require.Equal(t, file, run.File)

	_, err = LoadRun(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
}

func TestNew(t *testing.T) {
	_, err := New([]Run{{Name: "a", Result: newResult("h")}})
	require.Error(t, err, "one run is not a comparison")

	_, err = New([]Run{{Name: "a", Result: newResult("h")}, {Name: "a", Result: newResult("h")}})
	require.Error(t, err, "names must be unique")

	a := newResult("h",
		caseResult("traceid/present", 10, nil),
		caseResult("search/nopredicate", 5, nil),
	)
	b := newResult("h",
		caseResult("search/nopredicate", 5, nil),
		caseResult("metrics/rate", 1, nil),
	)
	c, err := New([]Run{{Name: "a", Result: a}, {Name: "b", Result: b}})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, c.Names())
	require.Equal(t, 0, c.Baseline)
	require.Empty(t, c.Varying)
	require.Equal(t, []Detail{{Text: "same setup as the baseline", Kind: Derived}}, c.Describe(1, 0))

	// Cases come in the first run's order, then any only later runs have.
	require.Len(t, c.Cases, 3)
	require.Equal(t, "traceid/present", c.Cases[0].ID)
	require.Equal(t, "search/nopredicate", c.Cases[1].ID)
	require.Equal(t, "metrics/rate", c.Cases[2].ID)

	require.NotNil(t, c.Cases[0].Results[0])
	require.Nil(t, c.Cases[0].Results[1])
	require.Equal(t, []string{"vs b: missing"}, c.Problems(c.Cases[0], 0))
	require.Empty(t, c.Problems(c.Cases[1], 0))
	require.Equal(t, []string{"vs b: missing from the baseline"}, c.Problems(c.Cases[2], 0))
}

func TestProblems(t *testing.T) {
	a := newResult("host-a", caseResult("search/nopredicate", 5, nil))
	b := newResult("host-b", caseResult("search/nopredicate", 6, nil))
	b.Cases[0].Executions = 110
	b.Options.ReadBufferSize = 4 << 20
	b.RunEnv.GitSHA = "def"

	failed := caseResult("metrics/rate", 0, nil)
	failed.Error = "boom"
	b.Cases = append(b.Cases, failed)

	c, err := New([]Run{{Name: "a", Result: a}, {Name: "b", Result: b}})
	require.NoError(t, err)

	require.Equal(t, []string{"readBufferSize", "gitSHA", "hostname"}, c.Varying)
	require.Equal(t, []string{"vs b: matched 5 vs 6"}, c.Problems(c.Cases[0], 0), "the match count says it first")
	require.Equal(t, []string{"vs b: missing from the baseline"}, c.Problems(c.Cases[1], 0))
	require.Equal(t, []string{"vs a: missing"}, c.Problems(c.Cases[1], 1), "against another baseline")
}

func TestIncomparable(t *testing.T) {
	cr := func(matched int64, executions int, err string) *benchmark.CaseResult {
		return &benchmark.CaseResult{Matched: matched, Executions: executions, Error: err}
	}
	tests := []struct {
		name      string
		base, run *benchmark.CaseResult
		want      string
	}{
		{"alike", cr(5, 55, ""), cr(5, 55, ""), ""},
		{"missing", cr(5, 55, ""), nil, "missing"},
		{"missing from the baseline", nil, cr(5, 55, ""), "missing from the baseline"},
		{"failed", cr(5, 55, ""), cr(5, 55, "boom"), "failed: boom"},
		{"the baseline failed", cr(5, 55, "boom"), cr(5, 55, ""), "the baseline failed: boom"},
		{"matched", cr(5, 55, ""), cr(6, 55, ""), "matched 5 vs 6"},
		{"executions", cr(5, 55, ""), cr(5, 110, ""), "executions 55 vs 110"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Case{Results: []*benchmark.CaseResult{tt.base, tt.run}}.Incomparable(1, 0))
		})
	}
}

func TestFilterCases(t *testing.T) {
	newComparison := func() *Comparison {
		r := newResult("h",
			caseResult("traceid/present", 1, nil),
			caseResult("traceid/absent", 1, nil),
			caseResult("search/nopredicate", 1, nil),
		)
		c, err := New([]Run{{Name: "a", Result: r}, {Name: "b", Result: r}})
		require.NoError(t, err)
		return c
	}

	c := newComparison()
	require.NoError(t, c.FilterCases(nil))
	require.Len(t, c.Cases, 3)

	require.NoError(t, c.FilterCases([]string{"traceid/*"}))
	require.Len(t, c.Cases, 2)

	c = newComparison()
	require.NoError(t, c.FilterCases([]string{"search/*", "traceid/absent"}))
	require.Equal(t, "traceid/absent", c.Cases[0].ID)
	require.Equal(t, "search/nopredicate", c.Cases[1].ID)

	c = newComparison()
	require.Error(t, c.FilterCases([]string{"nothing/*"}))
	require.Len(t, c.Cases, 3, "a filter that matches nothing leaves the cases alone")
	require.Error(t, c.FilterCases([]string{"["}))
}

func TestMetrics(t *testing.T) {
	r := newResult("h", caseResult("traceid/present", 1, metrics.Set{
		"harness.wallNs":     measurement(1),
		"harness.cpuNs":      measurement(1),
		"backend.bytes":      measurement(1),
		"process.go_threads": measurement(1),
	}))
	c, err := New([]Run{{Name: "a", Result: r}, {Name: "b", Result: r}})
	require.NoError(t, err)

	got, err := c.Metrics([]string{"harness.wallNs", "backend.*", "harness.*", "missing.*"})
	require.NoError(t, err)
	require.Equal(t, []string{"harness.wallNs", "backend.bytes", "harness.cpuNs"}, got)

	_, err = c.Metrics([]string{"missing.*"})
	require.Error(t, err)
	_, err = c.Metrics([]string{"["})
	require.Error(t, err)
}

func TestCaseSeries(t *testing.T) {
	a := newResult("h", caseResult("traceid/present", 1, metrics.Set{"harness.wallNs": measurement(10)}))
	b := newResult("h", caseResult("traceid/present", 1, metrics.Set{"harness.wallNs": {Kind: metrics.Counter}}))
	c, err := New([]Run{{Name: "a", Result: a}, {Name: "b", Result: b}, {Name: "c", Result: newResult("h")}})
	require.NoError(t, err)

	s := c.Cases[0].Series("harness.wallNs")
	require.Equal(t, Nanoseconds, s.Unit)
	require.Len(t, s.Summaries, 3)
	require.Equal(t, 10.0, s.Summaries[0].P50)
	require.Nil(t, s.Summaries[1], "a summary of no executions is no data")
	require.Nil(t, s.Summaries[2], "a run without the case has no data")
}

func TestDelta(t *testing.T) {
	pct, ok := delta(100, 87)
	require.True(t, ok)
	require.InDelta(t, -13.0, pct, 1e-9)

	pct, ok = delta(50, 75)
	require.True(t, ok)
	require.InDelta(t, 50.0, pct, 1e-9)

	_, ok = delta(0, 5)
	require.False(t, ok)
}

func newResult(host string, cases ...benchmark.CaseResult) *benchmark.Result {
	return &benchmark.Result{
		SchemaVersion: benchmark.ResultSchemaVersion,
		RunEnv:        benchmark.RunEnv{GitSHA: "abc", GoVersion: "go1.27", GoMaxProcs: 12, Hostname: host},
		Options:       benchmark.RunOptions{Repeat: 1, Warmup: 1},
		Shards:        55,
		Cases:         cases,
	}
}

func caseResult(id string, matched int64, set metrics.Set) benchmark.CaseResult {
	return benchmark.CaseResult{ID: id, API: "search", Executions: 55, Matched: matched, Metrics: set}
}

// measurement is a spread around p50, wide enough to draw.
func measurement(p50 float64) metrics.Measurement {
	return metrics.Measurement{Kind: metrics.Counter, Summary: summary(p50/2, p50*0.8, p50, p50*1.2, p50*1.4, p50*1.6, p50*2)}
}

func summary(minV, p25, p50, p75, p90, p99, maxV float64) metrics.Summary {
	return metrics.Summary{Count: 10, Min: minV, P25: p25, P50: p50, P75: p75, P90: p90, P99: p99, Max: maxV}
}
