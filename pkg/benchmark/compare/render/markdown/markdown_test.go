package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
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
	require.NotContains(t, b.String(), "<details>", "nothing to fold away without a family to roll up")
}

func TestWriteFamilies(t *testing.T) {
	var b strings.Builder
	require.NoError(t, Write(&b, comparetest.Family(t), []string{"harness.wallNs"}, compare.P50, 0))
	md := b.String()

	summary, every, ok := strings.Cut(md, "\n<details>\n<summary>Every case</summary>\n\n")
	require.True(t, ok, "the cases of a rolled-up family are folded away")
	require.Contains(t, summary, "| search/ranked_generic/*/* (9 cases) ⚠ | 10ns | 10.7ns | +7.4% [-75.0% … +300.0%] |\n")
	require.Contains(t, summary, "- ⚠ search/ranked_generic/*/* leaves 1 of 9 cases out of its means\n")
	require.NotContains(t, summary, "search/ranked_generic/high/0")

	require.Contains(t, every, "| search/ranked_generic/high/0 ")
	require.Contains(t, every, "- ⚠ search/ranked_generic/low/8 vs next: matched 1 vs 2\n")
	require.True(t, strings.HasSuffix(every, "\n</details>\n"))
}
