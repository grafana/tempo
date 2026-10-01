// Package html serves a comparison as web pages, rendered on each request as
// pprof's web view is: a summary of every case for a metric, and a page per
// case with the box plots and tables of every metric. Links switch the metric,
// the percentile, and the baseline.
package html

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

//go:embed templates
var templates embed.FS

// Handler serves a comparison's pages, starting from the stat given.
func Handler(c *compare.Comparison, metrics []string, stat compare.Stat) (http.Handler, error) {
	tmpl, err := template.ParseFS(templates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing page templates: %w", err)
	}
	s := &server{c: c, metrics: metrics, stat: stat, tmpl: tmpl, cases: map[string]int{}}
	for i, cs := range c.Cases {
		s.cases[cs.ID] = i
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.summary)
	mux.HandleFunc("GET /case", s.casePage)
	return mux, nil
}

type server struct {
	c       *compare.Comparison
	metrics []string
	stat    compare.Stat
	tmpl    *template.Template
	cases   map[string]int
}

// view is what a page shows, as its query says.
type view struct {
	metric   int
	stat     compare.Stat
	baseline int
}

func (s *server) parseView(q url.Values) (view, error) {
	v := view{stat: s.stat, baseline: s.c.Baseline}
	if m := q.Get("metric"); m != "" {
		if v.metric = slices.Index(s.metrics, m); v.metric < 0 {
			return v, fmt.Errorf("unknown metric %q", m)
		}
	}
	if p := q.Get("p"); p != "" {
		stat, err := compare.ParseStat(p)
		if err != nil {
			return v, err
		}
		v.stat = stat
	}
	if b := q.Get("baseline"); b != "" {
		i, err := strconv.Atoi(b)
		if err != nil || i < 0 || i >= len(s.c.Runs) {
			return v, fmt.Errorf("no run %q to be the baseline", b)
		}
		v.baseline = i
	}
	return v, nil
}

func (s *server) values(v view) url.Values {
	return url.Values{
		"metric":   {s.metrics[v.metric]},
		"p":        {v.stat.String()},
		"baseline": {strconv.Itoa(v.baseline)},
	}
}

func (s *server) summaryURL(v view) string {
	return "/?" + s.values(v).Encode()
}

func (s *server) caseURL(v view, id string) string {
	q := s.values(v)
	q.Set("id", id)
	return "/case?" + q.Encode()
}

// page is what every page shows, how the runs differ, above its own content.
type page struct {
	Title    string
	CSS      template.CSS
	Heading  string
	Numbered bool
	Runs     []runView
	Summary  *summaryView
	Case     *caseView
}

type runView struct {
	Label   string
	Name    textView
	Details []textView
	// URL shows the same page with the run as the baseline.
	URL string
}

func (s *server) page(v view, title string, link func(view) string) page {
	runs := render.NewRuns(s.c, v.baseline)
	p := page{Title: title, CSS: css(), Heading: runs.Heading, Numbered: runs.Numbered}
	for i, r := range runs.Runs {
		rv := runView{Label: r.Label, Name: text(r.Name), URL: link(view{metric: v.metric, stat: v.stat, baseline: i})}
		for _, d := range r.Details {
			rv.Details = append(rv.Details, text(d))
		}
		p.Runs = append(p.Runs, rv)
	}
	return p
}

type summaryView struct {
	Title   string
	Metrics []tabView
	Stats   []tabView
	Columns []columnView
	Rows    []summaryRowView
	Notes   []string
}

type tabView struct {
	Label, URL string
	Active     bool
}

type columnView struct {
	textView
	Baseline bool
}

type summaryRowView struct {
	Group    string
	ID, URL  string
	Problems []string
	Cells    []cellView
}

type cellView struct {
	Value                  string
	Change                 textView
	Baseline, Incomparable bool
}

func (s *server) summary(w http.ResponseWriter, r *http.Request) {
	v, err := s.parseView(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	metric := s.metrics[v.metric]
	sm := render.NewSummary(s.c, metric, v.stat, v.baseline)

	sv := &summaryView{Title: sm.Title, Notes: sm.Notes()}
	for i, m := range s.metrics {
		sv.Metrics = append(sv.Metrics, tabView{Label: m, URL: s.summaryURL(view{metric: i, stat: v.stat, baseline: v.baseline}), Active: i == v.metric})
	}
	for _, st := range compare.Stats {
		sv.Stats = append(sv.Stats, tabView{Label: st.String(), URL: s.summaryURL(view{metric: v.metric, stat: st, baseline: v.baseline}), Active: st == v.stat})
	}
	for run, col := range sm.Columns {
		sv.Columns = append(sv.Columns, columnView{textView: text(col), Baseline: run == v.baseline})
	}
	for _, row := range sm.Rows {
		if row.Group != "" {
			sv.Rows = append(sv.Rows, summaryRowView{Group: row.Group})
			continue
		}
		rv := summaryRowView{ID: row.ID, URL: s.caseURL(v, row.ID), Problems: row.Problems}
		for run, cell := range row.Cells {
			rv.Cells = append(rv.Cells, cellView{
				Value: cell.Value, Change: text(cell.Change),
				Baseline: run == v.baseline, Incomparable: cell.Incomparable,
			})
		}
		sv.Rows = append(sv.Rows, rv)
	}

	p := s.page(v, metric, s.summaryURL)
	p.Summary = sv
	s.render(w, "summary", p)
}

type caseView struct {
	ID, Query        string
	Problems         []string
	SummaryURL       string
	PrevURL, NextURL string
	Metrics          []caseMetricView
}

type caseMetricView struct {
	Metric string
	Plot   template.HTML
	Header []string
	Rows   []tableRowView
}

type tableRowView struct {
	Name  textView
	Cells []string
}

func (s *server) casePage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	v, err := s.parseView(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	i, ok := s.cases[q.Get("id")]
	if !ok {
		http.Error(w, fmt.Sprintf("no case %q", q.Get("id")), http.StatusNotFound)
		return
	}
	cs := s.c.Cases[i]

	cv := &caseView{ID: cs.ID, Query: cs.Query, SummaryURL: s.summaryURL(v)}
	if i > 0 {
		cv.PrevURL = s.caseURL(v, s.c.Cases[i-1].ID)
	}
	if i < len(s.c.Cases)-1 {
		cv.NextURL = s.caseURL(v, s.c.Cases[i+1].ID)
	}
	for _, metric := range s.metrics {
		mv := render.NewCase(s.c, i, metric, v.baseline)
		cv.Problems = mv.Problems
		m := caseMetricView{Metric: metric, Plot: boxPlotSVG(mv.Plot), Header: mv.Table.Header}
		for _, row := range mv.Table.Rows {
			m.Rows = append(m.Rows, tableRowView{Name: text(row.Name), Cells: row.Cells})
		}
		cv.Metrics = append(cv.Metrics, m)
	}

	p := s.page(v, cs.ID, func(v view) string { return s.caseURL(v, cs.ID) })
	p.Case = cv
	s.render(w, "case", p)
}

// render writes a page whole, or an error in its place.
func (s *server) render(w http.ResponseWriter, name string, p page) {
	var b bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&b, name, p); err != nil {
		http.Error(w, "rendering page: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = b.WriteTo(w) // a client that went away needs nothing more
}

// textView is a piece of a view as a page shows it: a class for its style,
// and a colour of its own for a piece that shows a run.
type textView struct {
	Text, Class, Color string
}

func text(t render.Text) textView {
	v := textView{Text: t.Text, Class: class(t.Style)}
	if t.Style == render.RunName || t.Style == render.BaselineName {
		c, _ := t.Color()
		v.Color = c.Hex
	}
	return v
}

// class names a style in the page's CSS.
func class(s render.Style) string {
	var classes []string
	switch s {
	case render.Dim:
		classes = append(classes, "dim")
	case render.Warn:
		classes = append(classes, "warn")
	case render.Better, render.MuchBetter:
		classes = append(classes, "better")
	case render.Worse, render.MuchWorse:
		classes = append(classes, "worse")
	}
	if s.Bold() {
		classes = append(classes, "bold")
	}
	return strings.Join(classes, " ")
}

// css colours the styles' classes from the one palette every output shares.
func css() template.CSS {
	var b strings.Builder
	for _, s := range []render.Style{render.Dim, render.Warn, render.Better, render.Worse} {
		c, _ := render.Text{Style: s}.Color()
		fmt.Fprintf(&b, ".%s { color: %s; }\n", class(s), c.Hex)
	}
	b.WriteString(".bold { font-weight: 600; }\n")
	// Every value in it is a class name or a colour from the palette.
	return template.CSS(b.String()) // #nosec G203
}
