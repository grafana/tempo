package traceql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpansetFilter_extractConditions(t *testing.T) {
	tests := []struct {
		query         string
		conditions    []Condition
		allConditions bool
	}{
		{
			query: `{ .foo = "bar" && "bzz" = .fzz }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				newCondition(NewAttribute("fzz"), OpEqual, NewStaticString("bzz")),
			},
			allConditions: true,
		},
		{
			query: `{ .foo = "bar" || "bzz" = .fzz }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				newCondition(NewAttribute("fzz"), OpEqual, NewStaticString("bzz")),
			},
			allConditions: false,
		},
		{
			query:         `{ "foo" = "bar" }`,
			conditions:    []Condition{},
			allConditions: true,
		},
		{
			query: `{ .foo = .bar }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpNone),
				newCondition(NewAttribute("bar"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ (.foo = "bar") = true }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
			},
			allConditions: true,
		},
		{
			query: `{ true = (.foo = "bar") }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
			},
			allConditions: true,
		},
		{
			query: `{ (.foo = "bar") = .bar }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				newCondition(NewAttribute("bar"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ .bar = (.foo = "bar") }`,
			conditions: []Condition{
				newCondition(NewAttribute("bar"), OpNone),
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
			},
			allConditions: true,
		},
		// TODO we need a smarter engine to handle this - we can either negate OpEqual or just fetch .fzz
		// {
		// 	query: `{ (.foo = "bar") = !(.fzz = "bzz") }`,
		// 	conditions: []Condition{
		// 		newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
		// 		newCondition(NewAttribute("fzz"), OpNotEqual, NewStaticString("bzz")),
		// 	},
		// 	allConditions: true,
		// },
		{
			query: `{ (.foo = "bar") = !.bar }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				newCondition(NewAttribute("bar"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ .foo = .bar + 1 }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpNone),
				newCondition(NewAttribute("bar"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ (.foo = 2) && (.bar / 1 > 3) }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticInt(2)),
				newCondition(NewAttribute("bar"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ .foo && true }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ true || .foo }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpNone),
			},
			allConditions: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			spansetFilter := expr.Pipeline.Elements[0].(*SpansetFilter)

			req := &FetchSpansRequest{
				Conditions:    []Condition{},
				AllConditions: true,
			}
			spansetFilter.extractConditions(req)

			assert.Equal(t, tt.conditions, req.Conditions)
			assert.Nil(t, req.SecondPassConditions)
			assert.Equal(t, tt.allConditions, req.AllConditions, "FetchSpansRequest.AllConditions")
		})
	}
}

func TestScalarFilter_extractConditions(t *testing.T) {
	tests := []struct {
		query         string
		conditions    []Condition
		allConditions bool
	}{
		{
			query: `{ .foo = "a" } | count() > 10`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
			},
			allConditions: false,
		},
		{
			query: `{ .foo = "a" } | avg(duration) > 10ms`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
				newCondition(NewIntrinsic(IntrinsicDuration), OpNone),
			},
			allConditions: false,
		},
		{
			query: `({ span.http.status_code = 200 } | count()) > ({ span.http.status_code = 500 } | count())`,
			conditions: []Condition{
				newCondition(NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"), OpEqual, NewStaticInt(200)),
				newCondition(NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"), OpEqual, NewStaticInt(500)),
			},
			allConditions: false,
		},
		{
			query: `{ .foo = "a" } | 3 > 2`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
			},
			allConditions: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			req := &FetchSpansRequest{
				Conditions:    []Condition{},
				AllConditions: true,
			}
			expr.Pipeline.extractConditions(req)

			assert.Equal(t, tt.conditions, req.Conditions)
			assert.Nil(t, req.SecondPassConditions)
			assert.Equal(t, tt.allConditions, req.AllConditions, "FetchSpansRequest.AllConditions")
		})
	}
}

func TestStructuralNestedSet_extractConditions(t *testing.T) {
	tests := []struct {
		query         string
		conditions    []Condition
		allConditions bool
	}{
		{
			query: `{} >> {}`,
			conditions: []Condition{
				newCondition(NewIntrinsic(IntrinsicStructuralDescendant), OpNone),
			},
			allConditions: false,
		},
		{
			query: `{ nestedSetRight = 2 }`,
			conditions: []Condition{
				newCondition(NewIntrinsic(IntrinsicNestedSetRight), OpEqual, NewStaticInt(2)),
			},
			allConditions: true,
		},

		{
			query: `{ nestedSetParent = 1 } > { nestedSetLeft < 3 }`,
			conditions: []Condition{
				newCondition(NewIntrinsic(IntrinsicStructuralChild), OpNone),
				newCondition(NewIntrinsic(IntrinsicNestedSetParent), OpEqual, NewStaticInt(1)),
				newCondition(NewIntrinsic(IntrinsicNestedSetLeft), OpLess, NewStaticInt(3)),
			},
			allConditions: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			req := &FetchSpansRequest{
				Conditions:    []Condition{},
				AllConditions: true,
			}
			expr.Pipeline.extractConditions(req)

			assert.Equal(t, tt.conditions, req.Conditions)
			assert.Nil(t, req.SecondPassConditions)
			assert.Equal(t, tt.allConditions, req.AllConditions, "FetchSpansRequest.AllConditions")
		})
	}
}

func TestSelect_extractConditions(t *testing.T) {
	tests := []struct {
		query                string
		conditions           []Condition
		secondPassConditions []Condition
		allConditions        bool
	}{
		{
			query: `{ .foo = "a" } | select(resource.service.name)`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
			},
			secondPassConditions: []Condition{
				newCondition(NewScopedAttribute(AttributeScopeResource, false, "service.name"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ } | select(.name,name)`,
			conditions: []Condition{
				newCondition(NewIntrinsic(IntrinsicSpanStartTime), OpNone),
			},
			secondPassConditions: []Condition{
				newCondition(NewAttribute("name"), OpNone),
				newCondition(NewIntrinsic(IntrinsicName), OpNone),
			},
			allConditions: true,
		},
		{
			// Pipleline elements after a select are always directed to the second pass
			query: `{ } | select(span.foo) | { span.foo = "a" && span.bar = "b"}`,
			conditions: []Condition{
				newCondition(NewIntrinsic(IntrinsicSpanStartTime), OpNone),
			},
			secondPassConditions: []Condition{
				// span.foo=a has no effect because it's already covered by the select statement
				newCondition(NewScopedAttribute(AttributeScopeSpan, false, "foo"), OpNone),
				newCondition(NewScopedAttribute(AttributeScopeSpan, false, "bar"), OpEqual, NewStaticString("b")),
			},
			allConditions: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			req := &FetchSpansRequest{
				Conditions:    []Condition{},
				AllConditions: true,
			}
			expr.Pipeline.extractConditions(req)

			assert.Equal(t, tt.conditions, req.Conditions)
			assert.Equal(t, tt.secondPassConditions, req.SecondPassConditions)
			assert.Equal(t, tt.allConditions, req.AllConditions, "FetchSpansRequest.AllConditions")
		})
	}
}

func TestMetricsAggregate_extractConditions(t *testing.T) {
	tests := []struct {
		query  string
		first  []Condition
		second []Condition
		all    bool
	}{
		{
			// Empty spanset implies start time
			query:  `{} | rate() by (name)`,
			first:  []Condition{newCondition(IntrinsicSpanStartTimeAttribute, OpNone)},
			second: []Condition{newCondition(IntrinsicNameAttribute, OpNone)},
			all:    true,
		},
		{
			// by() clause doesn't overwrite the existing condition
			query:  `{name="foo"} | rate() by (name)`,
			first:  []Condition{newCondition(IntrinsicNameAttribute, OpEqual, NewStaticString("foo"))},
			second: nil,
			all:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			req := &FetchSpansRequest{
				AllConditions: true,
			}
			expr.extractConditions(req)

			require.Equal(t, tt.first, req.Conditions)
			require.Equal(t, tt.second, req.SecondPassConditions)
			require.Equal(t, tt.all, req.AllConditions, "FetchSpansRequest.AllConditions")
		})
	}
}

func TestBinaryOperation_extractConditions(t *testing.T) {
	tests := []struct {
		name string
		op   *BinaryOperation
		want []Condition
	}{
		{
			name: "string array equal",
			op:   &BinaryOperation{Op: OpEqual, LHS: NewAttribute("attr"), RHS: NewStaticStringArray([]string{"a", "b"})},
			want: []Condition{
				newCondition(NewAttribute("attr"), OpEqual, NewStaticStringArray([]string{"a", "b"})),
			},
		},
		{
			name: "int array equal",
			op:   &BinaryOperation{Op: OpEqual, LHS: NewAttribute("attr"), RHS: NewStaticIntArray([]int{1, 2, 3})},
			want: []Condition{
				newCondition(NewAttribute("attr"), OpEqual, NewStaticIntArray([]int{1, 2, 3})),
			},
		},
		{
			name: "string array not regex",
			op:   &BinaryOperation{Op: OpNotRegex, LHS: NewAttribute("attr"), RHS: NewStaticStringArray([]string{"a.*", "b.*"})},
			want: []Condition{
				newCondition(NewAttribute("attr"), OpNotRegex, NewStaticStringArray([]string{"a.*", "b.*"})),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &FetchSpansRequest{
				AllConditions: true,
			}
			tt.op.extractConditions(req)
			require.Equal(t, tt.want, req.Conditions)
			require.Equal(t, true, req.AllConditions, "allConditions should be true")
		})
	}
}
