package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestWrite(t *testing.T) {
	c := comparetest.Comparison(t)
	c.Runs[3].Name = "4|MiB" // a name that would break a table

	var b strings.Builder
	require.NoError(t, Write(&b, c, []string{"harness.wallNs"}, compare.P50, 0))
	md := b.String()

	require.Contains(t, md, "### Benchmark comparison\n\nRuns in order of readBufferSize; baseline base.\n\n")
	// The runs' table is padded too, both its columns aligned left.
	require.Contains(t, md, "| run        | setup ")
	require.Contains(t, md, "| :--------- | :----")
	require.Contains(t, md, "| **4\\|MiB** | readBufferSize default → 4MiB ")
	require.Contains(t, md, "#### harness.wallNs · p50 per execution\n\nChange from base.\n\n")

	// Columns are padded to line up, the case aligned left and numbers right,
	// so the table reads in a terminal as it does rendered.
	require.Contains(t, md, ""+
		"| case                 |  base | again |       | 2MiB |       |         4\\|MiB |            |\n"+
		"| :------------------- | ----: | ----: | ----: | ---: | ----: | -------------: | ---------: |\n"+
		"| traceid/present      | 100ns | 104ns | +4.0% | 98ns | -2.0% |           87ns | **-13.0%** |\n"+
		"| search/nopredicate ⚠ |  10ns |  10ns |    0% | 10ns |    0% | not comparable |            |\n")
	require.Contains(t, md, "\n- ⚠ search/nopredicate vs 4\\|MiB: matched 5 vs 6\n")
	require.NotContains(t, md, "traceByID", "headings are for the terminal; a markdown table stays flat")

	// Numbered runs are listed with their numbers.
	c.Runs[3].Name = "a-long-name-for-4MiB"
	b.Reset()
	require.NoError(t, Write(&b, c, []string{"harness.wallNs"}, compare.P50, 0))
	require.Contains(t, b.String(), "| #4 **a-long-name-for-4MiB** | readBufferSize default → 4MiB ")
	require.Contains(t, b.String(), "| case                 |    #1 |    #2 |       |   #3 |       |             #4 |            |\n")
	require.Contains(t, b.String(), "Change from #1.\n")

	// Every run fanned out alike, so no case totals are shown.
	require.NotContains(t, b.String(), "case total per pass")
}

func TestWriteTotalsTable(t *testing.T) {
	withTotal := func(m metrics.Measurement, total float64) metrics.Measurement {
		m.Total = total
		return m
	}
	run := func(name string, searchExecutions int, searchTotal float64) compare.Run {
		r := comparetest.Result("mac",
			comparetest.Case("traceid/present", 200, metrics.Set{
				"harness.wallNs": withTotal(comparetest.Measurement(50), 5000),
			}),
			comparetest.Case("search/nopredicate", 5, metrics.Set{
				"harness.wallNs": withTotal(comparetest.Measurement(10), searchTotal),
			}),
		)
		r.Cases[0].API = "traceByID"
		// The search case's executions measure different amounts of work, as
		// when the runs' blocks shard differently, but the match counts agree.
		r.Cases[1].Executions = searchExecutions
		return compare.Run{Name: name, Result: r}
	}
	c, err := compare.New([]compare.Run{
		run("base", 55, 5500),
		run("200mb", 28, 5600),
	})
	require.NoError(t, err)

	var b strings.Builder
	require.NoError(t, Write(&b, c, []string{"harness.wallNs"}, compare.P50, 0))
	md := b.String()

	// The output teaches why some per-execution comparisons are withheld.
	require.Contains(t, md, "Shard counts differ between runs")

	// The per-execution table keeps the case whose shard counts agree, and
	// drops the one they do not.
	perExec := strings.Index(md, "#### harness.wallNs · p50 per execution")
	totals := strings.Index(md, "#### harness.wallNs · case total per pass")
	require.NotEqual(t, -1, perExec)
	require.NotEqual(t, -1, totals)
	require.Contains(t, md[perExec:totals], "traceid/present")
	require.NotContains(t, md[perExec:totals], "search/nopredicate")

	// The totals table compares every case.
	require.Contains(t, md[totals:], "traceid/present")
	require.Contains(t, md[totals:], "search/nopredicate")
	// 5500ns and 5600ns total over the case, a +1.8% change.
	require.Contains(t, md[totals:], "5.5µs")
	require.Contains(t, md[totals:], "5.6µs")
	require.Contains(t, md[totals:], "+1.8%")

	require.NotContains(t, md, "not comparable")
}
