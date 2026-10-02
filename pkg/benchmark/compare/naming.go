package compare

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// nameRuns names every run not named on the command line after what sets it
// apart: the options and build that vary between the runs, or, when only where
// they ran varies, that. Names do not depend on the baseline, so they stay put
// when it moves. It returns the fields any name came from.
func nameRuns(runs []Run, all [][]Setting, vary []Setting) (namedBy []string) {
	fields := fieldNames(vary, Setup, Build)
	if len(fields) == 0 {
		fields = fieldNames(vary, Environment)
	}

	var unnamed []int
	counts := map[string]int{}
	for i := range runs {
		if runs[i].Name == "" {
			unnamed = append(unnamed, i)
			runs[i].Name = nameFor(all[i], fields, runs[i].File)
		}
		counts[runs[i].Name]++
	}

	// Runs set up alike, like repeats of one setup, would share a name, so they
	// are named after their files instead.
	for _, i := range unnamed {
		switch {
		case counts[runs[i].Name] > 1:
			runs[i].Name = fileStem(runs[i].File)
		case len(fields) > 0:
			namedBy = fields
		}
	}

	// Number any name still taken, leaving explicit names alone.
	taken := map[string]bool{}
	for i, r := range runs {
		if !slices.Contains(unnamed, i) {
			taken[r.Name] = true
		}
	}
	for _, i := range unnamed {
		name := runs[i].Name
		for n := 2; taken[name]; n++ {
			name = fmt.Sprintf("%s #%d", runs[i].Name, n)
		}
		runs[i].Name, taken[name] = name, true
	}
	return namedBy
}

func nameFor(ss []Setting, fields []string, file string) string {
	switch len(fields) {
	case 0:
		return fileStem(file)
	case 1:
		return lookup(ss, fields[0]).Value
	}
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = f + "=" + lookup(ss, f).Value
	}
	return strings.Join(parts, " ")
}

func fieldNames(ss []Setting, kinds ...Kind) []string {
	var names []string
	for _, s := range ss {
		if slices.Contains(kinds, s.Kind) {
			names = append(names, s.Field)
		}
	}
	return names
}

func fileStem(file string) string {
	if file == "" {
		return "run"
	}
	base := filepath.Base(file)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// orderOf is the field to put the runs in order of: the only option or build
// field that varies, when every run's value of it is a number or left at
// default.
func orderOf(all [][]Setting, vary []Setting) (string, bool) {
	fields := fieldNames(vary, Setup, Build)
	if len(fields) != 1 {
		return "", false
	}
	for _, ss := range all {
		if s := lookup(ss, fields[0]); !s.isNumber && s.Value != defaultValue {
			return "", false
		}
	}
	return fields[0], true
}

// order is the permutation that puts runs in order of a field, with those left
// at default first, since that is where an experiment usually starts from.
func order(all [][]Setting, field string) []int {
	perm := make([]int, len(all))
	for i := range perm {
		perm[i] = i
	}
	slices.SortStableFunc(perm, func(a, b int) int {
		sa, sb := lookup(all[a], field), lookup(all[b], field)
		if sa.isNumber != sb.isNumber {
			if sb.isNumber {
				return -1
			}
			return 1
		}
		return cmp.Compare(sa.number, sb.number)
	})
	return perm
}
