package traceql

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type evalTC struct {
	query  string
	input  []*Spanset
	output []*Spanset
}

func testEvaluator(t *testing.T, tc evalTC) {
	t.Helper()

	t.Run(tc.query, func(t *testing.T) {
		ast, err := Parse(tc.query)
		require.NoError(t, err)

		// clone input to confirm it doesn't get modified
		cloneIn := make([]*Spanset, len(tc.input))
		for i := range tc.input {
			cloneIn[i] = tc.input[i].clone()
			cloneIn[i].Spans = append([]Span(nil), tc.input[i].Spans...)
		}

		actual, err := ast.Pipeline.evaluate(tc.input)
		require.NoError(t, err)

		// sort expected/actual spansets. grouping requires this b/c map iteration makes the output
		// non-deterministic.
		makeSort := func(ss []*Spanset) func(i, j int) bool {
			return func(i, j int) bool {
				return bytes.Compare(ss[i].Spans[0].ID(), ss[j].Spans[0].ID()) < 0
			}
		}
		sort.Slice(actual, makeSort(actual))
		sort.Slice(tc.output, makeSort(tc.output))

		require.Equal(t, tc.output, actual)
		require.Equal(t, tc.input, cloneIn)
	})
}

func TestSpansetFilter_matches(t *testing.T) {
	tests := []struct {
		query   string
		span    Span
		matches bool
	}{
		{
			query: `{ ("foo" != "bar") && !("foo" = "bar") }`,
			span: &mockSpan{
				attributes: nil,
			},
			matches: true,
		},
		{
			query: `{ .foo = .bar }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticString("bzz"),
					NewAttribute("bar"): NewStaticString("bzz"),
				},
			},
			matches: true,
		},
		{
			// Missing attribute
			query: `{ .foo = "bar" }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("fzz"): NewStaticString("bar"),
				},
			},
			matches: false,
		},
		{
			query: `{ .foo = .bar }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticString("str"),
					NewAttribute("bar"): NewStaticInt(5),
				},
			},
			matches: false,
		},
		{
			// Types don't match with operator
			query: `{ .foo =~ .bar }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticInt(3),
					NewAttribute("bar"): NewStaticInt(5),
				},
			},
			matches: false,
		},
		{
			query: `{ .field1 =~ "hello w.*" && .field2 !~ "bye b.*" }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("field1"): NewStaticString("hello world"),
					NewAttribute("field2"): NewStaticString("bye world"),
				},
			},
			matches: true,
		},
		{
			query: `{ .foo > 2 && .foo >= 3.5 && .foo < 5 && .foo <= 3.5 && .duration > 1800ms }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("foo"):      NewStaticFloat(3.5),
					NewAttribute("duration"): NewStaticDuration(2 * time.Second),
				},
			},
			matches: true,
		},
		{
			query: `{ .foo = "scope_span" }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewScopedAttribute(AttributeScopeSpan, false, "foo"):     NewStaticString("scope_span"),
					NewScopedAttribute(AttributeScopeResource, false, "foo"): NewStaticString("scope_resource"),
				},
			},
			matches: true,
		},
		{
			query: `{ .foo = "scope_resource" }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewScopedAttribute(AttributeScopeResource, false, "foo"): NewStaticString("scope_resource"),
				},
			},
			matches: true,
		},
		{
			query: `{ span.foo = "scope_span" }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewScopedAttribute(AttributeScopeSpan, false, "foo"):     NewStaticString("scope_span"),
					NewScopedAttribute(AttributeScopeResource, false, "foo"): NewStaticString("scope_resource"),
				},
			},
			matches: true,
		},
		{
			query: `{ resource.foo = "scope_resource" }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewScopedAttribute(AttributeScopeSpan, false, "foo"):     NewStaticString("scope_span"),
					NewScopedAttribute(AttributeScopeResource, false, "foo"): NewStaticString("scope_resource"),
				},
			},
			matches: true,
		},
	}
	for _, tt := range tests {
		// create a evalTC and use testEvaluator
		tc := evalTC{
			query: tt.query,
			input: []*Spanset{
				{Spans: []Span{tt.span}},
			},
			output: []*Spanset{},
		}
		if tt.matches {
			tc.output = tc.input
		}
		testEvaluator(t, tc)
	}
}

func TestGroup(t *testing.T) {
	testCases := []evalTC{
		{
			"{ } | by(.foo)",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{
					Spans: []Span{
						&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					},
					Attributes: []*SpansetAttribute{{Name: "by(.foo)", Val: NewStaticString("a")}},
				},
				{
					Spans: []Span{
						&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
						&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
					},
					Attributes: []*SpansetAttribute{{Name: "by(.foo)", Val: NewStaticString("b")}},
				},
			},
		},
		{
			"{ } | by(.foo) | by(.bar)",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a"), NewAttribute("bar"): NewStaticString("1")}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b"), NewAttribute("bar"): NewStaticString("1")}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b"), NewAttribute("bar"): NewStaticString("2")}},
				}},
			},
			[]*Spanset{
				{
					Spans: []Span{
						&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a"), NewAttribute("bar"): NewStaticString("1")}},
					},
					Attributes: []*SpansetAttribute{{Name: "by(.foo)", Val: NewStaticString("a")}, {Name: "by(.bar)", Val: NewStaticString("1")}},
				},
				{
					Spans: []Span{
						&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b"), NewAttribute("bar"): NewStaticString("1")}},
					},
					Attributes: []*SpansetAttribute{{Name: "by(.foo)", Val: NewStaticString("b")}, {Name: "by(.bar)", Val: NewStaticString("1")}},
				},
				{
					Spans: []Span{
						&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b"), NewAttribute("bar"): NewStaticString("2")}},
					},
					Attributes: []*SpansetAttribute{{Name: "by(.foo)", Val: NewStaticString("b")}, {Name: "by(.bar)", Val: NewStaticString("2")}},
				},
			},
		},
	}

	for _, tc := range testCases {
		testEvaluator(t, tc)
	}
}

func TestCoalesce(t *testing.T) {
	testCases := []evalTC{
		{
			"{ } | coalesce()",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
				{
					Spans: []Span{
						&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
						&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
					},
					// coalesce() should drop attributes
					Attributes: []*SpansetAttribute{{Name: "by(.foo)", Val: NewStaticString("a")}},
				},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
		},
	}

	for _, tc := range testCases {
		testEvaluator(t, tc)
	}
}

func TestSpansetOperationEvaluate(t *testing.T) {
	testCases := []evalTC{
		{
			"{ .foo = `a` } && { .foo = `b` }",
			[]*Spanset{
				{Spans: []Span{
					// This spanset will be kept because it satisfies both conditions
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
				{Spans: []Span{
					// This spanset will be dropped
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
			},
		},
		{
			"{ .foo = `a` } || { .foo = `b` }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
				{Spans: []Span{
					// Second span will be dropped
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
					&mockSpan{id: []byte{4}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("c")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
				{Spans: []Span{
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
		},
		{
			"{ true } && { true } && { true }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
			},
		},
		{
			"{ true } || { true } || { true }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
			},
		},
		{
			"{ .parent } >> { .child }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
					newMockSpan([]byte{1}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
		},
		{
			"{ .child } << { .parent }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
					newMockSpan([]byte{1}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
				}},
			},
		},
		{
			"{ .parent } > { .child }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
					newMockSpan([]byte{1}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
		},
		{
			"{ .child } < { .parent }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
					newMockSpan([]byte{1}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
				}},
			},
		},
		{
			"{ .child1 } ~ { .child2 }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("child1", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{1}).WithAttrBool("child2", true).WithNestedSetInfo(1, 4, 5),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("child2", true).WithNestedSetInfo(1, 4, 5),
				}},
			},
		},
		{
			"{ } !< { .child }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
					newMockSpan([]byte{2}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{2}).WithAttrBool("child", true).WithNestedSetInfo(0, 1, 4),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{2}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
		},
		{
			"{ } !> { .parent }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{2}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
				}},
			},
		},
		{
			"{ .child1 } !~ { .child2 }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("child1", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{1}).WithAttrBool("child2", true).WithNestedSetInfo(1, 4, 5),
					newMockSpan([]byte{1}).WithAttrBool("child2", true).WithNestedSetInfo(4, 5, 6),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("child2", true).WithNestedSetInfo(4, 5, 6),
				}},
			},
		},
		{
			"{ } !<< { .child }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
					newMockSpan([]byte{2}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{2}).WithAttrBool("child", true).WithNestedSetInfo(0, 1, 4),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{2}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
		},
		{
			"{ } !>> { .parent }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{2}).WithAttrBool("child", true).WithNestedSetInfo(1, 2, 3),
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent", true).WithNestedSetInfo(0, 1, 4),
				}},
			},
		},
		{ // tests that child operators do not modify the spanset
			"{ } > { } > { } > { }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("child1", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{1}).WithAttrBool("child2", true).WithNestedSetInfo(1, 4, 5),
				}},
			},
			[]*Spanset{},
		},
		{ // tests that parent operators do not modify the spanset
			"{ } < { } < { } < { }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent1", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{1}).WithAttrBool("parent2", true).WithNestedSetInfo(1, 4, 5),
				}},
			},
			[]*Spanset{},
		},
		{ // tests that parent operators do not modify the spanset
			"{ } &< { } &< { } &< { }",
			[]*Spanset{
				{Spans: []Span{
					newMockSpan([]byte{1}).WithAttrBool("parent1", true).WithNestedSetInfo(1, 2, 3),
					newMockSpan([]byte{1}).WithAttrBool("parent2", true).WithNestedSetInfo(1, 4, 5),
				}},
			},
			[]*Spanset{},
		},
	}

	for _, tc := range testCases {
		testEvaluator(t, tc)
	}
}

// test cases for array
func TestSpansetOperationEvaluateArray(t *testing.T) {
	testCases := []evalTC{
		// string arrays
		{
			"{ .foo = `bar` }", // match string array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
				}},
			},
		},
		{
			"{ .foo = `bar` || .bat = `baz` }", // match string array with or
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
				}},
			},
		},
		{
			"{ .foo != `baz` }", // match string array not equal
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"baz"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
				}},
			},
		},
		{
			"{ .foo =~ `ba` }", // match string array with regex
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"dog", "cat"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
				}},
			},
		},
		{
			"{ .foo !~ `ba` }", // regex non-matching
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"cat"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"cat"})}},
				}},
			},
		},
		// int arrays
		{
			"{ .foo = 2 }", // match int array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
		},
		{
			"{ .foo != 3 }", // match int array not equal
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
					// this is filtered out as expected??
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 3})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
				}},
			},
		},
		{
			"{ .foo = 2.5 }", // match float array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 2.5})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 2.5})}},
				}},
			},
		},
		{
			"{ .foo = 3.14 }", // match another float array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{3.14, 6.28})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.23, 4.56})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{3.14, 6.28})}},
				}},
			},
		},
		{
			"{ .foo > 1 }", // match int array greater than
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{0, 1})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
		},
		{
			"{ .foo >= 1 }", // match int array greater equal than
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{0, 1})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{0, -1})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{0, 1})}},
				}},
			},
		},
		{
			"{ .foo < 2 }", // match int array less than
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 3})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 3})}},
				}},
			},
		},
		{
			"{ .foo <= 2 }", // match int array less than equal
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 3})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{2, 4})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 3})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{2, 4})}},
				}},
			},
		},
		// match float arrays
		{
			"{ .foo != 2.5 }", // match float array not equal
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.5, 4.0})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.5})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.5, 4.0})}},
				}},
			},
		},
		{
			"{ .foo < 2.0 }", // match float array less than
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 4.0})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
				}},
			},
		},
		{
			"{ .foo <= 2.0 }", // match float array less than
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 4.0})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{3.0, 4.0})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 4.0})}},
				}},
			},
		},
		{
			"{ .foo > 2.5 }", // match float array greater than
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 3.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.0, 2.0})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 3.0})}},
				}},
			},
		},
		{
			"{ .foo >= 2.5 }", // match float array greater than
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 3.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.0, 2.0})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.0, 2.5})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 3.0})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.0, 2.5})}},
				}},
			},
		},
		// match bool arrays
		{
			"{ .foo = true }", // match boolean array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
				}},
			},
		},
		{
			"{ .foo = false }", // match another boolean array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
				}},
			},
		},
		{
			"{ .foo != true }", // match boolean array not equal to true
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
				}},
			},
		},
		{
			"{ .foo = !true }", // match boolean array not equal to true
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
				}},
			},
		},
		{
			"{ .foo = true && .foo = false }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
				}},
			},
		},
		{
			"{ .foo = true || .foo = false }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
				}},
			},
		},
		// empty arrays
		{
			"{ .foo = 1 }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{})}},
				}},
			},
			[]*Spanset{},
		},
		{
			"{ .foo = `test` }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{})}},
				}},
			},
			[]*Spanset{},
		},
	}
	for _, tc := range testCases {
		testEvaluator(t, tc)
	}
}

// tests to make sure symmetric operations are supported for arrays...
func TestSpansetOperationEvaluateArraySymmetric(t *testing.T) {
	testCases := []evalTC{
		// string arrays
		{
			"{ `bar` = .foo }", // Symmetric match for string array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
				}},
			},
		},
		{
			"{ `baz` != .foo }", // Symmetric not equal for string array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"baz"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
				}},
			},
		},
		{
			"{ `ba` =~ .foo }", // Symmetric regex match for string array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					// &mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
					// &mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"cat", "dog"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					// &mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
				}},
			},
		},
		{
			"{ `ba` !~ .foo }", // Symmetric regex non-match for string array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
				}},
			},
		},
		// int arrays
		{
			"{ 2 = .foo }", // Symmetric match for int array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
		},
		{
			"{ 3 != .foo }", // Symmetric not equal for int array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
				}},
			},
		},
		{
			"{ 2 < .foo }", // Symmetric less-than for int array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{0, 1})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					// spans with any array elements grater then 2 will not be filtered out.
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 2})}},
				}},
			},
		},
		{
			"{ 2 <= .foo }", // Symmetric less-than equal for int array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{0, 1})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
		},
		{
			"{ 3 > .foo }", // Symmetric grater-than for int array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 3})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 3})}},
				}},
			},
		},
		{
			"{ 3 >= .foo }", // Symmetric grater-than-equal for int array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 3})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{4}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{4, 6})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 3})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
		},
		// match float arrays
		{
			"{ 2.5 = .foo }", // Symmetric match for float array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 2.5})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.5})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 2.5})}},
				}},
			},
		},
		{
			"{ 3.14 != .foo }", // Symmetric not equal for float array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.23, 4.56})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{3.14, 6.28})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{3.14})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.23, 4.56})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{3.14, 6.28})}},
				}},
			},
		},
		{
			"{ 2.0 > .foo }", // Symmetric grater-than for float array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 4.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
				}},
			},
		},
		{
			"{ 2.0 >= .foo }", // Symmetric grater-than-equal for float array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 4.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{3.5, 3.0})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 4.0})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{1.5, 3.0})}},
				}},
			},
		},
		{
			"{ 3.5 < .foo }", // Symmetric less-than for float array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 3.5})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{4.0, 3.5})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{4.0, 5.5})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{4.0, 3.5})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{4.0, 5.5})}},
				}},
			},
		},
		{
			"{ 3.5 <= .foo }", // Symmetric less-than-equal for float array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 3.5})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{4.0, 3.5})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{4.0, 5.5})}},
					&mockSpan{id: []byte{4}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 1.5})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{2.0, 3.5})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{4.0, 3.5})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloatArray([]float64{4.0, 5.5})}},
				}},
			},
		},
		// match bool arrays
		{
			"{ true = .foo }", // Symmetric match for boolean array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
				}},
			},
		},
		{
			"{ false != .foo }", // Symmetric not equal for boolean array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
		},
		{
			"{ !false = .foo }", // Symmetric not equal for boolean array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
		},
		{
			"{ true = .foo && false = .foo }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
				}},
			},
		},
		{
			"{ true = .foo || false = .foo }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
				}},
			},
		},
	}
	for _, tc := range testCases {
		testEvaluator(t, tc)
	}
}

func TestSpansetOperationEvaluateArrayUnsupported(t *testing.T) {
	testCases := []evalTC{
		{
			"{ .foo + 3 = 4 }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
			[]*Spanset{},
		},
		{
			"{ 4 = .foo + 3 }",
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
			[]*Spanset{},
		},
	}
	for _, tc := range testCases {
		t.Helper()

		t.Run(tc.query, func(t *testing.T) {
			ast, err := Parse(tc.query)
			require.NoError(t, err)

			// clone input to confirm it doesn't get modified
			cloneIn := make([]*Spanset, len(tc.input))
			for i := range tc.input {
				cloneIn[i] = tc.input[i].clone()
				cloneIn[i].Spans = append([]Span(nil), tc.input[i].Spans...)
			}

			_, err = ast.Pipeline.evaluate(tc.input)
			require.Error(t, err, errors.ErrUnsupported)
		})
	}
}

func TestScalarFilterEvaluate(t *testing.T) {
	testCases := []evalTC{
		{
			"{ .foo = `a` } | count() > 1",
			[]*Spanset{
				{Spans: []Span{
					// This has 1 match
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
				{Spans: []Span{
					// This has 2 matches
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
			},
			[]*Spanset{
				{
					Scalar: NewStaticInt(2),
					Spans: []Span{
						&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
						&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					},
					Attributes: []*SpansetAttribute{{Name: "count()", Val: NewStaticInt(2)}},
				},
			},
		},
		{
			"{ .foo = `a` } | avg(duration) >= 10ms",
			[]*Spanset{
				{Spans: []Span{
					// Avg duration = 5ms
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(2 * time.Millisecond),
						},
					},
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(8 * time.Millisecond),
						},
					},
				}},
				{Spans: []Span{
					// Avg duration = 10ms
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(5 * time.Millisecond),
						},
					},
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(15 * time.Millisecond),
						},
					},
				}},
			},
			[]*Spanset{
				{
					Scalar: NewStaticDuration(10.0 * time.Millisecond),
					Spans: []Span{
						&mockSpan{
							attributes: map[Attribute]Static{
								NewAttribute("foo"):             NewStaticString("a"),
								NewIntrinsic(IntrinsicDuration): NewStaticDuration(5 * time.Millisecond),
							},
						},
						&mockSpan{
							attributes: map[Attribute]Static{
								NewAttribute("foo"):             NewStaticString("a"),
								NewIntrinsic(IntrinsicDuration): NewStaticDuration(15 * time.Millisecond),
							},
						},
					},
					Attributes: []*SpansetAttribute{{Name: "avg(duration)", Val: NewStaticDuration(10 * time.Millisecond)}},
				},
			},
		},
		// max
		{
			"{ .foo = `a` } | max(duration) >= 10ms",
			[]*Spanset{
				{Spans: []Span{
					// max duration = 8ms
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(2 * time.Millisecond),
						},
					},
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(8 * time.Millisecond),
						},
					},
				}},
				{Spans: []Span{
					// max duration = 15ms
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(5 * time.Millisecond),
						},
					},
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(15 * time.Millisecond),
						},
					},
				}},
			},
			[]*Spanset{
				{
					Scalar: NewStaticDuration(15 * time.Millisecond),
					Spans: []Span{
						&mockSpan{
							attributes: map[Attribute]Static{
								NewAttribute("foo"):             NewStaticString("a"),
								NewIntrinsic(IntrinsicDuration): NewStaticDuration(5 * time.Millisecond),
							},
						},
						&mockSpan{
							attributes: map[Attribute]Static{
								NewAttribute("foo"):             NewStaticString("a"),
								NewIntrinsic(IntrinsicDuration): NewStaticDuration(15 * time.Millisecond),
							},
						},
					},
					Attributes: []*SpansetAttribute{{Name: "max(duration)", Val: NewStaticDuration(15 * time.Millisecond)}},
				},
			},
		},
		// min
		{
			"{ .foo = `a` } | min(duration) <= 10ms",
			[]*Spanset{
				{Spans: []Span{
					// min duration = 2ms
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(2 * time.Millisecond),
						},
					},
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(8 * time.Millisecond),
						},
					},
				}},
				{Spans: []Span{
					// min duration = 5ms
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(12 * time.Millisecond),
						},
					},
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(15 * time.Millisecond),
						},
					},
				}},
			},
			[]*Spanset{
				{
					Scalar: NewStaticDuration(2 * time.Millisecond),
					Spans: []Span{
						&mockSpan{
							attributes: map[Attribute]Static{
								NewAttribute("foo"):             NewStaticString("a"),
								NewIntrinsic(IntrinsicDuration): NewStaticDuration(2 * time.Millisecond),
							},
						},
						&mockSpan{
							attributes: map[Attribute]Static{
								NewAttribute("foo"):             NewStaticString("a"),
								NewIntrinsic(IntrinsicDuration): NewStaticDuration(8 * time.Millisecond),
							},
						},
					},
					Attributes: []*SpansetAttribute{{Name: "min(duration)", Val: NewStaticDuration(2 * time.Millisecond)}},
				},
			},
		},
		// sum
		{
			"{ .foo = `a` } | sum(duration) = 10ms",
			[]*Spanset{
				{Spans: []Span{
					// sum duration = 10ms
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(2 * time.Millisecond),
						},
					},
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(8 * time.Millisecond),
						},
					},
				}},
				{Spans: []Span{
					// sum duration = 27ms
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(12 * time.Millisecond),
						},
					},
					&mockSpan{
						attributes: map[Attribute]Static{
							NewAttribute("foo"):             NewStaticString("a"),
							NewIntrinsic(IntrinsicDuration): NewStaticDuration(15 * time.Millisecond),
						},
					},
				}},
			},
			[]*Spanset{
				{
					Scalar: NewStaticDuration(10 * time.Millisecond),
					Spans: []Span{
						&mockSpan{
							attributes: map[Attribute]Static{
								NewAttribute("foo"):             NewStaticString("a"),
								NewIntrinsic(IntrinsicDuration): NewStaticDuration(2 * time.Millisecond),
							},
						},
						&mockSpan{
							attributes: map[Attribute]Static{
								NewAttribute("foo"):             NewStaticString("a"),
								NewIntrinsic(IntrinsicDuration): NewStaticDuration(8 * time.Millisecond),
							},
						},
					},
					Attributes: []*SpansetAttribute{{Name: "sum(duration)", Val: NewStaticDuration(10 * time.Millisecond)}},
				},
			},
		},
	}

	for _, tc := range testCases {
		testEvaluator(t, tc)
	}
}

func TestBinaryOperationsWorkAcrossNumberTypes(t *testing.T) {
	testCases := []evalTC{
		{
			"{ .foo > 0 }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		{
			"{ .foo < 2 }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		{
			"{ .foo = 1 }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		{
			"{ .foo > 0. }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		{
			"{ .foo < 2. }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		{
			"{ .foo = 1. }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		{
			"{ .foo > 0ns }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		{
			"{ .foo < 2ns }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		{
			"{ .foo = 1ns }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticDuration(1)}},
			}}},
		},
		// binops work with statics
		{
			"{ 1 > 0. }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
		},
		{
			"{ 0 < 2. }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
		},
		{
			"{ 1 = 1. }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
		},
		{
			"{ 1ms = 1ms }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
		},
		{
			"{ 1ns = 1 }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticFloat(1)}},
			}}},
		},
		// binops work with attributes
		{
			"{ .foo < .bar }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(2)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(2), NewAttribute("bar"): NewStaticFloat(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(2)}},
			}}},
		},
		{
			"{ .bar > .foo }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(2)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(2), NewAttribute("bar"): NewStaticFloat(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(2)}},
			}}},
		},
		{
			"{ .foo = .bar }",
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}},
				&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(2), NewAttribute("bar"): NewStaticFloat(1)}},
			}}},
			[]*Spanset{{Spans: []Span{
				&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}},
			}}},
		},
	}

	for _, tc := range testCases {
		testEvaluator(t, tc)
	}
}

func TestArithmetic(t *testing.T) {
	testCases := []evalTC{
		// static arithmetic works
		{
			"{ 1 + 1 = 2 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ 2 - 2 > -1 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ 1 / 10. = .1 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ 1 / 10 = 0 }", // integer division
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ 3 * 2 = 6 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ 10 % 3 = 1 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ 2 ^ 2 = 4 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ 2m + 2m = 4m }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ 2m * 2 = 4m }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		// attribute arithmetic works
		{
			"{ .foo + .bar = 2 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ .foo - 2 = -1 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ .foo / .bar != 3 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ .foo * .bar = 1 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ .foo % .bar = 0 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ .foo ^ .bar < 3 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		{
			"{ .foo * 3ms = 3ms }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1), NewAttribute("bar"): NewStaticFloat(1)}}}}},
		},
		// complex true
		{
			"{ (2 - .bar) * .foo = -15}",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
		},
		{
			"{ 2 - .bar * .foo = -19}",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
		},
		{
			"{ 2 ^ (.bar * .foo) = 2097152}",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
		},
		{
			"{ .bar % 2 = .foo - 2 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
		},
		// complex false
		{
			"{ (2 - .bar) * .foo < -15}",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
			[]*Spanset{},
		},
		{
			"{ 2 - .bar * .foo > -19}",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
			[]*Spanset{},
		},
		{
			"{ 2 ^ (.bar * .foo) != 2097152}",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
			[]*Spanset{},
		},
		{
			"{ .bar % 2 < .foo - 2 }",
			[]*Spanset{{Spans: []Span{&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3), NewAttribute("bar"): NewStaticFloat(7)}}}}},
			[]*Spanset{},
		},
	}

	for _, tc := range testCases {
		testEvaluator(t, tc)
	}
}

func TestSpansetExistence(t *testing.T) {
	tests := []struct {
		query   string
		span    Span
		matches bool
	}{
		// return traces where .foo exists
		{
			query: `{ .foo != nil }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("bar"): NewStaticString("bzz"),
				},
			},
			matches: false,
		},
		{
			query: `{ .bar != nil }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("bar"): NewStaticString("bzz"),
				},
			},
			matches: true,
		},
		{
			query: `{ .int != nil }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("int"): NewStaticInt(1),
				},
			},
			matches: true,
		},
		{
			query: `{ .duration != nil }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("duration"): NewStaticDuration(time.Minute),
				},
			},
			matches: true,
		},
		{
			query: `{ .float != nil }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("float"): NewStaticFloat(2.0),
				},
			},
			matches: true,
		},
	}
	for _, tt := range tests {
		// create a evalTC and use testEvaluator
		tc := evalTC{
			query: tt.query,
			input: []*Spanset{
				{Spans: []Span{tt.span}},
			},
			output: []*Spanset{},
		}
		if tt.matches {
			tc.output = tc.input
		}
		testEvaluator(t, tc)
	}
}

func BenchmarkBinOp(b *testing.B) {
	ops := []struct {
		op BinaryOperation
	}{
		{
			op: BinaryOperation{
				Op:  OpEqual,
				LHS: NewStaticInt(1),
				RHS: NewStaticInt(1),
			},
		},
		{
			op: BinaryOperation{
				Op:  OpEqual,
				LHS: NewStaticFloat(1),
				RHS: NewStaticInt(1),
			},
		},
		{
			op: BinaryOperation{
				Op:  OpEqual,
				LHS: NewStaticDuration(1),
				RHS: NewStaticFloat(1),
			},
		},
		{
			op: BinaryOperation{
				Op:  OpEqual,
				LHS: NewStaticFloat(1),
				RHS: NewStaticFloat(1),
			},
		},
	}

	for _, o := range ops {
		b.Run(o.op.String(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = o.op.execute(&mockSpan{})
			}
		})
	}
}

// BenchmarkUniquespans benchmarks the performance of the uniqueSpans function using
// different numbers of spansets and spans.
func BenchmarkUniqueSpans(b *testing.B) {
	sizes := []int{1, 10, 100, 1000, 10000}

	for _, lhs := range sizes {
		for i := len(sizes) - 1; i >= 0; i-- {
			rhs := sizes[i]
			b.Run(fmt.Sprintf("%d|%d", rhs, lhs), func(b *testing.B) {
				lhsSpansets := []*Spanset{{Spans: make([]Span, lhs)}}
				rhsSpansets := []*Spanset{{Spans: make([]Span, rhs)}}
				for j := 0; j < lhs; j++ {
					lhsSpansets[0].Spans[j] = &mockSpan{id: []byte{byte(j)}}
				}
				for j := 0; j < rhs; j++ {
					rhsSpansets[0].Spans[j] = &mockSpan{id: []byte{byte(j)}}
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					uniqueSpans(lhsSpansets, rhsSpansets)
				}
			})
		}
	}
}

func BenchmarkAggregate(b *testing.B) {
	agg := newAggregate(aggregateAvg, NewStaticInt(3))
	ss := make([]*Spanset, 1)
	ss[0] = &Spanset{
		Spans: make([]Span, 1000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agg.evaluate(ss)
	}
}
