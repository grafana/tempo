package process

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Filter returns all elements of the list that match at least one expression
// and are not ignored
func Filter(list manifest.List, exprs Matchers) manifest.List {
	out := make(manifest.List, 0, len(list))
	for _, m := range list {
		if !exprs.MatchString(m.KindName()) {
			continue
		}
		if exprs.IgnoreString(m.KindName()) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// Matcher is a single filter expression. The passed argument of Matcher is of the
// form `kind/name` (manifest.KindName())
type Matcher interface {
	MatchString(string) bool
}

// Ignorer is like matcher, but for explicitly ignoring resources
type Ignorer interface {
	IgnoreString(string) bool
}

// Matchers is a collection of multiple expressions.
// A matcher may also implement Ignorer to explicitly ignore fields
type Matchers []Matcher

// MatchString returns whether at least one expression (OR) matches the string
func (e Matchers) MatchString(s string) bool {
	b := false
	for _, exp := range e {
		b = b || exp.MatchString(s)
	}
	return b
}

func (e Matchers) IgnoreString(s string) bool {
	b := false
	for _, exp := range e {
		i, ok := exp.(Ignorer)
		if !ok {
			continue
		}
		b = b || i.IgnoreString(s)
	}
	return b
}

// RegExps is a helper to construct Matchers from regular expressions
func RegExps(rs []*regexp.Regexp) Matchers {
	xprs := make(Matchers, 0, len(rs))
	for _, r := range rs {
		xprs = append(xprs, r)
	}
	return xprs
}

func StrExps(strs ...string) (Matchers, error) {
	exps := make(Matchers, 0, len(strs))
	for _, raw := range strs {
		// trim exlamation mark, not supported by regex
		s := fmt.Sprintf(`(?i)^%s$`, strings.TrimPrefix(raw, "!"))

		// create regexp matcher
		var exp Matcher
		exp, err := regexp.Compile(s)
		if err != nil {
			return nil, ErrBadExpr{err}
		}

		// if negative (!), invert regex behaviour
		if strings.HasPrefix(raw, "!") {
			exp = NegMatcher{exp: exp}
		}
		exps = append(exps, exp)
	}
	return exps, nil
}

func MustStrExps(strs ...string) Matchers {
	exps, err := StrExps(strs...)
	if err != nil {
		panic(err)
	}
	return exps
}

// ErrBadExpr occurs when the regexp compiling fails
type ErrBadExpr struct {
	inner error
}

func (e ErrBadExpr) Error() string {
	caser := cases.Title(language.English)
	return fmt.Sprintf("%s.\nSee https://tanka.dev/output-filtering/#regular-expressions for details on regular expressions.", caser.String(e.inner.Error()))
}

// NexMatcher is a matcher that inverts the original behaviour
type NegMatcher struct {
	exp Matcher
}

func (n NegMatcher) MatchString(_ string) bool {
	return true
}

func (n NegMatcher) IgnoreString(s string) bool {
	return n.exp.MatchString(s)
}
