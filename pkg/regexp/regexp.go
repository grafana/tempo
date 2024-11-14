package regexp

import (
	"fmt"
	"unsafe"

	"github.com/prometheus/prometheus/model/labels"
)

// jpe - test
type Regexp struct {
	matchers    []*labels.FastRegexMatcher
	matches     map[string]bool
	shouldMatch bool
}

func NewRegexp(regexps []string, shouldMatch bool) (*Regexp, error) {
	matchers := make([]*labels.FastRegexMatcher, 0, len(regexps))

	for _, r := range regexps {
		m, err := labels.NewFastRegexMatcher(r)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, m)
	}

	// only memoize if there's a unoptimized matcher
	// TODO: should we limit memoization?
	var matches map[string]bool
	for _, m := range matchers {
		if !m.IsOptimized() {
			matches = make(map[string]bool)
			break
		}
	}

	return &Regexp{
		matchers:    matchers,
		matches:     matches,
		shouldMatch: shouldMatch,
	}, nil
}

func (r *Regexp) Match(b []byte) bool {
	return r.MatchString(unsafe.String(unsafe.SliceData(b), len(b)))
}

func (r *Regexp) MatchString(s string) bool {
	// if we're memoizing check existing matches
	if r.matches != nil {
		if matched, ok := r.matches[s]; ok {
			return matched
		}
	}

	matched := false
	for _, m := range r.matchers {
		if m.MatchString(s) == r.shouldMatch {
			matched = true
			break
		}
	}

	if r.matches != nil {
		r.matches[s] = matched
	}

	return matched
}

func (r *Regexp) Reset() {
	if r.matches != nil {
		clear(r.matches)
	}
}

func (r *Regexp) String() string {
	var strings string
	for _, m := range r.matchers {
		strings += fmt.Sprintf("%s, ", m.GetRegexString())
	}

	return strings
}
