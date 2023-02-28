package traceql

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpansetFilter_matches(t *testing.T) {
	tests := []struct {
		query   string
		span    Span
		matches bool
		// TODO do we actually care about the error mesasge?
		err bool
	}{
		{
			query: `{ ("foo" != "bar") && !("foo" = "bar") }`,
			span: Span{
				Attributes: nil,
			},
			matches: true,
		},
		{
			query: `{ .foo = .bar }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticString("bzz"),
					NewAttribute("bar"): NewStaticString("bzz"),
				},
			},
			matches: true,
		},
		{
			// Missing attribute
			query: `{ .foo = "bar" }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("fzz"): NewStaticString("bar"),
				},
			},
			matches: false,
		},
		{
			query: `{ .foo = .bar }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticString("str"),
					NewAttribute("bar"): NewStaticInt(5),
				},
			},
			matches: false,
		},
		{
			// Types don't match with operator
			query: `{ .foo =~ .bar }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticInt(3),
					NewAttribute("bar"): NewStaticInt(5),
				},
			},
			matches: false,
		},
		{
			query: `{ .field1 =~ "hello w.*" && .field2 !~ "bye b.*" }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("field1"): NewStaticString("hello world"),
					NewAttribute("field2"): NewStaticString("bye world"),
				},
			},
			matches: true,
		},
		{
			query: `{ .foo > 2 && .foo >= 3.5 && .foo < 5 && .foo <= 3.5 && .duration > 1800ms }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("foo"):      NewStaticFloat(3.5),
					NewAttribute("duration"): NewStaticDuration(2 * time.Second),
				},
			},
			matches: true,
		},
		{
			query: `{ .foo = "scope_span" }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewScopedAttribute(AttributeScopeSpan, false, "foo"):     NewStaticString("scope_span"),
					NewScopedAttribute(AttributeScopeResource, false, "foo"): NewStaticString("scope_resource"),
				},
			},
			matches: true,
		},
		{
			query: `{ .foo = "scope_resource" }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewScopedAttribute(AttributeScopeResource, false, "foo"): NewStaticString("scope_resource"),
				},
			},
			matches: true,
		},
		{
			query: `{ span.foo = "scope_span" }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewScopedAttribute(AttributeScopeSpan, false, "foo"):     NewStaticString("scope_span"),
					NewScopedAttribute(AttributeScopeResource, false, "foo"): NewStaticString("scope_resource"),
				},
			},
			matches: true,
		},
		{
			query: `{ resource.foo = "scope_resource" }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewScopedAttribute(AttributeScopeSpan, false, "foo"):     NewStaticString("scope_span"),
					NewScopedAttribute(AttributeScopeResource, false, "foo"): NewStaticString("scope_resource"),
				},
			},
			matches: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			spansetFilter := expr.Pipeline.Elements[0].(SpansetFilter)

			matches, err := spansetFilter.matches(tt.span)

			if tt.err {
				fmt.Println(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.matches, matches)
			}
		})
	}

}

func TestSpansetOperationEvaluate(t *testing.T) {
	testCases := []struct {
		query  string
		input  []Spanset
		output []Spanset
	}{
		{
			"{ .foo = `a` } && { .foo = `b` }",
			[]Spanset{
				{Spans: []Span{
					// This spanset will be kept because it satisfies both conditions
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
				{Spans: []Span{
					// This spanset will be dropped
					{ID: []byte{3}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
		},
		{
			"{ .foo = `a` } || { .foo = `b` }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
				{Spans: []Span{
					// Second span will be dropped
					{ID: []byte{3}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
					{ID: []byte{4}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("c")}},
				}},
			},
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
				{Spans: []Span{
					{ID: []byte{3}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			ast, err := Parse(tc.query)
			require.NoError(t, err)

			filt := ast.Pipeline.Elements[0].(SpansetOperation)

			actual, err := filt.evaluate(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.output, actual)
		})
	}
}

func TestScalarFilterEvaluate(t *testing.T) {
	testCases := []struct {
		query  string
		input  []Spanset
		output []Spanset
	}{
		{
			"{ .foo = `a` } | count() > 1",
			[]Spanset{
				{Spans: []Span{
					// This has 1 match
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
				{Spans: []Span{
					// This has 2 matches
					{ID: []byte{3}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					{ID: []byte{4}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
			},
			[]Spanset{
				{
					Scalar: NewStaticInt(2),
					Spans: []Span{
						{ID: []byte{3}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
						{ID: []byte{4}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					},
				},
			},
		},
		{
			"{ .foo = `a` } | avg(duration) >= 10ms",
			[]Spanset{
				{Spans: []Span{
					// Avg duration = 5ms
					{ID: []byte{1}, Attributes: map[Attribute]Static{
						NewAttribute("foo"):             NewStaticString("a"),
						NewIntrinsic(IntrinsicDuration): NewStaticDuration(2 * time.Millisecond)},
					},
					{ID: []byte{2}, Attributes: map[Attribute]Static{
						NewAttribute("foo"):             NewStaticString("a"),
						NewIntrinsic(IntrinsicDuration): NewStaticDuration(8 * time.Millisecond)},
					},
				}},
				{Spans: []Span{
					// Avg duration = 10ms
					{ID: []byte{3}, Attributes: map[Attribute]Static{
						NewAttribute("foo"):             NewStaticString("a"),
						NewIntrinsic(IntrinsicDuration): NewStaticDuration(5 * time.Millisecond)},
					},
					{ID: []byte{4}, Attributes: map[Attribute]Static{
						NewAttribute("foo"):             NewStaticString("a"),
						NewIntrinsic(IntrinsicDuration): NewStaticDuration(15 * time.Millisecond)},
					},
				}},
			},
			[]Spanset{
				{
					// TODO - Type handling of aggregate output could use some improvement.
					// avg(duration) should probably return a Duration instead of a float.
					Scalar: NewStaticFloat(10.0 * float64(time.Millisecond)),
					Spans: []Span{
						{ID: []byte{3}, Attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(5 * time.Millisecond)},
						},
						{ID: []byte{4}, Attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(15 * time.Millisecond)},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			ast, err := Parse(tc.query)
			require.NoError(t, err)

			actual, err := ast.Pipeline.evaluate(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.output, actual)
		})
	}
}

func TestBinaryOperationWorksWithFloatsAndInts(t *testing.T) {
	testCases := []struct {
		query  string
		input  []Spanset
		output []Spanset
	}{
		// binops work int -> float
		{
			"{ .foo > 0 }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		{
			"{ .foo < 2 }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		{
			"{ .foo = 1 }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		// binops work float -> int
		{
			"{ .foo > 0. }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		{
			"{ .foo < 2. }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		{
			"{ .foo = 1. }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		// binops work with statics
		{
			"{ 1 > 0. }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		{
			"{ 0 < 2. }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		{
			"{ 1 = 1. }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{
					Spans: []Span{
						{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
						{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
					},
				},
			},
		},
		// binops work with attributes
		{
			"{ .foo < .bar }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(2)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(2), NewAttribute("bar"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(2)}},
				}},
			},
		},
		{
			"{ .bar > .foo }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(2)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(2), NewAttribute("bar"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(2)}},
				}},
			},
		},
		{
			"{ .foo = .bar }",
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}},
					{ID: []byte{2}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(2), NewAttribute("bar"): NewStaticFloat(1)}},
				}},
			},
			[]Spanset{
				{Spans: []Span{
					{ID: []byte{1}, Attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}},
				}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			ast, err := Parse(tc.query)
			require.NoError(t, err)

			actual, err := ast.Pipeline.evaluate(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.output, actual)
		})
	}
}
