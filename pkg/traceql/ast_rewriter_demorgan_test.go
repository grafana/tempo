package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDeMorganRewriter documents which negations are pushed down to the comparisons and which are kept.
// A comparison with a missing attribute is false, and so is its negation, so { !(.a = "x") } is
// rewritten to { .a != "x" }, which doesn't match spans without .a either.
func TestDeMorganRewriter(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		want     string
		optCount int
	}{
		// convertible: operators with an exact complement for any attribute, array or not
		{name: "equal", query: "{ !(.a = `x`) }", want: "{ .a != `x` }", optCount: 1},
		{name: "not equal", query: "{ !(.a != `x`) }", want: "{ .a = `x` }", optCount: 1},
		{name: "regex", query: "{ !(.a =~ `x`) }", want: "{ .a !~ `x` }", optCount: 1},
		{name: "not regex", query: "{ !(.a !~ `x`) }", want: "{ .a =~ `x` }", optCount: 1},
		{name: "equal with int", query: "{ !(.a = 1) }", want: "{ .a != 1 }", optCount: 1},
		{name: "equal with static on the left", query: "{ !(`x` = .a) }", want: "{ `x` != .a }", optCount: 1},
		{name: "scoped span attribute", query: "{ !(span.a = `x`) }", want: "{ span.a != `x` }", optCount: 1},
		{name: "scoped resource attribute", query: "{ !(resource.a = `x`) }", want: "{ resource.a != `x` }", optCount: 1},
		{name: "intrinsic", query: "{ !(name = `x`) }", want: "{ name != `x` }", optCount: 1},
		{name: "service name", query: "{ !(resource.service.name = `x`) }", want: "{ resource.service.name != `x` }", optCount: 1},
		{name: "status", query: "{ !(status = error) }", want: "{ status != error }", optCount: 1},

		// convertible: nil checks are never false for both
		{name: "not exists", query: "{ !(.a != nil) }", want: "{ .a = nil }", optCount: 1},
		{name: "not not exists", query: "{ !(.a = nil) }", want: "{ .a != nil }", optCount: 1},
		{name: "not exists scoped", query: "{ !(span.a != nil) }", want: "{ span.a = nil }", optCount: 1},
		{name: "not exists intrinsic", query: "{ !(name != nil) }", want: "{ name = nil }", optCount: 1},

		// convertible: ordering operators only for intrinsics that are never arrays
		{name: "duration greater", query: "{ !(duration > 1s) }", want: "{ duration <= 1s }", optCount: 1},
		{name: "duration greater or equal", query: "{ !(duration >= 1s) }", want: "{ duration < 1s }", optCount: 1},
		{name: "duration less", query: "{ !(duration < 1s) }", want: "{ duration >= 1s }", optCount: 1},
		{name: "duration less or equal", query: "{ !(duration <= 1s) }", want: "{ duration > 1s }", optCount: 1},
		{name: "name ordering", query: "{ !(name > `x`) }", want: "{ name <= `x` }", optCount: 1},
		{name: "scoped span duration", query: "{ !(span:duration > 1s) }", want: "{ duration <= 1s }", optCount: 1},
		{name: "scoped span name", query: "{ !(span:name > `x`) }", want: "{ name <= `x` }", optCount: 1},
		{name: "trace duration", query: "{ !(trace:duration > 1s) }", want: "{ traceDuration <= 1s }", optCount: 1},
		{name: "trace root name", query: "{ !(trace:rootName > `x`) }", want: "{ rootName <= `x` }", optCount: 1},
		{name: "trace root service", query: "{ !(trace:rootService > `x`) }", want: "{ rootServiceName <= `x` }", optCount: 1},
		{name: "status message", query: "{ !(statusMessage > `x`) }", want: "{ statusMessage <= `x` }", optCount: 1},
		{name: "child count", query: "{ !(span:childCount > 1) }", want: "{ span:childCount <= 1 }", optCount: 1},
		{name: "nested set left", query: "{ !(nestedSetLeft > 1) }", want: "{ nestedSetLeft <= 1 }", optCount: 1},
		{name: "instrumentation name", query: "{ !(instrumentation:name > `x`) }", want: "{ instrumentation:name <= `x` }", optCount: 1},
		{name: "instrumentation version", query: "{ !(instrumentation:version > `x`) }", want: "{ instrumentation:version <= `x` }", optCount: 1},

		// convertible: De Morgan's laws
		{name: "and", query: "{ !(.a = `x` && .b = `y`) }", want: "{ (.a != `x`) || (.b != `y`) }", optCount: 1},
		{name: "or", query: "{ !(.a = `x` || .b = `y`) }", want: "{ (.a != `x`) && (.b != `y`) }", optCount: 1},
		{name: "nested", query: "{ !(.a = `x` && (.b = `y` || .c = `z`)) }", want: "{ (.a != `x`) || ((.b != `y`) && (.c != `z`)) }", optCount: 1},
		{name: "double negation", query: "{ !(!(.a = `x`)) }", want: "{ .a = `x` }", optCount: 2},
		{name: "negation inside and", query: "{ duration > 0 && !(.a = `x`) }", want: "{ (duration > 0) && (.a != `x`) }", optCount: 1},
		{name: "negation inside or", query: "{ .a = `y` || !(.a = `x`) }", want: "{ (.a = `y`) || (.a != `x`) }", optCount: 1},
		{name: "duration range", query: "{ !(duration > 1s && duration < 2s) }", want: "{ (duration <= 1s) || (duration >= 2s) }", optCount: 1},

		// partially convertible: the comparisons that can't be flipped keep their negation
		{name: "and with ordering attribute", query: "{ !(.a = `x` && .b > 1) }", want: "{ (.a != `x`) || (!(.b > 1)) }", optCount: 1},
		{name: "or with exists", query: "{ !(.a = `x` || .b != nil) }", want: "{ (.a != `x`) && (.b = nil) }", optCount: 1},

		// not convertible
		{
			// attributes can be arrays, which match when any element matches: no element > 1 is not any element <= 1
			name: "attribute greater", query: "{ !(.a > 1) }", want: "{ !(.a > 1) }", optCount: 0,
		},
		{name: "attribute greater or equal", query: "{ !(.a >= 1) }", want: "{ !(.a >= 1) }", optCount: 0},
		{name: "attribute less", query: "{ !(.a < 1) }", want: "{ !(.a < 1) }", optCount: 0},
		{name: "attribute less or equal", query: "{ !(.a <= 1) }", want: "{ !(.a <= 1) }", optCount: 0},
		{name: "scoped attribute ordering", query: "{ !(span.a > 1) }", want: "{ !(span.a > 1) }", optCount: 0},
		{name: "intrinsic and attribute ordering", query: "{ !(duration > .a) }", want: "{ !(duration > .a) }", optCount: 0},
		{
			// resource.service.name is an attribute in the query. It could be something other than a flat string.
			name: "service name ordering", query: "{ !(resource.service.name > `x`) }", want: "{ !(resource.service.name > `x`) }", optCount: 0,
		},
		{name: "kind ordering", query: "{ !(kind > server) }", want: "{ !(kind > server) }", optCount: 0},
		{name: "status ordering", query: "{ !(status > error) }", want: "{ !(status > error) }", optCount: 0},
		{name: "event attribute exists", query: "{ !(event.a != nil) }", want: "{ !(event.a != nil) }", optCount: 0},
		{name: "event attribute", query: "{ !(event.a = `x`) }", want: "{ !(event.a = `x`) }", optCount: 0},
		{name: "event name", query: "{ !(event:name = `x`) }", want: "{ !(event:name = `x`) }", optCount: 0},
		{name: "link attribute", query: "{ !(link.a = `x`) }", want: "{ !(link.a = `x`) }", optCount: 0},
		{name: "link span id", query: "{ !(link:spanID = `x`) }", want: "{ !(link:spanID = `x`) }", optCount: 0},
		{name: "two attributes", query: "{ !(.a = .b) }", want: "{ !(.a = .b) }", optCount: 0},
		{name: "arithmetic", query: "{ !(.a + 1 = 2) }", want: "{ !((.a + 1) = 2) }", optCount: 0},
		{name: "attribute alone", query: "{ !.a }", want: "{ !.a }", optCount: 0},
		{name: "without negation", query: "{ .a = `x` && .b = `y` }", want: "{ (.a = `x`) && (.b = `y`) }", optCount: 0},

		// negations in other parts of the query
		{name: "second filter of a pipeline", query: "{ .a = `x` } | { !(.b = `y`) }", want: "{ .a = `x` }|{ .b != `y` }", optCount: 1},
		{name: "spanset operation", query: "{ !(.a = `x`) } && { !(.b = `y`) }", want: "({ .a != `x` }) && ({ .b != `y` })", optCount: 2},
		{name: "aggregate", query: "{ !(.a = `x`) } | count() > 1", want: "{ .a != `x` }|(count()) > 1", optCount: 1},
		{name: "select", query: "{ !(.a = `x`) } | select(.b)", want: "{ .a != `x` }|select(.b)", optCount: 1},
	}

	rewriter := newDeMorganRewriter()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := ParseNoOptimizations(tc.query)
			require.NoError(t, err)

			rewrite := rewriter.RewriteRoot(expr)
			require.Equal(t, tc.want, rewrite.String())
			require.Equal(t, tc.optCount, rewrite.OptimizationCount)
		})
	}
}

// TestDeMorganRewriterAfterOrToIn checks the full default chain. or_to_in runs first and creates
// IN and NOT IN, which de Morgan then flips.
func TestDeMorganRewriterAfterOrToIn(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  string
	}{
		{name: "not or", query: "{ !(.a = `x` || .a = `y`) }", want: "{ .a NOT IN [`x`, `y`] }"},
		{name: "not and", query: "{ !(.a != `x` && .a != `y`) }", want: "{ .a IN [`x`, `y`] }"},
		{name: "not or regex", query: "{ !(.a =~ `x` || .a =~ `y`) }", want: "{ .a MATCH NONE [`x`, `y`] }"},
		{name: "not and regex", query: "{ !(.a !~ `x` && .a !~ `y`) }", want: "{ .a MATCH ANY [`x`, `y`] }"},
		{name: "or beside not", query: "{ .a = `x` || .a = `y` || !(.b = `z`) }", want: "{ (.a IN [`x`, `y`]) || (.b != `z`) }"},
		{name: "different attributes", query: "{ !(.a = `x` || .b = `y`) }", want: "{ (.a != `x`) && (.b != `y`) }"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := Parse(tc.query)
			require.NoError(t, err)
			require.Equal(t, tc.want, expr.String())
		})
	}
}
