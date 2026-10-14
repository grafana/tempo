package benchmark

import (
	"fmt"
	"strconv"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

// rankedAttributesPerGroup is how many attributes of each group — with a
// dedicated column and without — the generated cases cover.
const rankedAttributesPerGroup = 10

var profiledIntrinsics = []traceql.Intrinsic{
	traceql.IntrinsicName,
	traceql.IntrinsicStatus,
	traceql.IntrinsicKind,
	traceql.IntrinsicDuration,
}

func pickAttributes(attrs *attributes.Profiles, n int, keep func(attributes.Profile) bool) []attributes.Profile {
	var dedicated, generic []attributes.Profile
	for _, p := range attrs.Ranked {
		switch {
		case !keep(p) || !queryable(p):
		case p.Dedicated && len(dedicated) < n:
			dedicated = append(dedicated, p)
		case !p.Dedicated && len(generic) < n:
			generic = append(generic, p)
		}
	}
	return append(dedicated, generic...)
}

// hasValues skips numerics with too many values to list: their quantiles suit a
// range predicate, not `=`.
func hasValues(p attributes.Profile) bool {
	return len(p.Values) > 0
}

// queryable skips attributes a newer profile might rank that this run cannot
// yet express in TraceQL.
func queryable(p attributes.Profile) bool {
	_, ok := traceqlAttribute(p)
	return ok
}

func traceqlAttribute(p attributes.Profile) (traceql.Attribute, bool) {
	switch p.Scope {
	case attributes.ScopeResource:
		return traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, p.Name), true
	case attributes.ScopeSpan:
		return traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, p.Name), true
	case attributes.ScopeIntrinsic:
		for _, i := range profiledIntrinsics {
			if i.String() == p.Name {
				return traceql.NewIntrinsic(i), true
			}
		}
	}
	return traceql.Attribute{}, false
}

const (
	bandHigh   = "high"
	bandMedium = "medium"
	bandLow    = "low"
)

type bandedValue struct {
	band  string
	value attributes.Value
}

// bandValues bands relative to the attribute's own values: bands over all spans
// would leave most top-ranked attributes with only a low value.
func bandValues(p attributes.Profile) []bandedValue {
	vs := p.Values
	switch len(vs) {
	case 0:
		return nil
	case 1:
		return []bandedValue{{bandHigh, vs[0]}}
	case 2:
		return []bandedValue{{bandHigh, vs[0]}, {bandLow, vs[1]}}
	default:
		return []bandedValue{{bandHigh, vs[0]}, {bandMedium, vs[(len(vs)-1)/2]}, {bandLow, vs[len(vs)-1]}}
	}
}

// valueFilter quotes strings with strconv.Quote, which round-trips any value
// since the lexer reads literals with strconv.Unquote. Other types are profiled
// in TraceQL's own form.
func valueFilter(p attributes.Profile, v string) string {
	attr, _ := traceqlAttribute(p)
	if p.Type == attributes.TypeString {
		v = strconv.Quote(v)
	}
	return fmt.Sprintf("{ %s = %s }", attr, v)
}
