package render

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
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

func TestLeads(t *testing.T) {
	r := newResult("mac")
	c, err := compare.New([]compare.Run{{Name: "base", Result: r}, {Name: "4MiB", Result: r}})
	require.NoError(t, err)
	require.Equal(t, []string{"base  ", "4MiB  "}, Leads(c))

	// Runs numbered to head columns are listed with their numbers, as the key.
	c.Runs[1].Name = "a-longer-name"
	require.Equal(t, []string{"#1 base           ", "#2 a-longer-name  "}, Leads(c))
}

func TestHeading(t *testing.T) {
	c := &compare.Comparison{Runs: []compare.Run{{Name: "a"}, {Name: "b"}}}
	require.Equal(t, "runs; baseline a", Heading(c, 0))

	c.NamedBy = []string{"readBufferSize", "readBufferCount"}
	require.Equal(t, "runs named by readBufferSize, readBufferCount; baseline b", Heading(c, 1))

	c.OrderedBy = "readBufferSize"
	require.Equal(t, "runs named by readBufferSize, readBufferCount in order of readBufferSize; baseline a", Heading(c, 0))

	c.NamedBy = []string{"readBufferSize"}
	require.Equal(t, "runs named by readBufferSize, in its order; baseline a", Heading(c, 0))
}

func TestDescribe(t *testing.T) {
	base := newResult("mac")
	same := newResult("mac")
	other := newResult("linux")
	other.Options.ReadBufferSize = 4 << 20
	c, err := compare.New([]compare.Run{{Name: "base", Result: base}, {Name: "same", Result: same}, {Name: "other", Result: other}})
	require.NoError(t, err)

	// The baseline shows what the others are measured from.
	require.Equal(t,
		"baseline · readBufferSize default · hostname mac · gitSHA abc · goVersion go1.27 · goMaxProcs 12",
		DetailText(Describe(c, 0, 0)))

	require.Equal(t, []Detail{{Text: "same setup as the baseline", Kind: compare.Derived}}, Describe(c, 1, 0))

	// A change in where the run happened is flagged.
	require.Equal(t, []Detail{
		{Text: "readBufferSize default → 4MiB", Kind: compare.Setup},
		{Text: "⚠ hostname mac → linux", Kind: compare.Environment},
	}, Describe(c, 2, 0))

	// Against another baseline, the changes are from it instead.
	require.Equal(t, "readBufferSize 4MiB → default · ⚠ hostname linux → mac", DetailText(Describe(c, 0, 2)))
}

func TestProblems(t *testing.T) {
	c := newSummaryComparison(t)
	require.Empty(t, Problems(c, c.Cases[0], 0))
	require.Equal(t, []string{"vs 4MiB: matched 5 vs 6"}, Problems(c, c.Cases[1], 0))
	require.Equal(t, []string{"vs base: matched 6 vs 5", "vs again: matched 6 vs 5", "vs 2MiB: matched 6 vs 5"}, Problems(c, c.Cases[1], 3))
}
