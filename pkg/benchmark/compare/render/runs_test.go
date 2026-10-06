package render

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
)

func TestLabels(t *testing.T) {
	c := &compare.Comparison{Runs: []compare.Run{{Name: "default"}, {Name: "4MiB"}}}
	labels, numbered := Labels(c)
	require.False(t, numbered)
	require.Equal(t, []string{"default", "4MiB"}, labels)

	c.Runs = append(c.Runs, compare.Run{Name: "readBufferSize=4MiB"})
	labels, numbered = Labels(c)
	require.True(t, numbered, "one name too long numbers every run, so they read alike")
	require.Equal(t, []string{"#1", "#2", "#3"}, labels)
}

func TestHeading(t *testing.T) {
	c := &compare.Comparison{Runs: []compare.Run{{Name: "a"}, {Name: "b"}}}
	require.Equal(t, "runs; baseline a", Heading(c, 0))

	c.NamedBy = []string{"searchLimit", "readBufferSize"}
	require.Equal(t, "runs named by searchLimit, readBufferSize; baseline b", Heading(c, 1))

	c.OrderedBy = "readBufferSize"
	require.Equal(t, "runs named by searchLimit, readBufferSize in order of readBufferSize; baseline a", Heading(c, 0))

	c.NamedBy = []string{"readBufferSize"}
	require.Equal(t, "runs named by readBufferSize, in its order; baseline a", Heading(c, 0))
}

func TestNewRuns(t *testing.T) {
	base := comparetest.Result("mac")
	same := comparetest.Result("mac")
	other := comparetest.Result("linux")
	other.Options.ReadBufferSize = 4 << 20
	other.Shards = 110
	c, err := compare.New([]compare.Run{{Name: "base", Result: base}, {Name: "same", Result: same}, {Name: "other", Result: other}})
	require.NoError(t, err)

	v := NewRuns(c, 0)
	require.Equal(t, "runs in order of readBufferSize; baseline base", v.Heading)
	require.False(t, v.Numbered)
	require.Equal(t, Text{Text: "base", Style: BaselineName, Run: 0}, v.Runs[0].Name)
	require.Equal(t, Text{Text: "other", Style: RunName, Run: 2}, v.Runs[2].Name)
	require.Equal(t, "base", v.Runs[0].Label)

	// The baseline shows what the others are measured from, receding behind
	// the word that says it is the baseline.
	require.Equal(t,
		"baseline · readBufferSize default · hostname mac · shards 55 · gitSHA abc · goVersion go1.27 · goMaxProcs 12",
		v.Runs[0].Description())
	require.Equal(t, Plain, v.Runs[0].Details[0].Style)
	require.Equal(t, Dim, v.Runs[0].Details[1].Style)

	require.Equal(t, []Text{{Text: "same setup as the baseline", Style: Dim}}, v.Runs[1].Details)

	// What the experiment varies reads plainly, a change in where a run
	// happened is flagged, and what followed from the rest recedes.
	require.Equal(t, []Text{
		{Text: "readBufferSize default → 4MiB"},
		{Text: "⚠ hostname mac → linux", Style: Warn},
		{Text: "shards 55 → 110", Style: Dim},
	}, v.Runs[2].Details)

	// Against another baseline, the changes are from it instead.
	require.Equal(t, "readBufferSize 4MiB → default · ⚠ hostname linux → mac · shards 110 → 55", NewRuns(c, 2).Runs[0].Description())
}

func TestProblems(t *testing.T) {
	c := comparetest.Comparison(t)
	require.Empty(t, Problems(c, c.Cases[0], 0))
	require.Equal(t, []string{"vs 4MiB: matched 5 vs 6"}, Problems(c, c.Cases[1], 0))
	require.Equal(t, []string{"vs base: matched 6 vs 5", "vs again: matched 6 vs 5", "vs 2MiB: matched 6 vs 5"}, Problems(c, c.Cases[1], 3))
}
