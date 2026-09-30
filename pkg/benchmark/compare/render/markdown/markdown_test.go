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
	require.NoError(t, Write(&b, c, []string{"harness.wallNs"}, compare.P50, 0))
	require.Contains(t, b.String(), "| #4 **a-long-name-for-4MiB** | readBufferSize default → 4MiB |\n")
	require.Contains(t, b.String(), "| case | #1 | #2 | | #3 | | #4 | |\n")
	require.Contains(t, b.String(), "Change from #1.\n")
}
