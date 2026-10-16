package benchmark

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

// rankedAttributesPerGroup is how many attributes of each group — with a
// dedicated column and without — the generated cases cover.
const rankedAttributesPerGroup = 10

// tagValueAttributesPerGroup is how many string attributes of each group get a
// tag-value lookup.
const tagValueAttributesPerGroup = 20

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

const (
	rankedDedicated = "ranked_dedicated"
	rankedGeneric   = "ranked_generic"
)

func rankedGroup(p attributes.Profile) string {
	if p.Dedicated {
		return rankedDedicated
	}
	return rankedGeneric
}

// Family is the generated query a case ID runs, without the band and
// attribute it varies, like metrics/ranked_generic/rate: what benchmark
// compare rolls up into one row. A fixed case has none, since each asks a
// question of its own.
func Family(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) < 3 || (parts[1] != rankedDedicated && parts[1] != rankedGeneric) {
		return ""
	}
	parts = parts[:len(parts)-1]
	if b := parts[len(parts)-1]; b == bandHigh || b == bandMedium || b == bandLow {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, "/")
}

// hasValues skips numerics with too many values to list: their quantiles suit a
// range predicate, not `=`.
func hasValues(p attributes.Profile) bool {
	return len(p.Values) > 0
}

func isStringAttribute(p attributes.Profile) bool {
	return p.Type == attributes.TypeString
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

// attributeIDs use the name, not the rank, so an attribute keeps its ID across
// experiment variants. The type is added only to tell two picks of one name apart.
func attributeIDs(ps []attributes.Profile) []string {
	seen := make(map[string]int, len(ps))
	for _, p := range ps {
		seen[p.Scope+"."+p.Name]++
	}

	ids := make([]string, len(ps))
	for i, p := range ps {
		ids[i] = p.Scope + "." + p.Name
		if seen[ids[i]] > 1 {
			ids[i] += "." + p.Type
		}
	}
	return ids
}
