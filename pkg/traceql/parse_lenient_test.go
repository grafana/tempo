package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseLenient tests the lenient parser end-to-end.
// Expected values use the AST stringer format: backtick-quoted strings,
// spaces inside braces, and parenthesized sub-expressions.
func TestParseLenient(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		// Valid queries pass through to Parse() unchanged.
		{
			name:     "simple matcher",
			in:       `{ .foo = "bar" }`,
			expected: "{ .foo = `bar` }",
		},
		{
			name:     "multiple matchers",
			in:       `{ .a = 1 && .b = "two" }`,
			expected: "{ (.a = 1) && (.b = `two`) }",
		},
		{
			name:     "intrinsic",
			in:       `{ duration > 5s }`,
			expected: "{ duration > 5s }",
		},
		{
			name:     "empty spanset",
			in:       `{ }`,
			expected: "{ true }",
		},
		{
			name:     "empty spanset no space",
			in:       `{}`,
			expected: "{ true }",
		},

		// Incomplete matchers: operator removed, attribute kept as OpNone.
		{
			name:     "single incomplete matcher",
			in:       `{ .foo = }`,
			expected: "{ .foo }",
		},
		{
			name:     "incomplete after complete",
			in:       `{ .a = 1 && .b = }`,
			expected: "{ (.a = 1) && .b }",
		},
		{
			name:     "incomplete before complete",
			in:       `{ .a = && .b = 1 }`,
			expected: "{ .a && (.b = 1) }",
		},
		{
			name:     "multiple incomplete matchers",
			in:       `{ .a = && .b = && .c = 1 }`,
			expected: "{ (.a && .b) && (.c = 1) }",
		},
		{
			name:     "all matchers incomplete",
			in:       `{ .a = && .b = }`,
			expected: "{ .a && .b }",
		},
		{
			name:     "incomplete not-equal",
			in:       `{ .foo != }`,
			expected: "{ .foo }",
		},
		{
			name:     "incomplete with negation",
			in:       `{ !.foo = }`,
			expected: "{ !.foo }",
		},

		// Scoped attributes.
		{
			name:     "scoped attribute incomplete",
			in:       `{ resource.service.name = }`,
			expected: "{ resource.service.name }",
		},
		{
			name:     "mixed scopes incomplete",
			in:       `{ span.foo = "bar" && resource.baz = }`,
			expected: "{ (span.foo = `bar`) && resource.baz }",
		},
		{
			name:     "parent scoped incomplete",
			in:       `{ parent.span.foo = }`,
			expected: "{ parent.span.foo }",
		},
		{
			name:     "parent resource scoped incomplete",
			in:       `{ parent.resource.service.name = }`,
			expected: "{ parent.resource.service.name }",
		},
		{
			name:     "parent duration incomplete",
			in:       `{ parent.duration = }`,
			expected: "{ parent.duration }",
		},
		{
			name:     "event scoped attribute incomplete",
			in:       `{ event.foo = }`,
			expected: "{ event.foo }",
		},
		{
			name:     "link scoped attribute incomplete",
			in:       `{ link.foo = }`,
			expected: "{ link.foo }",
		},
		{
			name:     "instrumentation scoped attribute incomplete",
			in:       `{ instrumentation.foo = }`,
			expected: "{ instrumentation.foo }",
		},

		// Scoped intrinsics.
		{
			name:     "event:name incomplete",
			in:       `{ event:name = }`,
			expected: "{ event:name }",
		},
		{
			name:     "trace:duration incomplete",
			in:       `{ trace:duration = }`,
			expected: "{ traceDuration }",
		},
		{
			name:     "span:status incomplete",
			in:       `{ span:status = }`,
			expected: "{ status }",
		},
		{
			name:     "link:traceID incomplete",
			in:       `{ link:traceID = }`,
			expected: "{ link:traceID }",
		},
		{
			name:     "instrumentation:name incomplete",
			in:       `{ instrumentation:name = }`,
			expected: "{ instrumentation:name }",
		},

		// All comparison operators with incomplete matchers.
		{
			name:     "regex incomplete",
			in:       `{ .foo =~ }`,
			expected: "{ .foo }",
		},
		{
			name:     "not-regex incomplete",
			in:       `{ .foo !~ }`,
			expected: "{ .foo }",
		},
		{
			name:     "greater-than incomplete",
			in:       `{ .foo > }`,
			expected: "{ .foo }",
		},
		{
			name:     "greater-equal incomplete",
			in:       `{ .foo >= }`,
			expected: "{ .foo }",
		},
		{
			name:     "less-than incomplete",
			in:       `{ .foo < }`,
			expected: "{ .foo }",
		},
		{
			name:     "less-than-or-equal incomplete",
			in:       `{ .foo <= }`,
			expected: "{ .foo }",
		},
		{
			name:     "all comparison ops incomplete",
			in:       `{ .a > && .b < }`,
			expected: "{ .a && .b }",
		},
		{
			name:     "all comparison ops incomplete gte lte",
			in:       `{ .a >= && .b <= }`,
			expected: "{ .a && .b }",
		},
		{
			name:     "all comparison ops incomplete regex",
			in:       `{ .a =~ && .b !~ }`,
			expected: "{ .a && .b }",
		},

		// Value types preserved in complete matchers.
		{
			name:     "string value",
			in:       `{ .foo = "bar" }`,
			expected: "{ .foo = `bar` }",
		},
		{
			name:     "integer value",
			in:       `{ .foo = 200 }`,
			expected: "{ .foo = 200 }",
		},
		{
			name:     "duration value",
			in:       `{ duration > 5s }`,
			expected: "{ duration > 5s }",
		},
		{
			name:     "boolean value",
			in:       `{ .foo = true }`,
			expected: "{ .foo = true }",
		},
		{
			name:     "status value",
			in:       `{ status = error }`,
			expected: "{ status = error }",
		},
		{
			name:     "kind value",
			in:       `{ kind = server }`,
			expected: "{ kind = server }",
		},
		{
			name:     "nil value with incomplete",
			in:       `{ .foo = nil && .bar = }`,
			expected: "{ (.foo = nil) && .bar }",
		},
		{
			name:     "multiple nil values with incomplete",
			in:       `{ .a = nil && .b = nil && .c = }`,
			expected: "{ ((.a = nil) && (.b = nil)) && .c }",
		},
		{
			name:     "existence check with incomplete",
			in:       `{ .foo != nil && .bar = }`,
			expected: "{ (.foo != nil) && .bar }",
		},

		// OR with incomplete.
		{
			name:     "OR with incomplete",
			in:       `{ .a = "foo" || .b = }`,
			expected: "{ (.a = `foo`) || .b }",
		},

		// Parentheses cleanup.
		{
			name:     "parenthesized incomplete",
			in:       `{ (.a = ) && .b = 1 }`,
			expected: "{ .a && (.b = 1) }",
		},
		{
			name:     "all incomplete inside parens",
			in:       `{ (.a = && .b = ) && .c = 1 }`,
			expected: "{ (.a && .b) && (.c = 1) }",
		},
		{
			name:     "nested parens with incomplete",
			in:       `{ ((.a = ) && .b = 1) || .c = 2 }`,
			expected: "{ (.a && (.b = 1)) || (.c = 2) }",
		},
		{
			name:     "inner parens incomplete",
			in:       `{ (.a = 1 && (.b = )) && .c = 2 }`,
			expected: "{ ((.a = 1) && .b) && (.c = 2) }",
		},

		// Missing closing brace.
		{
			name:     "missing closing brace with incomplete",
			in:       `{ .a = 1 && .b =`,
			expected: "{ (.a = 1) && .b }",
		},
		{
			name:     "missing closing brace valid",
			in:       `{ .a = 1`,
			expected: "{ .a = 1 }",
		},
		{
			name:     "missing closing brace gte incomplete",
			in:       `{ .a = 1 && .b >=`,
			expected: "{ (.a = 1) && .b }",
		},
		{
			name:     "missing closing brace nre incomplete",
			in:       `{ .a = 1 && .b !~`,
			expected: "{ (.a = 1) && .b }",
		},

		// Structural operators preserved.
		{
			name:     "structural with incomplete in first",
			in:       `{ .foo = "bar" && .baz = } >> { .bar = "qux" }`,
			expected: "({ (.foo = `bar`) && .baz }) >> ({ .bar = `qux` })",
		},
		{
			name:     "structural with incomplete in second",
			in:       `{ .foo = "bar" } >> { .bar = }`,
			expected: "({ .foo = `bar` }) >> ({ .bar })",
		},
		{
			name:     "structural valid",
			in:       `{ .foo = "bar" } >> { .bar = "baz" }`,
			expected: "({ .foo = `bar` }) >> ({ .bar = `baz` })",
		},
		{
			name:     "chained structural with incomplete",
			in:       `{ .a = } >> { .b = } >> { .c = "foo" }`,
			expected: "(({ .a }) >> ({ .b })) >> ({ .c = `foo` })",
		},
		{
			name:     "mixed structural and spanset ops with incomplete",
			in:       `{ .a = "foo" } && { .b = } || { .c = "bar" }`,
			expected: "(({ .a = `foo` }) && ({ .b })) || ({ .c = `bar` })",
		},

		// Pipelines preserved after cleanup.
		{
			name:     "incomplete with rate pipeline",
			in:       `{ .foo = "bar" && .baz = } | rate() by (.qux)`,
			expected: "{ (.foo = `bar`) && .baz } | rate()by(.qux)",
		},
		{
			name:     "incomplete with select pipeline",
			in:       `{ .foo = && .bar = "baz" } | select(.qux)`,
			expected: "{ .foo && (.bar = `baz`) }|select(.qux)",
		},
		{
			name:     "incomplete with count pipeline",
			in:       `{ .foo = } | count() > 5`,
			expected: "{ .foo }|(count()) > 5",
		},
		{
			name:     "valid query with pipeline",
			in:       `{ .foo = "bar" } | rate()`,
			expected: "{ .foo = `bar` } | rate()",
		},
		{
			name:     "incomplete with multiple pipeline stages",
			in:       `{ .foo = } | count() > 5 | avg(duration) > 1s`,
			expected: "{ .foo }|(count()) > 5|(avg(duration)) > 1s",
		},
		{
			name:     "incomplete with hints",
			in:       `{ .foo = } | rate() with(sample=0.5)`,
			expected: "{ .foo } | rate() with(sample=0.5)",
		},

		// OR conditions (valid, pass through).
		{
			name:     "OR conditions",
			in:       `{ .foo = "bar" || .baz = "qux" }`,
			expected: "{ (.foo = `bar`) || (.baz = `qux`) }",
		},

		// Quoted attributes.
		{
			name:     "quoted attribute name",
			in:       `{ span."foo bar" = "baz" }`,
			expected: "{ span.\"foo bar\" = `baz` }",
		},

		// Regex matchers.
		{
			name:     "regex matcher",
			in:       `{ .foo =~ "(a|b)" }`,
			expected: "{ .foo =~ `(a|b)` }",
		},
		{
			name:     "mixed regex and incomplete",
			in:       `{ .foo =~ "(a|b)" && .bar = }`,
			expected: "{ (.foo =~ `(a|b)`) && .bar }",
		},

		// Reversed operands.
		{
			name:     "reversed operands with incomplete",
			in:       `{ 200 = .status_code && .method = }`,
			expected: "{ (200 = .status_code) && .method }",
		},

		// Arithmetic in matchers.
		{
			name:     "arithmetic expression with incomplete",
			in:       `{ .a + .b = 3 && .c = }`,
			expected: "{ ((.a + .b) = 3) && .c }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ParseLenient(tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual.String())
		})
	}
}

func TestParseLenientErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty string", in: ""},
		{name: "garbage", in: "not a query at all ???"},
		{name: "only operator", in: "&&"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseLenient(tc.in)
			require.Error(t, err)
		})
	}
}

// TestParseLenientKnownFailures documents cases that the lenient parser
// cannot currently recover. These are commented out so they serve as
// documentation of known limitations, not as skipped tests.
func TestParseLenientKnownFailures(t *testing.T) {
	knownFailures := []struct {
		name string
		in   string
	}{
		// Quoted attribute names: the token cleanup doesn't handle quoted
		// identifiers correctly, so the cleaned query fails to re-parse.
		{
			name: "quoted attribute incomplete",
			in:   `{ span."foo bar" = }`,
		},
		{
			name: "unscoped quoted attribute incomplete",
			in:   `{ ."foo bar" = }`,
		},
		{
			name: "resource quoted attribute incomplete",
			in:   `{ resource."service name" = }`,
		},

		// Bare intrinsics (not prefixed with dot) cannot be recovered because
		// after removing the operator, the intrinsic keyword alone doesn't
		// parse as a valid spanset filter expression (e.g. `{ statusMessage }`
		// is not valid TraceQL).
		{
			name: "statusMessage incomplete",
			in:   `{ statusMessage = }`,
		},
		{
			name: "duration incomplete",
			in:   `{ duration = }`,
		},
		{
			name: "name incomplete",
			in:   `{ name = }`,
		},
		{
			name: "kind incomplete",
			in:   `{ kind = }`,
		},
		{
			name: "status incomplete",
			in:   `{ status = }`,
		},

		// Pipe-separated spanset filters: the cleanup only processes tokens
		// before the first pipe, so incomplete matchers in later spanset
		// stages are not cleaned.
		{
			name: "incomplete in second spanset after pipe",
			in:   `{ .foo = "bar" } | { .baz = }`,
		},
	}

	for _, tc := range knownFailures {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseLenient(tc.in)
			if err == nil {
				// If this starts passing, the limitation has been fixed — update the test!
				t.Errorf("known failure %q now passes — move it to TestParseLenient", tc.name)
			}
		})
	}
}

func TestRemoveIncompleteMatchers(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "no incomplete matchers",
			in:       `{ .a = 1 && .b = "two" }`,
			expected: `{ .a = 1 && .b = "two" }`,
		},
		{
			name:     "single incomplete",
			in:       `{ .a = }`,
			expected: `{ .a }`,
		},
		{
			name:     "trailing incomplete",
			in:       `{ .a = 1 && .b = }`,
			expected: `{ .a = 1 && .b }`,
		},
		{
			name:     "leading incomplete",
			in:       `{ .a = && .b = 1 }`,
			expected: `{ .a && .b = 1 }`,
		},
		{
			name:     "all incomplete removes connectors",
			in:       `{ .a = && .b = }`,
			expected: `{ .a && .b }`,
		},
		{
			name:     "missing closing brace auto-closed",
			in:       `{ .a = 1 && .b =`,
			expected: `{ .a = 1 && .b }`,
		},
		{
			name:     "pipeline preserved",
			in:       `{ .a = } | rate ( ) by ( .b )`,
			expected: `{ .a } | rate ( ) by ( .b )`,
		},
		{
			name:     "parens with incomplete keep attribute",
			in:       `{ (.a = ) && .b = 1 }`,
			expected: `{ ( .a ) && .b = 1 }`,
		},
		{
			name:     "empty input",
			in:       "",
			expected: "",
		},
		{
			name:     "scoped attribute",
			in:       `{ resource.service.name = && span.foo = "bar" }`,
			expected: `{ resource.service.name && span.foo = "bar" }`,
		},
		{
			name:     "structural operator passes through",
			in:       `{ .a = } >> { .b = "foo" }`,
			expected: `{ .a } >> { .b = "foo" }`,
		},
		{
			name:     "OR connector preserved",
			in:       `{ .a = "foo" || .b = }`,
			expected: `{ .a = "foo" || .b }`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := removeIncompleteMatchers(tc.in)
			require.Equal(t, tc.expected, actual)
		})
	}
}
