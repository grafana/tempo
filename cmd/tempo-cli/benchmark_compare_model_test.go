package main

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestBenchmarkCompareModel(t *testing.T) {
	m := newBenchmarkCompareModel(newTestComparison(t), []string{"harness.wallNs", "backend.reads"}, compare.P50)

	require.Equal(t, "loading...", m.View().Content, "nothing is drawn before the size is known")

	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	fits := func(view string) {
		t.Helper()
		for _, line := range strings.Split(view, "\n") {
			require.LessOrEqual(t, lipgloss.Width(line), 120, "line %q is wider than the screen", line)
		}
		require.LessOrEqual(t, len(strings.Split(view, "\n")), 30, "the view is taller than the screen")
	}
	press := func(keys ...tea.KeyPressMsg) {
		for _, k := range keys {
			m.Update(k)
		}
	}
	key := func(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }
	down := tea.KeyPressMsg{Code: tea.KeyDown}
	up := tea.KeyPressMsg{Code: tea.KeyUp}
	right := tea.KeyPressMsg{Code: tea.KeyRight}
	left := tea.KeyPressMsg{Code: tea.KeyLeft}
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	esc := tea.KeyPressMsg{Code: tea.KeyEscape}

	// It opens on a summary laid out as benchstat does: the baseline's value,
	// then each run's value and change.
	view := compareViewText(m)
	fits(view)
	require.Contains(t, view, "runs in order of readBufferSize; baseline base")
	require.Contains(t, view, "base  baseline · readBufferSize default · gitSHA unknown")
	require.Contains(t, view, "next  readBufferSize default → 4MiB")
	require.Contains(t, view, "harness.wallNs · p50 per execution · change from base")
	require.NotContains(t, view, "noise")
	require.Contains(t, view, "  case               │ base │      next")
	require.Contains(t, view, "\n traceByID ")
	require.Contains(t, view, "▸ traceid/present    │ 32ms │ 27.8ms  -13.0%")
	require.Contains(t, view, "  search/nopredicate │ 10ms │  8.7ms  -13.0%")

	press(key('p'))
	require.Contains(t, compareViewText(m), "harness.wallNs · p90 per execution")
	require.Contains(t, compareViewText(m), "▸ traceid/present    │ 44.8ms │   39ms  -13.0%")
	press(key('p'), key('p'))
	require.Contains(t, compareViewText(m), "harness.wallNs · p50 per execution", "percentiles wrap around")

	press(right)
	require.Contains(t, compareViewText(m), "backend.reads · p50 per execution")
	press(left)
	require.Contains(t, compareViewText(m), "harness.wallNs · p50 per execution")

	// Enter opens the selected case on the metric in focus, and escape comes
	// back with the selection kept.
	press(down, enter)
	view = compareViewText(m)
	fits(view)
	require.Contains(t, view, "search/nopredicate · harness.wallNs · per execution")
	require.Contains(t, view, "query: {}")
	require.Contains(t, view, "esc summary")
	press(esc)
	require.Contains(t, compareViewText(m), "harness.wallNs · p50 per execution")
	require.Contains(t, compareViewText(m), "▸ search/nopredicate")

	press(up, enter, down)
	require.Contains(t, compareViewText(m), "search/nopredicate · harness.wallNs")
	press(down, down)
	require.Equal(t, 1, m.selected, "the selection stops at the last case")
	press(up, up)
	require.Equal(t, 0, m.selected, "and at the first")

	press(right)
	require.Contains(t, compareViewText(m), "traceid/present · backend.reads")
	press(right)
	require.Contains(t, compareViewText(m), "traceid/present · harness.wallNs", "metrics wrap around")
	press(left)
	require.Contains(t, compareViewText(m), "traceid/present · backend.reads", "both ways")

	press(key('b'))
	view = compareViewText(m)
	require.Contains(t, view, "baseline next")
	require.Contains(t, view, "base  readBufferSize 4MiB → default", "changes are from the new baseline")

	_, cmd := m.Update(key('q'))
	require.NotNil(t, cmd)
	require.IsType(t, tea.QuitMsg{}, cmd())

	// Escape quits from the summary, since there is nothing to go back to.
	_, cmd = m.Update(esc)
	require.Nil(t, cmd, "escape leaves the details first")
	_, cmd = m.Update(esc)
	require.NotNil(t, cmd)
	require.IsType(t, tea.QuitMsg{}, cmd())
}

func TestBenchmarkCompareModelNotComparable(t *testing.T) {
	c := newTestComparison(t)
	c.Runs[1].Result.Cases[1].Matched++
	c, err := compare.New(c.Runs)
	require.NoError(t, err)

	m := newBenchmarkCompareModel(c, []string{"harness.wallNs"}, compare.P50)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	// A run that answered differently is not compared, and the selected case
	// says why.
	view := compareViewText(m)
	require.Contains(t, view, "▸ search/nopredicate ⚠ │ 10ms │ not comparable")
	require.Contains(t, view, "⚠ search/nopredicate vs next: matched 1100 vs 1101")
}

func TestBenchmarkCompareModelSummaryScrolls(t *testing.T) {
	c := newTestComparison(t)
	for i := range 30 {
		cs := c.Cases[0]
		cs.ID = fmt.Sprintf("traceid/extra-%02d", i)
		c.Cases = append(c.Cases, cs)
	}
	m := newBenchmarkCompareModel(c, []string{"harness.wallNs"}, compare.P50)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	for range 25 {
		m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}

	view := compareViewText(m)
	require.LessOrEqual(t, len(strings.Split(view, "\n")), 20)
	require.Contains(t, view, "▸ traceid/extra-23", "the selected case stays in view")
	require.Contains(t, view, "harness.wallNs · p50 per execution", "the title stays put")
	require.NotContains(t, view, "traceid/present ", "the top has scrolled away")
}

func TestBenchmarkCompareModelNumbersLongNames(t *testing.T) {
	c := newTestComparison(t)
	c.Runs[1].Name = "readBufferSize=4MiB"
	m := newBenchmarkCompareModel(c, []string{"harness.wallNs"}, compare.P50)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Numbers head the columns, and the runs are listed with them as the key.
	view := compareViewText(m)
	require.Contains(t, view, "#1 base                 baseline")
	require.Contains(t, view, "#2 readBufferSize=4MiB  readBufferSize")
	require.Contains(t, view, "  case               │   #1 │       #2")
	require.Contains(t, view, "change from #1", "the baseline is named by its number too")
}

func TestCompareChangeText(t *testing.T) {
	// Better is blue and worse orange, bold past MajorChange.
	require.NotEqual(t, compareBetterText.GetForeground(), compareWorseText.GetForeground())
	require.Equal(t, compareBetterText.GetForeground(), compareChangeText(-5).GetForeground())
	require.Equal(t, compareWorseText.GetForeground(), compareChangeText(5).GetForeground())
	require.False(t, compareChangeText(-5).GetBold())
	require.True(t, compareChangeText(-13).GetBold())
}

func TestBenchmarkCompareModelSmallScreen(t *testing.T) {
	m := newBenchmarkCompareModel(newTestComparison(t), []string{"harness.wallNs"}, compare.P50)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})

	view := compareViewText(m)
	require.LessOrEqual(t, len(strings.Split(view, "\n")), 8, "the view is cut rather than scrolling the header away")
	require.Contains(t, view, "tempo-cli benchmark compare")
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;:?]*[a-zA-Z]`)

// compareViewText is the view as it reads, without its styling.
func compareViewText(m *benchmarkCompareModel) string {
	return ansiEscape.ReplaceAllString(m.View().Content, "")
}

func newTestComparison(t *testing.T) *compare.Comparison {
	t.Helper()

	result := func(scale float64) *benchmark.Result {
		m := func(p50 float64) metrics.Measurement {
			p50 *= scale
			return metrics.Measurement{Kind: metrics.Counter, Summary: metrics.Summary{
				Count: 10, Min: p50 / 2, P25: p50 * 0.8, P50: p50, P75: p50 * 1.2, P90: p50 * 1.4, P99: p50 * 1.6, Max: p50 * 2,
			}}
		}
		return &benchmark.Result{
			SchemaVersion: benchmark.ResultSchemaVersion,
			RunEnv:        benchmark.RunEnv{GoVersion: "go1.27", GoMaxProcs: 12, Hostname: "h"},
			Cases: []benchmark.CaseResult{
				{ID: "traceid/present", API: "traceByID", Executions: 10, Matched: 10, Metrics: metrics.Set{
					"harness.wallNs": m(32e6),
					"backend.reads":  m(200),
				}},
				{ID: "search/nopredicate", API: "search", Query: "{}", Executions: 55, Matched: 1100, Metrics: metrics.Set{
					"harness.wallNs": m(10e6),
				}},
			},
		}
	}

	base, next := result(1), result(0.87)
	next.Options.ReadBufferSize = 4 << 20
	c, err := compare.New([]compare.Run{{Name: "base", Result: base}, {Name: "next", Result: next}})
	require.NoError(t, err)
	return c
}
