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

// Leads lead each run's line of description: its name, padded so the
// descriptions line up, after its number when runs are numbered to head
// columns, so the lines are the key to them.
func Leads(c *compare.Comparison) []string {
	names := c.Names()
	labels, numbered := Labels(c)
	width := nameWidth(names)
	leads := make([]string, len(names))
	for i, name := range names {
		leads[i] = fitName(name, width)
		if numbered {
			leads[i] = pad(labels[i], longest(labels)+1) + leads[i]
		}
	}
	return leads
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

// Detail is one piece of how a run is described, with the kind of setting it is
// about so a view can style it.
type Detail struct {
	Text string
	Kind compare.Kind
}

// Describe says how a run was set up against the baseline: what it changed, or,
// for the baseline itself, what the others are measured from. A change in where
// a run happened is flagged, since latencies from two environments are hard to
// compare.
func Describe(c *compare.Comparison, run, baseline int) []Detail {
	if run == baseline {
		details := []Detail{{Text: "baseline", Kind: compare.Setup}}
		for _, s := range c.BaselineSettings(baseline) {
			details = append(details, Detail{Text: s.String(), Kind: compare.Derived})
		}
		return details
	}

	chs := c.Changes(run, baseline)
	if len(chs) == 0 {
		return []Detail{{Text: "same setup as the baseline", Kind: compare.Derived}}
	}
	details := make([]Detail, len(chs))
	for i, ch := range chs {
		text := ch.String()
		if ch.Kind == compare.Environment {
			text = "⚠ " + text
		}
		details[i] = Detail{Text: text, Kind: ch.Kind}
	}
	return details
}

// DetailText runs a description's details together as plain text.
func DetailText(details []Detail) string {
	texts := make([]string, len(details))
	for i, d := range details {
		texts[i] = d.Text
	}
	return strings.Join(texts, " · ")
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
