package render

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
)

// maxLabel is the longest run name that heads a column. Past it, runs are
// numbered instead, and listed with their numbers.
const maxLabel = 8

// Labels head the runs' columns: their names, or their numbers in order when
// any name is too long to. Labels do not depend on the baseline, so they stay
// put when it moves.
func Labels(c *compare.Comparison) (labels []string, numbered bool) {
	names := c.Names()
	for _, n := range names {
		if utf8.RuneCountInString(n) > maxLabel {
			numbered = true
		}
	}
	if !numbered {
		return names, false
	}
	labels = make([]string, len(names))
	for i := range names {
		labels[i] = fmt.Sprintf("#%d", i+1)
	}
	return labels, true
}

// Heading says how the runs were named and ordered, and which one the others
// are compared against.
func Heading(c *compare.Comparison, baseline int) string {
	s := "runs"
	if len(c.NamedBy) > 0 {
		s += " named by " + strings.Join(c.NamedBy, ", ")
	}
	if c.OrderedBy != "" {
		if slices.Equal(c.NamedBy, []string{c.OrderedBy}) {
			s += ", in its order"
		} else {
			s += " in order of " + c.OrderedBy
		}
	}
	return s + "; baseline " + c.Runs[baseline].Name
}

// RunsView says how the runs differ: how they were named and ordered, and how
// each was set up against the baseline.
type RunsView struct {
	Heading string
	// Numbered says the runs head columns by number, so each is listed with
	// its number, as the key to them.
	Numbered bool
	Runs     []RunLine
}

// RunLine is one run: its name, and how it was set up against the baseline.
type RunLine struct {
	// Name is the run's name, shown as the run.
	Name Text
	// Label heads the run's columns.
	Label   string
	Details []Text
}

// Description runs a line's details together as plain text.
func (r RunLine) Description() string {
	texts := make([]string, len(r.Details))
	for i, d := range r.Details {
		texts[i] = d.Text
	}
	return strings.Join(texts, " · ")
}

// NewRuns describes every run against the baseline.
func NewRuns(c *compare.Comparison, baseline int) RunsView {
	labels, numbered := Labels(c)
	v := RunsView{Heading: Heading(c, baseline), Numbered: numbered}
	for i, name := range c.Names() {
		v.Runs = append(v.Runs, RunLine{Name: RunText(name, i, baseline), Label: labels[i], Details: describe(c, i, baseline)})
	}
	return v
}

// describe says how a run was set up against the baseline: what it changed, or,
// for the baseline itself, what the others are measured from. What an
// experiment varies reads plainly, a change in where a run happened is
// flagged, since latencies from two environments are hard to compare, and
// what followed from the rest recedes.
func describe(c *compare.Comparison, run, baseline int) []Text {
	if run == baseline {
		details := []Text{{Text: "baseline"}}
		for _, s := range c.BaselineSettings(baseline) {
			details = append(details, Text{Text: s.String(), Style: Dim})
		}
		return details
	}

	chs := c.Changes(run, baseline)
	if len(chs) == 0 {
		return []Text{{Text: "same setup as the baseline", Style: Dim}}
	}
	details := make([]Text, len(chs))
	for i, ch := range chs {
		details[i] = Text{Text: ch.String()}
		switch ch.Kind {
		case compare.Environment:
			details[i] = Text{Text: Flag + " " + ch.String(), Style: Warn}
		case compare.Derived:
			details[i].Style = Dim
		}
	}
	return details
}

// Problems writes why runs cannot be compared with the baseline on a case, as
// "vs <run>: <reason>".
func Problems(c *compare.Comparison, cs compare.Case, baseline int) []string {
	labels, _ := Labels(c)
	var out []string
	for _, p := range c.Problems(cs, baseline) {
		out = append(out, "vs "+labels[p.Run]+": "+p.Reason)
	}
	return out
}

// MatchedProblems is Problems over the runs' match counts alone, for a case
// totals summary, where a different execution count does not block the
// comparison.
func MatchedProblems(c *compare.Comparison, cs compare.Case, baseline int) []string {
	labels, _ := Labels(c)
	var out []string
	for _, p := range c.MatchedProblems(cs, baseline) {
		out = append(out, "vs "+labels[p.Run]+": "+p.Reason)
	}
	return out
}
