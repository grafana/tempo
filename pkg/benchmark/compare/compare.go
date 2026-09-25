// Package compare lines benchmark results up against each other and draws them
// as text, so a person can see what changed between two or more runs.
//
// It holds no terminal state: every renderer returns plain strings, or lines
// of segments saying what each piece shows, which an interactive view styles
// and a static one prints as they are.
package compare

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// Run is one benchmark result under the name it is compared by.
type Run struct {
	// Name is empty until New names the run, when it was not named explicitly.
	Name string
	// File is where the result was read from, if it was.
	File   string
	Result *benchmark.Result
}

// LoadRun reads the result an argument names, as "name=path" or a bare path.
// A bare path is left unnamed, for New to name after what sets it apart.
func LoadRun(arg string) (Run, error) {
	name, file := parseRunArg(arg)
	f, err := os.Open(file)
	if err != nil {
		return Run{}, fmt.Errorf("opening result %s: %w", file, err)
	}
	defer f.Close()

	result, err := benchmark.LoadResult(f)
	if err != nil {
		return Run{}, fmt.Errorf("reading result %s: %w", file, err)
	}
	return Run{Name: name, File: file, Result: result}, nil
}

// parseRunArg splits "name=path" into its parts. An = after a path separator
// is part of the path.
func parseRunArg(arg string) (name, file string) {
	if n, f, ok := strings.Cut(arg, "="); ok && n != "" && !strings.ContainsRune(n, filepath.Separator) {
		return n, f
	}
	return "", arg
}

// Case is one query shape across every run.
type Case struct {
	ID    string
	API   string
	Query string
	// Results holds the case from each run, in run order, nil where a run does
	// not have it.
	Results []*benchmark.CaseResult
}

// Comparison is a set of runs with their cases lined up by ID.
type Comparison struct {
	Runs  []Run
	Cases []Case
	// Baseline is the run the others are compared against to begin with: the
	// one given first.
	Baseline int
	// Varying names the settings that are not the same in every run.
	Varying []string
	// NamedBy names the settings runs without a name were named after.
	NamedBy []string
	// OrderedBy names the setting the runs were put in order of, if any.
	OrderedBy string

	settings [][]Setting
}

// New lines runs up by case. Runs without a name are named after what sets them
// apart, and when that is one number, like a buffer size, they are put in its
// order. The run given first stays the baseline wherever it lands.
//
// Cases appear in the order the baseline lists them, followed by any only other
// runs have.
func New(runs []Run) (*Comparison, error) {
	if len(runs) < 2 {
		return nil, errors.New("at least two results are needed to compare")
	}
	seen := make(map[string]bool, len(runs))
	for _, r := range runs {
		if r.Name != "" && seen[r.Name] {
			return nil, fmt.Errorf("two results are named %q, name them apart with name=path", r.Name)
		}
		seen[r.Name] = true
	}

	runs = slices.Clone(runs)
	all := make([][]Setting, len(runs))
	for i, r := range runs {
		ss, err := settings(r.Result)
		if err != nil {
			return nil, fmt.Errorf("reading the settings of %s: %w", r.File, err)
		}
		all[i] = ss
	}
	vary := varying(all)

	c := &Comparison{NamedBy: nameRuns(runs, all, vary)}
	for _, v := range vary {
		c.Varying = append(c.Varying, v.Field)
	}

	perm := make([]int, len(runs))
	for i := range perm {
		perm[i] = i
	}
	if field, ok := orderOf(all, vary); ok {
		perm = order(all, field)
		c.OrderedBy = field
	}
	for i, p := range perm {
		c.Runs = append(c.Runs, runs[p])
		c.settings = append(c.settings, all[p])
		if p == 0 {
			c.Baseline = i
		}
	}

	c.alignCases()
	return c, nil
}

// alignCases lines the runs' cases up by ID, taking the baseline's order first.
func (c *Comparison) alignCases() {
	index := map[string]int{}
	add := func(i int) {
		for j := range c.Runs[i].Result.Cases {
			cr := &c.Runs[i].Result.Cases[j]
			k, ok := index[cr.ID]
			if !ok {
				k = len(c.Cases)
				index[cr.ID] = k
				c.Cases = append(c.Cases, Case{
					ID:      cr.ID,
					API:     cr.API,
					Query:   cr.Query,
					Results: make([]*benchmark.CaseResult, len(c.Runs)),
				})
			}
			c.Cases[k].Results[i] = cr
		}
	}
	add(c.Baseline)
	for i := range c.Runs {
		if i != c.Baseline {
			add(i)
		}
	}
}

// Names lists the runs' names in order.
func (c *Comparison) Names() []string {
	names := make([]string, len(c.Runs))
	for i, r := range c.Runs {
		names[i] = r.Name
	}
	return names
}

// FilterCases keeps the cases whose ID matches any pattern, in path.Match
// syntax. No patterns keeps every case.
func (c *Comparison) FilterCases(patterns []string) error {
	if len(patterns) == 0 {
		return nil
	}
	var kept []Case
	for _, cs := range c.Cases {
		ok, err := matchAny(patterns, cs.ID)
		if err != nil {
			return err
		}
		if ok {
			kept = append(kept, cs)
		}
	}
	if len(kept) == 0 {
		return fmt.Errorf("no case matches %s", strings.Join(patterns, ", "))
	}
	c.Cases = kept
	return nil
}

// Metrics returns the metric keys any case reports that match the patterns, in
// path.Match syntax. They come in pattern order, sorted within a pattern.
func (c *Comparison) Metrics(patterns []string) ([]string, error) {
	var all []string
	seen := map[string]bool{}
	for _, cs := range c.Cases {
		for _, cr := range cs.Results {
			if cr == nil {
				continue
			}
			for key := range cr.Metrics {
				if !seen[key] {
					seen[key] = true
					all = append(all, key)
				}
			}
		}
	}
	slices.Sort(all)

	var out []string
	picked := map[string]bool{}
	for _, p := range patterns {
		for _, key := range all {
			ok, err := path.Match(p, key)
			if err != nil {
				return nil, fmt.Errorf("invalid metric pattern %q: %w", p, err)
			}
			if ok && !picked[key] {
				picked[key] = true
				out = append(out, key)
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no metric matches %s", strings.Join(patterns, ", "))
	}
	return out, nil
}

// Series is one metric of one case across every run.
type Series struct {
	Unit Unit
	// Summaries holds each run's per-execution summary, in run order, nil where
	// a run did not report the metric.
	Summaries []*metrics.Summary
}

// Series picks out one metric of the case.
func (cs Case) Series(metric string) Series {
	s := Series{Unit: UnitFor(metric), Summaries: make([]*metrics.Summary, len(cs.Results))}
	for i, cr := range cs.Results {
		if cr == nil {
			continue
		}
		if m, ok := cr.Metrics[metric]; ok && m.Summary.Count > 0 {
			s.Summaries[i] = &m.Summary
		}
	}
	return s
}

// Incomparable says why a run's results for the case cannot be compared with
// the baseline's, or nothing when they can.
//
// A different match count means the two runs answered different questions,
// and a different execution count means their per-execution numbers measure
// different amounts of work, as when a shard size changes.
func (cs Case) Incomparable(run, baseline int) string {
	base, r := cs.Results[baseline], cs.Results[run]
	switch {
	case base == nil:
		return "missing from the baseline"
	case r == nil:
		return "missing"
	case base.Error != "":
		return "the baseline failed: " + base.Error
	case r.Error != "":
		return "failed: " + r.Error
	case base.Matched != r.Matched:
		return fmt.Sprintf("matched %d vs %d", base.Matched, r.Matched)
	case base.Executions != r.Executions:
		return fmt.Sprintf("executions %d vs %d", base.Executions, r.Executions)
	}
	return ""
}

// Problems says, for each run that cannot be compared with the baseline on the
// case, why not, as "vs <run>: <why>".
func (c *Comparison) Problems(cs Case, baseline int) []string {
	labels, _ := c.RunLabels()
	var out []string
	for run := range c.Runs {
		if run == baseline {
			continue
		}
		if why := cs.Incomparable(run, baseline); why != "" {
			out = append(out, "vs "+labels[run]+": "+why)
		}
	}
	return out
}

// delta is the change from base to v, in percent. It is not ok when base is
// zero, since no change from nothing is a percentage.
func delta(base, v float64) (pct float64, ok bool) {
	if base == 0 {
		return 0, false
	}
	return (v - base) / base * 100, true
}

func matchAny(patterns []string, s string) (bool, error) {
	for _, p := range patterns {
		ok, err := path.Match(p, s)
		if err != nil {
			return false, fmt.Errorf("invalid pattern %q: %w", p, err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}
