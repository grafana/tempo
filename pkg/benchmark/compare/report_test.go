package compare

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestWriteReport(t *testing.T) {
	result := func(size int, p50 float64) *benchmark.Result {
		r := newResult("mac", caseResult("search/nopredicate", 5, metrics.Set{"harness.wallNs": measurement(p50)}))
		r.Cases[0].Query = "{}"
		r.Options.ReadBufferSize = size
		return r
	}
	c, err := New([]Run{
		{File: "base.json", Result: result(0, 10e6)},
		{File: "big.json", Result: result(4<<20, 8.7e6)},
	})
	require.NoError(t, err)

	var b strings.Builder
	require.NoError(t, WriteReport(&b, c, []string{"harness.wallNs"}, P50, c.Baseline, 80))
	report := b.String()

	require.Contains(t, report, "runs named by readBufferSize, in its order; baseline default\n")
	require.Contains(t, report, "  default  baseline · readBufferSize default · gitSHA abc")
	require.Contains(t, report, "  4MiB     readBufferSize default → 4MiB\n")
	require.Contains(t, report, "harness.wallNs · p50 per execution · change from default\n"+
		"  case               │ default │     4MiB\n"+
		" search\n"+
		"  search/nopredicate │    10ms │ 8.7ms  -13.0%\n")
	require.Contains(t, report, "search/nopredicate · harness.wallNs · per execution\nquery: {}\n")
	require.Contains(t, report, "4MiB     10  8.7ms")
	require.Contains(t, report, "-13.0%")
}

func TestRunLeads(t *testing.T) {
	r := newResult("mac")
	c, err := New([]Run{{Name: "base", Result: r}, {Name: "4MiB", Result: r}})
	require.NoError(t, err)
	require.Equal(t, []string{"base  ", "4MiB  "}, c.RunLeads())

	// Runs numbered to head columns are listed with their numbers, as the key.
	c.Runs[1].Name = "a-longer-name"
	require.Equal(t, []string{"#1 base           ", "#2 a-longer-name  "}, c.RunLeads())
}

func TestRunsHeading(t *testing.T) {
	c := &Comparison{Runs: []Run{{Name: "a"}, {Name: "b"}}}
	require.Equal(t, "runs; baseline a", c.RunsHeading(0))

	c.NamedBy = []string{"readBufferSize", "readBufferCount"}
	require.Equal(t, "runs named by readBufferSize, readBufferCount; baseline b", c.RunsHeading(1))

	c.OrderedBy = "readBufferSize"
	require.Equal(t, "runs named by readBufferSize, readBufferCount in order of readBufferSize; baseline a", c.RunsHeading(0))

	c.NamedBy = []string{"readBufferSize"}
	require.Equal(t, "runs named by readBufferSize, in its order; baseline a", c.RunsHeading(0))
}

func TestWriteMarkdown(t *testing.T) {
	c := newSummaryComparison(t)
	c.Runs[3].Name = "4|MiB" // a name that would break a table

	var b strings.Builder
	require.NoError(t, WriteMarkdown(&b, c, []string{"harness.wallNs"}, P50, 0))
	md := b.String()

	require.Contains(t, md, "### Benchmark comparison\n\nRuns in order of readBufferSize; baseline base.\n\n")
	require.Contains(t, md, "| run | setup |\n|:--|:--|\n| **base** | baseline · readBufferSize default")
	require.Contains(t, md, "| **4\\|MiB** | readBufferSize default → 4MiB |\n")
	require.Contains(t, md, "#### harness.wallNs · p50 per execution\n\nChange from base.\n\n")
	require.Contains(t, md, "| case | base | again | | 2MiB | | 4\\|MiB | |\n|:--|--:|--:|--:|--:|--:|--:|--:|\n")
	require.Contains(t, md, "| traceid/present | 100ns | 104ns | +4.0% | 98ns | -2.0% | 87ns | -13.0% |\n")
	require.Contains(t, md, "| search/nopredicate ⚠ | 10ns | 10ns | 0% | 10ns | 0% | not comparable | |\n")
	require.Contains(t, md, "\n- ⚠ search/nopredicate vs 4\\|MiB: matched 5 vs 6\n")
	require.NotContains(t, md, "traceByID", "headings are for the terminal; a markdown table stays flat")

	// Numbered runs are listed with their numbers.
	c.Runs[3].Name = "a-long-name-for-4MiB"
	b.Reset()
	require.NoError(t, WriteMarkdown(&b, c, []string{"harness.wallNs"}, P50, 0))
	require.Contains(t, b.String(), "| #4 **a-long-name-for-4MiB** | readBufferSize default → 4MiB |\n")
	require.Contains(t, b.String(), "| case | #1 | #2 | | #3 | | #4 | |\n")
	require.Contains(t, b.String(), "Change from #1.\n")
}
