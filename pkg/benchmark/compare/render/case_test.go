package render

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
)

func TestNewCase(t *testing.T) {
	c := comparetest.Comparison(t)
	c.Cases[1].Query = "{}"

	v := NewCase(c, 1, "harness.wallNs", 0)
	require.Equal(t, "search/nopredicate · harness.wallNs · per execution", v.Title)
	require.Equal(t, "{}", v.Query)
	require.Equal(t, []string{"vs 4MiB: matched 5 vs 6"}, v.Problems)
	require.Len(t, v.Plot.Rows, 4)
	require.Len(t, v.Table.Rows, 4)
	require.Equal(t, BaselineName, v.Plot.Rows[0].Name.Style)

	// A metric the case does not report still has a row per run.
	v = NewCase(c, 1, "backend.reads", 2)
	require.Nil(t, v.Plot.Rows[0].Summary)
	require.Equal(t, BaselineName, v.Table.Rows[2].Name.Style)
}
