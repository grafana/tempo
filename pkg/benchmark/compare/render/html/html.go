// Package html serves a comparison as web pages, rendered on each request as
// pprof's web view is: a summary of every case for a metric, and a page per
// case with the box plots and tables of every metric. A family of cases the
// summary rolls up has a page of its own, the summary of only its cases. Links
// switch the metric, the percentile, and the baseline.
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
	s := &server{c: c, metrics: metrics, stat: stat, tmpl: tmpl, cases: map[string]int{}, families: map[string][]int{}}
	for i, cs := range c.Cases {
		s.cases[cs.ID] = i
		if f := compare.FamilyOf(cs.ID); f != "" {
			s.families[f] = append(s.families[f], i)
		}
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
	// families indexes Comparison.Cases by family, in case order.
	families map[string][]int
}

// view is what a page shows, as its query says. A family narrows both the
// summary and a case page's previous and next links to its cases.
type view struct {
	metric   int
	stat     compare.Stat
	baseline int
	family   string
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
	if f := q.Get("family"); f != "" {
		if _, ok := s.families[f]; !ok {
			return v, fmt.Errorf("no family %q", f)
		}
		v.family = f
	}
	return v, nil
}

func (s *server) values(v view) url.Values {
	q := url.Values{
		"metric":   {s.metrics[v.metric]},
		"p":        {v.stat.String()},
		"baseline": {strconv.Itoa(v.baseline)},
	}
	if v.family != "" {
		q.Set("family", v.family)
	}
	return q
}

// summaryURL and caseURL link to a page relative to the server's root, as the
// templates write every link: under "/", so that none can lead off the server.
func (s *server) summaryURL(v view) string {
	return "?" + s.values(v).Encode()
}

func (s *server) caseURL(v view, id string) string {
	q := s.values(v)
	q.Set("id", id)
	return "case?" + q.Encode()
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
		as := v
		as.baseline = i
		rv := runView{Label: r.Label, Name: text(r.Name), URL: link(as)}
		for _, d := range r.Details {
			rv.Details = append(rv.Details, text(d))
		}
		p.Runs = append(p.Runs, rv)
	}
	return p
}

type summaryView struct {
	Title string
	// Family and SummaryURL are set only on a family's page.
	Family     string
	SummaryURL string
	Metrics    []tabView
	Stats      []tabView
	Columns    []columnView
	Rows       []summaryRowView
	Notes      []string
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
	Cases    int
	Problems []string
	Cells    []cellView
}

type cellView struct {
	Value                  string
	Change, Range          textView
	Baseline, Incomparable bool
}

func (s *server) summary(w http.ResponseWriter, r *http.Request) {
	v, err := s.parseView(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	metric := s.metrics[v.metric]
	title := metric
	sv := &summaryView{}
	var sm render.SummaryView
	if v.family == "" {
		sm = render.NewSummary(s.c, metric, v.stat, v.baseline)
	} else {
		sm = render.NewCaseSummary(s.c, metric, v.stat, v.baseline, v.family)
		all := v
		all.family = ""
		sv.Family, sv.SummaryURL = compare.FamilyGlob(s.c.Cases[s.families[v.family][0]].ID), s.summaryURL(all)
		title += " · " + sv.Family
	}
	sv.Title, sv.Notes = sm.Title, sm.Notes()

	for i, m := range s.metrics {
		to := v
		to.metric = i
		sv.Metrics = append(sv.Metrics, tabView{Label: m, URL: s.summaryURL(to), Active: i == v.metric})
	}
	for _, st := range compare.Stats {
		to := v
		to.stat = st
		sv.Stats = append(sv.Stats, tabView{Label: st.String(), URL: s.summaryURL(to), Active: st == v.stat})
	}
	for run, col := range sm.Columns {
		sv.Columns = append(sv.Columns, columnView{textView: text(col), Baseline: run == v.baseline})
	}
	for _, row := range sm.Rows {
		if row.Group != "" {
			sv.Rows = append(sv.Rows, summaryRowView{Group: row.Group})
			continue
		}
		rv := summaryRowView{ID: row.ID, URL: s.caseURL(v, row.ID), Cases: row.Cases, Problems: row.Problems}
		if row.Family != "" {
			in := v
			in.family = row.Family
			rv.URL = s.summaryURL(in)
		}
		for run, cell := range row.Cells {
			rv.Cells = append(rv.Cells, cellView{
				Value: cell.Value, Change: text(cell.Change), Range: text(cell.Range),
				Baseline: run == v.baseline, Incomparable: cell.Incomparable,
			})
		}
		sv.Rows = append(sv.Rows, rv)
	}

	p := s.page(v, title, s.summaryURL)
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

	// From a family's page, the steps between cases stay in the family.
	steps := s.families[v.family]
	if v.family == "" {
		steps = make([]int, len(s.c.Cases))
		for j := range steps {
			steps[j] = j
		}
	}
	at := slices.Index(steps, i)
	if at < 0 {
		http.Error(w, fmt.Sprintf("case %q is not in family %q", cs.ID, v.family), http.StatusNotFound)
		return
	}

	cv := &caseView{ID: cs.ID, Query: cs.Query, Problems: render.Problems(s.c, cs, v.baseline), SummaryURL: s.summaryURL(v)}
	if at > 0 {
		cv.PrevURL = s.caseURL(v, s.c.Cases[steps[at-1]].ID)
	}
	if at < len(steps)-1 {
		cv.NextURL = s.caseURL(v, s.c.Cases[steps[at+1]].ID)
	}
	names := s.c.Names()
	for _, metric := range s.metrics {
		series := cs.Series(metric)
		cv.Metrics = append(cv.Metrics, caseMetricView{
			Metric: metric,
			Plot:   boxPlotSVG(newBoxPlot(names, series, v.baseline)),
			Header: tableHeader,
			Rows:   tableRows(names, series, v.baseline),
		})
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
		v.Color = color(t)
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

// css colours the styles' classes from the palette.
func css() template.CSS {
	var b strings.Builder
	for _, s := range []render.Style{render.Dim, render.Warn, render.Better, render.Worse} {
		fmt.Fprintf(&b, ".%s { color: %s; }\n", class(s), color(render.Text{Style: s}))
	}
	b.WriteString(".bold { font-weight: 600; }\n")
	// Every value in it is a class name or a colour from the palette.
	return template.CSS(b.String()) // #nosec G203
}
