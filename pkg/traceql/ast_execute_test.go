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
			"{ .foo = `bar` }", // string in array
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
			"{ .foo = `bar` || .bat = `baz` }", // string in array with or
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
			"{ .foo != `baz` }", // string not in array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"baz"})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "one"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "one"})}},
				}},
			},
		},
		{
			"{ .foo =~ `ba.*` }", // string match any in array
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
			"{ .foo !~ `ba.*` }", // string match none in array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"foo", "baz"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "baz"})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"cat"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"cat"})}},
				}},
			},
		},
		// int arrays
		{
			"{ .foo = 2 }", // int in array
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
			"{ .foo != 3 }", // int not in array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 4})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{3, 3})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticIntArray([]int{1, 2})}},
				}},
			},
		},
		{
			"{ .foo = 2.5 }", // float in array
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
			"{ .foo = 3.14 }", // float in array array
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
			"{ .foo > 1 }", // float > 1 in aray
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
			"{ .foo >= 1 }", // float >= 1 in array
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
			"{ .foo < 2 }", // float < 2 in array
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
			"{ .foo <= 2 }", // float <= 2 in array
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
			"{ .foo != 2.5 }", // float not in array
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
				}},
			},
		},
		{
			"{ .foo < 2.0 }", // float < 2.0 in array
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
			"{ .foo <= 2.0 }", // float <= 2.0 in array
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
			"{ .foo > 2.5 }", // float > 2.5 in array
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
			"{ .foo >= 2.5 }", // float >= 2.5 in array
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
			"{ .foo = true }", // boolean in array
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
			"{ .foo = false }", // boolean in array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
				}},
			},
		},
		{
			"{ .foo != true }", // boolean not in array
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
				}},
			},
		},
		{
			"{ .foo = !true }", // negated boolean in array
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
			"{ `bar` = .foo }", // symmetric: string in array
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
			"{ `baz` != .foo }", // symmetric: string not in array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "foo"})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"baz"})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticStringArray([]string{"bar", "foo"})}},
				}},
			},
		},
		// int arrays
		{
			"{ 2 = .foo }", // symmetric: int not in array
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
			"{ 3 != .foo }", // symmetric: in not in array
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
				}},
			},
		},
		{
			"{ 2 < .foo }", // symmetric: int > 2 in array
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
			"{ 2 <= .foo }", // symmetric: in >= 2 in array
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
			"{ 3 > .foo }", // symmetric: in < 3 in array
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
			"{ 3 >= .foo }", // symmetric: in <= 3 in array
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
			"{ 2.5 = .foo }", // symmetric: float in array
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
			"{ 3.14 != .foo }", // symmetric: float not in array
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
				}},
			},
		},
		{
			"{ 2.0 > .foo }", // symmetric: float < 2.0 in array
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
			"{ 2.0 >= .foo }", // symmetric: float <= 2.0 in array
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
			"{ 3.5 < .foo }", // symmetric: float > 3.5 in array
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
			"{ 3.5 <= .foo }", // symmetric: float >= 3.5 in array
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
			"{ true = .foo }", // symmetric: boolean in array
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
			"{ false != .foo }", // symmetric: boolean not in array
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{1}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, false})}},
					&mockSpan{id: []byte{2}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{false, false})}},
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{id: []byte{3}, attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticBooleanArray([]bool{true, true})}},
				}},
			},
		},
		{
			"{ !false = .foo }", // symmetric: negated boolean in array
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

func TestBinOp(t *testing.T) {
	testCases := []struct {
		op        Operator
		lhs       Static
		rhs       Static
		expected  Static
		symmetric bool
	}{
		{
			op:       OpGreater,
			lhs:      NewStaticString("foo"),
			rhs:      NewStaticString(""),
			expected: StaticTrue,
		},
		// Comparisons of strings starting with a number were previously broken.
		{
			op:       OpGreater,
			lhs:      NewStaticString("123"),
			rhs:      NewStaticString(""),
			expected: StaticTrue,
		},
		// Test cases for of array operations
		{
			op:        OpEqual,
			lhs:       NewStaticInt(2),
			rhs:       NewStaticIntArray([]int{1, 2, 3}),
			expected:  StaticTrue,
			symmetric: true,
		},
		{
			op:        OpEqual,
			lhs:       NewStaticInt(2),
			rhs:       NewStaticStringArray([]string{"1", "2", "3"}),
			expected:  StaticFalse,
			symmetric: true,
		},
		{
			op:        OpNotEqual,
			lhs:       NewStaticString("1"),
			rhs:       NewStaticStringArray([]string{"2", "3", "4"}),
			expected:  StaticTrue,
			symmetric: true,
		},
		{
			op:        OpNotEqual,
			lhs:       NewStaticString("3"),
			rhs:       NewStaticStringArray([]string{"1", "2", "3"}),
			expected:  StaticFalse,
			symmetric: true,
		},
		// Test cases for of array regex operations (regex is not symmetrical, the rhs is always the regex)
		{
			op:       OpRegex,
			lhs:      NewStaticString("two"),
			rhs:      NewStaticStringArray([]string{"one", "tw.*", "three"}),
			expected: StaticTrue,
		},
		{
			op:       OpRegex,
			lhs:      NewStaticString("two"),
			rhs:      NewStaticStringArray([]string{"on.*", "three", ".*four.*"}),
			expected: StaticFalse,
		},
		{
			op:       OpRegex,
			lhs:      NewStaticStringArray([]string{"one", "two", "three"}),
			rhs:      NewStaticString("tw.*"),
			expected: StaticTrue,
		},
		{
			op:       OpRegex,
			lhs:      NewStaticStringArray([]string{"one", "three", "four"}),
			rhs:      NewStaticString("tw.*"),
			expected: StaticFalse,
		},
		{
			op:       OpRegex,
			lhs:      NewStaticString("1.1"),
			rhs:      NewStaticFloatArray([]float64{1.1, 2.2, 3.3}),
			expected: StaticFalse,
		},
		{
			op:       OpNotRegex,
			lhs:      NewStaticString("one"),
			rhs:      NewStaticStringArray([]string{"tw.*", "th.*", "fo.*"}),
			expected: StaticTrue,
		},
		{
			op:       OpNotRegex,
			lhs:      NewStaticString("one"),
			rhs:      NewStaticStringArray([]string{"on.*", "tw.*", "th.*"}),
			expected: StaticFalse,
		},
		{
			op:       OpNotRegex,
			lhs:      NewStaticStringArray([]string{"two", "three", "four"}),
			rhs:      NewStaticString("on.*"),
			expected: StaticTrue,
		},
		{
			op:       OpNotRegex,
			lhs:      NewStaticStringArray([]string{"one", "two", "three"}),
			rhs:      NewStaticString("on.*"),
			expected: StaticFalse,
		},
	}

	for _, tc := range testCases {
		op := newBinaryOperation(tc.op, tc.lhs, tc.rhs)

		actual, err := op.execute(nil)
		require.NoError(t, err)
		require.Equal(t, tc.expected, actual, fmt.Sprintf("%s %s %s", tc.lhs, tc.op, tc.rhs))

		if tc.symmetric {
			op = newBinaryOperation(tc.op, tc.rhs, tc.lhs)
			actual, err = op.execute(nil)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual, fmt.Sprintf("%s %s %s", tc.rhs, tc.op, tc.lhs))
		}
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
		{
			query: `{ nil = nil }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("float"): NewStaticFloat(2.0),
				},
			},
			matches: false,
		},
		{
			query: `{ nil != nil }`,
			span: &mockSpan{
				attributes: map[Attribute]Static{
					NewAttribute("float"): NewStaticFloat(2.0),
				},
			},
			matches: false,
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

func TestNotParentWithEmptyLHS(t *testing.T) {
	// Build a single spanset with only RHS candidates matching name="list-articles"
	rhsChild := newMockSpan([]byte{1}).WithNestedSetInfo(0, 1, 2)
	rhsChild.attributes[IntrinsicNameAttribute] = NewStaticString("list-articles")

	ss := []*Spanset{
		{Spans: []Span{rhsChild}},
	}

	query := "{ span:name = `some-span-that-does-not-exist` } !< { span:name = `list-articles` }"
	ast, err := Parse(query)
	require.NoError(t, err)

	out, err := ast.Pipeline.evaluate(ss)
	require.NoError(t, err)

	// Expect the RHS span to be returned because no LHS parent exists (negated parent)
	require.Len(t, out, 1)
	require.Len(t, out[0].Spans, 1)
	nameStatic, _ := out[0].Spans[0].AttributeFor(IntrinsicNameAttribute)
	expected := NewStaticString("list-articles")
	require.True(t, nameStatic.Equals(&expected))
}
