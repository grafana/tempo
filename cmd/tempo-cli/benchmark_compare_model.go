package main

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render/term"
)

var (
	compareTitleStyle = term.Style(render.Title)
	compareDimStyle   = term.Style(render.Dim)
	compareWarnStyle  = term.Style(render.Warn)
	// Reverse video rather than a colour keeps the selection visible on light
	// and dark themes alike.
	compareSelectedStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
	compareTabStyle      = term.Style(render.Dim).Padding(0, 1)
	compareActiveStyle   = lipgloss.NewStyle().Reverse(true).Bold(true).Padding(0, 1)
)

const (
	compareSummaryHelp = "↑/↓ case  ←/→ metric  p percentile  enter details  b baseline  q quit"
	compareDetailHelp  = "↑/↓ case  ←/→ metric  b baseline  esc summary  q quit"
)

type compareScreen int

const (
	// compareSummary is one metric of every case, as benchstat lays it out.
	compareSummary compareScreen = iota
	// compareDetail is one case of one metric, as box plots over a table.
	compareDetail
)

var _ tea.Model = (*benchmarkCompareModel)(nil)

// benchmarkCompareModel opens on a summary of every case, benchstat-style, and
// drills into one case as box plots over a table. It holds what is on screen
// and follows the keys; package render decides what a screen shows, and
// package term lays it out and styles it.
type benchmarkCompareModel struct {
	cmp     *compare.Comparison
	metrics []string

	screen   compareScreen
	selected int
	// metric is the one the summary shows and a case opens on.
	metric   int
	stat     int
	baseline int

	width, height int
}

func newBenchmarkCompareModel(c *compare.Comparison, metrics []string, stat compare.Stat) *benchmarkCompareModel {
	return &benchmarkCompareModel{
		cmp:      c,
		metrics:  metrics,
		stat:     int(stat),
		baseline: c.Baseline,
	}
}

// Init implements tea.Model
func (m *benchmarkCompareModel) Init() tea.Cmd { return nil }

func (m *benchmarkCompareModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

func (m *benchmarkCompareModel) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "q", "ctrl+c":
		return tea.Quit
	case "esc":
		if m.screen == compareSummary {
			return tea.Quit
		}
		m.screen = compareSummary
	case "enter":
		m.screen = compareDetail
	case "p":
		m.stat = (m.stat + 1) % len(compare.Stats)
	case "up", "k":
		m.selected = max(0, m.selected-1)
	case "down", "j":
		m.selected = min(len(m.cmp.Cases)-1, m.selected+1)
	case "home", "g":
		m.selected = 0
	case "end", "G": // nolint: goconst // a key name, unrelated to the column headers that also say "end"
		m.selected = len(m.cmp.Cases) - 1
	case "right", "l", "tab":
		m.metric = (m.metric + 1) % len(m.metrics)
	case "left", "h", "shift+tab":
		m.metric = (m.metric - 1 + len(m.metrics)) % len(m.metrics)
	case "b":
		m.baseline = (m.baseline + 1) % len(m.cmp.Runs)
	}
	return nil
}

func (m *benchmarkCompareModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("loading...")
	}

	help := compareSummaryHelp
	if m.screen == compareDetail {
		help = compareDetailHelp
	}
	header := m.renderHeader()
	footer := compareDimStyle.Render(term.Clip(help, m.width))
	bodyHeight := max(1, m.height-lipgloss.Height(header)-lipgloss.Height(footer))

	var body string
	if m.screen == compareSummary {
		body = compareFit(m.renderSummary(bodyHeight), bodyHeight)
	} else {
		listWidth := m.listWidth()
		detailWidth := max(1, m.width-listWidth-3)
		list := m.renderList(listWidth, bodyHeight)
		detail := compareFit(m.renderDetail(detailWidth), bodyHeight)
		sep := compareDimStyle.Render(strings.TrimSuffix(strings.Repeat(" │ \n", bodyHeight), "\n"))
		body = lipgloss.JoinHorizontal(lipgloss.Top, list, sep, detail)
	}

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, body, footer))
	v.AltScreen = true
	return v
}

func (m *benchmarkCompareModel) renderHeader() string {
	fit := lipgloss.NewStyle().MaxWidth(m.width)
	runs := render.NewRuns(m.cmp, m.baseline)
	lines := []string{fit.Render(
		compareTitleStyle.Render("tempo-cli benchmark compare") + "  " + compareDimStyle.Render(runs.Heading),
	)}

	// One line per run, saying how it differs from the baseline. The lead is in
	// the run's colour, so this doubles as the key to the tables and plots.
	for _, l := range term.Runs(runs) {
		lines = append(lines, fit.Render(term.Paint(l)))
	}
	lines = append(lines, "")

	tab := func(label string, active bool) string {
		if active {
			return compareActiveStyle.Render(label)
		}
		return compareTabStyle.Render(label)
	}
	tabs := make([]string, len(m.metrics))
	for i, metric := range m.metrics {
		tabs[i] = tab(metric, i == m.metric)
	}
	row := strings.Join(tabs, " ")
	// Only the summary shows one percentile, so only it offers the choice.
	if m.screen == compareSummary {
		stats := make([]string, len(compare.Stats))
		for i, s := range compare.Stats {
			stats[i] = tab(s.String(), i == m.stat)
		}
		row += "    " + strings.Join(stats, "")
	}
	lines = append(lines, fit.Render(row), "")
	return strings.Join(lines, "\n")
}

// renderSummary lays one metric of every case out as benchstat does, keeps the
// selected case in view, and says under it why it is flagged, if it is.
func (m *benchmarkCompareModel) renderSummary(height int) string {
	v := render.NewSummary(m.cmp, m.metrics[m.metric], compare.Stats[m.stat], m.baseline)
	st := term.NewSummaryTable(v)
	fit := lipgloss.NewStyle().MaxWidth(m.width)

	top := []string{
		compareTitleStyle.Render(term.Clip(v.Title, m.width)),
		"",
		fit.Render("  " + term.Paint(st.Header)),
	}
	var bottom []string
	for _, r := range v.Rows {
		if r.Case == m.selected {
			for _, p := range r.Problems {
				bottom = append(bottom, compareWarnStyle.Render(term.Clip(render.Flag+" "+r.ID+" "+p, m.width)))
			}
		}
	}

	lines := make([]string, 0, len(st.Rows))
	selectedLine := 0
	for _, r := range st.Rows {
		if r.Case < 0 {
			lines = append(lines, fit.Render(" "+term.Paint(r.Line)))
			continue
		}
		marker, label := "  ", r.Line[0].Text
		if r.Case == m.selected {
			selectedLine = len(lines)
			marker, label = "▸ ", compareSelectedStyle.Render(label)
		}
		lines = append(lines, fit.Render(marker+label+term.Paint(r.Line[1:])))
	}

	// The title and the column names stay put while the rows scroll.
	first, last := compareScroll(selectedLine, len(lines), height-len(top)-len(bottom))
	return strings.Join(append(append(top, lines[first:last]...), bottom...), "\n")
}

// compareScroll is the window of n rows, visible at a time, that keeps the
// selected one in view.
func compareScroll(selected, n, visible int) (first, last int) {
	visible = max(1, visible)
	first = max(0, selected-visible+1)
	return first, min(n, first+visible)
}

// listWidth fits the longest case ID with its markers, up to a third of the
// screen.
func (m *benchmarkCompareModel) listWidth() int {
	longest := 0
	for _, cs := range m.cmp.Cases {
		longest = max(longest, utf8.RuneCountInString(cs.ID))
	}
	return max(10, min(longest+4, m.width/3))
}

// renderList shows the cases that fit around the selected one. A case the runs
// may not be comparable on is flagged.
func (m *benchmarkCompareModel) renderList(width, height int) string {
	first, last := compareScroll(m.selected, len(m.cmp.Cases), height)
	lines := make([]string, 0, height)
	for i := first; i < last; i++ {
		cs := m.cmp.Cases[i]
		flag := ""
		if len(m.cmp.Problems(cs, m.baseline)) > 0 {
			flag = " " + render.Flag
		}
		id := term.Clip(cs.ID, width-2-utf8.RuneCountInString(flag))
		if i == m.selected {
			lines = append(lines, compareSelectedStyle.Render("▸ "+id)+compareWarnStyle.Render(flag))
			continue
		}
		lines = append(lines, "  "+id+compareWarnStyle.Render(flag))
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func (m *benchmarkCompareModel) renderDetail(width int) string {
	v := render.NewCase(m.cmp, m.selected, m.metrics[m.metric], m.baseline)

	lines := []string{compareTitleStyle.Render(term.Clip(v.Title, width))}
	if v.Query != "" {
		lines = append(lines, compareDimStyle.Render(term.Clip("query: "+v.Query, width)))
	}
	lines = append(lines, "")

	plot := term.BoxPlot(v.Plot, width)
	for _, l := range append(plot.Header, plot.Rows...) {
		lines = append(lines, term.Paint(l))
	}
	lines = append(lines, "")

	header, rows := term.Table(v.Table, width)
	for _, l := range append([]term.Line{header}, rows...) {
		lines = append(lines, term.Paint(l))
	}
	lines = append(lines, "")

	for _, p := range v.Problems {
		lines = append(lines, compareWarnStyle.Render(term.Clip(render.Flag+" "+p, width)))
	}
	lines = append(lines, compareDimStyle.Render(term.Clip(term.Legend, width)))
	return strings.Join(lines, "\n")
}

// compareFit keeps the first height lines of s, so a view taller than the
// screen does not scroll the header away.
func compareFit(s string, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
