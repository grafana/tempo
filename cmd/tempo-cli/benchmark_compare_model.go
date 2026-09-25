package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
)

var (
	// Runs are told apart by colour, which stays clear of the blue and orange
	// changes are read in.
	compareRunColors = []color.Color{
		lipgloss.Color("170"), // magenta
		lipgloss.Color("78"),  // green
		lipgloss.Color("220"), // yellow
		lipgloss.Color("45"),  // cyan
		lipgloss.Color("141"), // purple
		lipgloss.Color("205"), // pink
		lipgloss.Color("37"),  // teal
		lipgloss.Color("180"), // tan
	}

	compareTitleStyle = lipgloss.NewStyle().Bold(true)
	compareDimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	compareWarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	// Reverse video rather than a colour keeps the selection visible on light
	// and dark themes alike.
	compareSelectedStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
	compareTabStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Padding(0, 1)
	compareActiveStyle   = lipgloss.NewStyle().Reverse(true).Bold(true).Padding(0, 1)

	// Changes read blue when better and orange when worse, which holds for most
	// colour-blind eyes too, and bold when large. Every default metric is
	// better lower.
	compareBetterText = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	compareWorseText  = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
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

// benchmarkCompareModel opens on a summary of every case, benchstat-style, and
// drills into one case as box plots over a table.
type benchmarkCompareModel struct {
	cmp     *compare.Comparison
	names   []string
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
		names:    c.Names(),
		metrics:  metrics,
		stat:     int(stat),
		baseline: c.Baseline,
	}
}

func (m *benchmarkCompareModel) Init() tea.Cmd {
	return nil
}

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
		m.baseline = (m.baseline + 1) % len(m.names)
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
	footer := compareDimStyle.Render(compareClip(help, m.width))
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
	lines := []string{fit.Render(
		compareTitleStyle.Render("tempo-cli benchmark compare") + "  " + compareDimStyle.Render(m.cmp.RunsHeading(m.baseline)),
	)}

	// One line per run, saying how it differs from the baseline. The lead is in
	// the run's colour, so this doubles as the key to the tables and plots.
	sep := compareDimStyle.Render(" · ")
	for i, lead := range m.cmp.RunLeads() {
		details := m.cmp.Describe(i, m.baseline)
		parts := make([]string, len(details))
		for j, d := range details {
			parts[j] = compareDetailStyle(d.Kind).Render(d.Text)
		}
		lines = append(lines, fit.Render(m.runStyle(i).Render(lead)+strings.Join(parts, sep)))
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
	sm := m.cmp.Summary(m.metrics[m.metric], compare.Stats[m.stat], m.baseline)
	header, rows := sm.Layout()
	fit := lipgloss.NewStyle().MaxWidth(m.width)

	top := []string{
		compareTitleStyle.Render(compareClip(sm.Title(), m.width)),
		"",
		fit.Render("  " + m.renderLine(header)),
	}
	selected := m.cmp.Cases[m.selected]
	var bottom []string
	for _, p := range m.cmp.Problems(selected, m.baseline) {
		bottom = append(bottom, compareWarnStyle.Render(compareClip("⚠ "+selected.ID+" "+p, m.width)))
	}

	lines := make([]string, 0, len(rows))
	selectedLine := 0
	for _, r := range rows {
		if r.Case < 0 {
			lines = append(lines, fit.Render(" "+m.renderLine(r.Line)))
			continue
		}
		marker, label := "  ", r.Line[0].Text
		if r.Case == m.selected {
			selectedLine = len(lines)
			marker, label = "▸ ", compareSelectedStyle.Render(label)
		}
		lines = append(lines, fit.Render(marker+label+m.renderLine(r.Line[1:])))
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

// renderLine styles each segment by what it shows.
func (m *benchmarkCompareModel) renderLine(l compare.Line) string {
	var b strings.Builder
	for _, s := range l {
		b.WriteString(m.segmentStyle(s).Render(s.Text))
	}
	return b.String()
}

func (m *benchmarkCompareModel) segmentStyle(s compare.Segment) lipgloss.Style {
	switch s.Kind {
	case compare.DimSegment:
		return compareDimStyle
	case compare.WarnSegment:
		return compareWarnStyle
	case compare.RunSegment:
		return m.runStyle(s.Run)
	case compare.ChangeSegment:
		return compareChangeText(s.Delta)
	default:
		return lipgloss.NewStyle()
	}
}

// compareChangeText colours a change by which way it went, bold when it went
// far.
func compareChangeText(pct float64) lipgloss.Style {
	style := compareWorseText
	if pct < 0 {
		style = compareBetterText
	}
	return style.Bold(math.Abs(pct) >= compare.MajorChange)
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
			flag = " ⚠"
		}
		id := compareClip(cs.ID, width-2-utf8.RuneCountInString(flag))
		if i == m.selected {
			lines = append(lines, compareSelectedStyle.Render("▸ "+id)+compareWarnStyle.Render(flag))
			continue
		}
		lines = append(lines, "  "+id+compareWarnStyle.Render(flag))
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func (m *benchmarkCompareModel) renderDetail(width int) string {
	cs := m.cmp.Cases[m.selected]
	metric := m.metrics[m.metric]
	s := cs.Series(metric)

	lines := []string{compareTitleStyle.Render(compareClip(compare.Title(cs, metric), width))}
	if cs.Query != "" {
		lines = append(lines, compareDimStyle.Render(compareClip("query: "+cs.Query, width)))
	}
	lines = append(lines, "")

	plot := compare.BoxPlot(m.names, s, width)
	for _, h := range plot.Header {
		lines = append(lines, compareDimStyle.Render(h))
	}
	for i, row := range plot.Rows {
		lines = append(lines, m.runStyle(i).Render(row))
	}
	lines = append(lines, "")

	header, rows := compare.Table(m.names, s, m.baseline)
	lines = append(lines, compareDimStyle.Render(compareClip(header, width)))
	for i, row := range rows {
		lines = append(lines, m.runStyle(i).Render(compareClip(row, width)))
	}
	lines = append(lines, "")

	for _, p := range m.cmp.Problems(cs, m.baseline) {
		lines = append(lines, compareWarnStyle.Render(compareClip("⚠ "+p, width)))
	}
	lines = append(lines, compareDimStyle.Render(compareClip(compare.Legend, width)))
	return strings.Join(lines, "\n")
}

// compareDetailStyle styles a piece of a run's description by what it is about:
// what the experiment varies reads plainly, where it ran is flagged, and what
// followed from the rest is dimmed.
func compareDetailStyle(kind compare.Kind) lipgloss.Style {
	switch kind {
	case compare.Environment:
		return compareWarnStyle
	case compare.Derived:
		return compareDimStyle
	default:
		return lipgloss.NewStyle()
	}
}

// runStyle colours a run so its rows can be followed from plot to table. The
// baseline is bold, since every delta is measured from it.
func (m *benchmarkCompareModel) runStyle(i int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(compareRunColors[i%len(compareRunColors)]).
		Bold(i == m.baseline)
}

// compareClip cuts plain text to width runes, marking the cut.
func compareClip(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	r := []rune(s)
	return fmt.Sprintf("%s…", string(r[:width-1]))
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
