// Package comparetest builds benchmark results and comparisons for the tests of
// the packages that render them.
package comparetest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// Result is a run on host with the given cases, and otherwise the defaults.
func Result(host string, cases ...benchmark.CaseResult) *benchmark.Result {
	return &benchmark.Result{
		SchemaVersion: benchmark.ResultSchemaVersion,
		RunEnv:        benchmark.RunEnv{GitSHA: "abc", GoVersion: "go1.27", GoMaxProcs: 12, Hostname: host},
		Options:       benchmark.RunOptions{Repeat: 1},
		Shards:        55,
		Cases:         cases,
	}
}

// Case is a search case of 55 executions.
func Case(id string, matched int64, set metrics.Set) benchmark.CaseResult {
	return benchmark.CaseResult{ID: id, API: "search", Executions: 55, Matched: matched, Metrics: set}
}

// Measurement is a spread around p50, wide enough to draw.
func Measurement(p50 float64) metrics.Measurement {
	return metrics.Measurement{Kind: metrics.Counter, Summary: Summary(p50/2, p50*0.8, p50, p50*1.2, p50*1.4, p50*1.6, p50*2)}
}

// Summary is a summary of ten executions with the given quantiles.
func Summary(minV, p25, p50, p75, p90, p99, maxV float64) metrics.Summary {
	return metrics.Summary{Count: 10, Min: minV, P25: p25, P50: p50, P75: p75, P90: p90, P99: p99, Max: maxV}
}

// Comparison has four runs: the baseline, another run set up like it, and
// two read buffer sizes, the last of which matched differently on a search.
func Comparison(t *testing.T) *compare.Comparison {
	t.Helper()
	run := func(name string, size int, p50 float64, matched int64) compare.Run {
		r := Result("mac",
			Case("traceid/present", 1, metrics.Set{
				"harness.wallNs": Measurement(p50),
				"backend.reads":  Measurement(200),
			}),
			Case("search/nopredicate", matched, metrics.Set{"harness.wallNs": Measurement(10)}),
		)
		r.Cases[0].API = "traceByID"
		r.Options.ReadBufferSize = size
		return compare.Run{Name: name, Result: r}
	}
	c, err := compare.New([]compare.Run{
		run("base", 0, 100, 5),
		run("again", 0, 104, 5),
		run("2MiB", 2<<20, 98, 5),
		run("4MiB", 4<<20, 87, 6),
	})
	require.NoError(t, err)
	return c
}

// Family compares base and next over search/nopredicate and nine cases of the
// family search/ranked_generic, three per band, all 10ns on base. Next is 4x on
// case 0, 1/4 on case 1, +10% on six more, and matches differently on case 8.
func Family(t *testing.T) *compare.Comparison {
	t.Helper()
	run := func(name string, next bool) compare.Run {
		r := Result("mac", Case("search/nopredicate", 1, metrics.Set{"harness.wallNs": Measurement(10)}))
		for i := range 9 {
			p50, matched := 10.0, int64(1)
			if next {
				switch i {
				case 0:
					p50 = 40
				case 1:
					p50 = 2.5
				case 8:
					matched = 2
				default:
					p50 = 11
				}
			}
			r.Cases = append(r.Cases, Case(fmt.Sprintf("search/ranked_generic/%s/%d", []string{"high", "medium", "low"}[i%3], i), matched, metrics.Set{"harness.wallNs": Measurement(p50)}))
		}
		return compare.Run{Name: name, Result: r}
	}
	c, err := compare.New([]compare.Run{run("base", false), run("next", true)})
	require.NoError(t, err)
	return c
}
