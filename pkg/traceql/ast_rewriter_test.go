package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinaryOpToArrayOpRewriter(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		want     string
		optCount int
	}{
		{
			name:  "empty",
			query: "{ }",
			want:  "{ true }",
		},
		// handle || to IN
		{
			name:     "simple or",
			query:    "{ .attr1 = `val1` || .attr1 = `val2` }",
			want:     "{ .attr1 IN [`val1`, `val2`] }",
			optCount: 1,
		},
		{
			name:     "simple or ints",
			query:    "{ .attr1 = 1 || .attr1 = 2 }",
			want:     "{ .attr1 IN [1, 2] }",
			optCount: 1,
		},
		{
			name:     "multiple or",
			query:    "{ .attr1 = `val1` || .attr1 = `val2` || .attr1 = `val3`}",
			want:     "{ .attr1 IN [`val1`, `val2`, `val3`] }",
			optCount: 2,
		},
		{
			name:     "mixed or and",
			query:    "{ .attr1 = `val1` || .attr1 = `val2` && .attr2 = `val3`}",
			want:     "{ (.attr1 IN [`val1`, `val2`]) && (.attr2 = `val3`) }",
			optCount: 1,
		},
		{
			name:     "mixed or and interleaved",
			query:    "{ .attr1 = `val1` || .attr1 = `val2` && .attr2 = `val3` || .attr1 = `val4`}",
			want:     "{ ((.attr1 IN [`val1`, `val2`]) && (.attr2 = `val3`)) || (.attr1 = `val4`) }",
			optCount: 1,
		},
		{
			name:     "multiple or wrong operator",
			query:    "{ .attr1 = `val1` || .attr1 = `val2` || .attr1 != `val3`}",
			want:     "{ (.attr1 IN [`val1`, `val2`]) || (.attr1 != `val3`) }",
			optCount: 1,
		},
		{
			name:     "multiple or wrong type",
			query:    "{ .attr1 = `val1` || .attr1 = `val2` || .attr1 = 1}",
			want:     "{ (.attr1 IN [`val1`, `val2`]) || (.attr1 = 1) }",
			optCount: 1,
		},
		// handle || to MATCH ANY
		{
			name:     "regex simple or",
			query:    "{ .attr1 =~ `val1` || .attr1 =~ `val2` }",
			want:     "{ .attr1 MATCH ANY [`val1`, `val2`] }",
			optCount: 1,
		},
		{
			name:     "regex multiple or",
			query:    "{ .attr1 =~ `val1` || .attr1 =~ `val2` || .attr1 =~ `val3`}",
			want:     "{ .attr1 MATCH ANY [`val1`, `val2`, `val3`] }",
			optCount: 2,
		},
		{
			name:     "regex mixed or and",
			query:    "{ .attr1 =~ `val1` || .attr1 =~ `val2` && .attr2 =~ `val3`}",
			want:     "{ (.attr1 MATCH ANY [`val1`, `val2`]) && (.attr2 =~ `val3`) }",
			optCount: 1,
		},
		{
			name:     "regex mixed or and interleaved",
			query:    "{ .attr1 =~ `val1` || .attr1 =~ `val2` && .attr2 =~ `val3` || .attr1 =~ `val4`}",
			want:     "{ ((.attr1 MATCH ANY [`val1`, `val2`]) && (.attr2 =~ `val3`)) || (.attr1 =~ `val4`) }",
			optCount: 1,
		},
		// handle && to NOT IN
		{
			name:     "simple and",
			query:    "{ .attr1 != `val1` && .attr1 != `val2` }",
			want:     "{ .attr1 NOT IN [`val1`, `val2`] }",
			optCount: 1,
		},
		{
			name:     "multiple and",
			query:    "{ .attr1 != `val1` && .attr1 != `val2` && .attr1 != `val3`}",
			want:     "{ .attr1 NOT IN [`val1`, `val2`, `val3`] }",
			optCount: 2,
		},
		{
			name:     "mixed and or",
			query:    "{ .attr1 != `val1` && .attr1 != `val2` || .attr2 != `val3`}",
			want:     "{ (.attr1 NOT IN [`val1`, `val2`]) || (.attr2 != `val3`) }",
			optCount: 1,
		},
		{
			name:     "multiple and wrong operator",
			query:    "{ .attr1 != `val1` && .attr1 != `val2` && .attr1 = `val3`}",
			want:     "{ (.attr1 NOT IN [`val1`, `val2`]) && (.attr1 = `val3`) }",
			optCount: 1,
		},
		// handle && to MATCH NONE
		{
			name:     "regex simple and",
			query:    "{ .attr1 !~ `val1` && .attr1 !~ `val2` }",
			want:     "{ .attr1 MATCH NONE [`val1`, `val2`] }",
			optCount: 1,
		},
		{
			name:     "regex multiple and",
			query:    "{ .attr1 !~ `val1` && .attr1 !~ `val2` && .attr1 !~ `val3`}",
			want:     "{ .attr1 MATCH NONE [`val1`, `val2`, `val3`] }",
			optCount: 2,
		},
		{
			name:     "regex mixed and or",
			query:    "{ .attr1 !~ `val1` && .attr1 !~ `val2` || .attr2 !~ `val3`}",
			want:     "{ (.attr1 MATCH NONE [`val1`, `val2`]) || (.attr2 !~ `val3`) }",
			optCount: 1,
		},
		// reverse operands
		{
			name:     "reversed or",
			query:    "{ `val1` = .a || `val2` = .a }",
			want:     "{ .a IN [`val1`, `val2`] }",
			optCount: 1,
		},
		{
			name:     "mixed or",
			query:    "{ `val1` = .a || .a = `val2` }",
			want:     "{ .a IN [`val1`, `val2`] }",
			optCount: 1,
		},
		{
			name:     "reversed int or",
			query:    "{ 1 = .a || 2 = .a }",
			want:     "{ .a IN [1, 2] }",
			optCount: 1,
		},
		// spanset filter
		{
			name:     "left spanset filter",
			query:    "{ .a = `val1` || .a = `val2` } >> { }",
			want:     "({ .a IN [`val1`, `val2`] }) >> ({ true })",
			optCount: 1,
		},
		{
			name:     "right spanset filter",
			query:    "{ } >> { .a = `val1` || .a = `val2` }",
			want:     "({ true }) >> ({ .a IN [`val1`, `val2`] })",
			optCount: 1,
		},
		// scope mismatch
		{
			name:     "scoped attribute mismatch",
			query:    "{ .a = `val1` || resource.a = `val2` }",
			want:     "{ (.a = `val1`) || (resource.a = `val2`) }",
			optCount: 0,
		},
	}

	rewriter := newBinaryOpToArrayOpRewriter()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parseWithOptimizationOption(tc.query, true)
			require.NoError(t, err)

			rewrite := rewriter.RewriteRoot(expr)
			require.Equal(t, tc.want, rewrite.String())
			require.Equal(t, tc.optCount, rewrite.OptimizationCount)
		})
	}
}

func TestFieldExpressionRewriter_VisitOrder(t *testing.T) {
	t.Run("BinaryOperation", func(t *testing.T) {
		cases := []struct {
			name       string
			query      string
			visitOrder []FieldExpression
		}{
			{
				name:  "simple",
				query: `{ .attr1 = "val1" }`,
				visitOrder: []FieldExpression{
					Attribute{Name: "attr1"},
					NewStaticString("val1"),
					&BinaryOperation{Op: OpEqual},
				},
			},
			{
				name:  "and",
				query: `{ .attr1 = "val1" && .attr2 != "val2" }`,
				visitOrder: []FieldExpression{
					Attribute{Name: "attr1"},
					NewStaticString("val1"),
					&BinaryOperation{Op: OpEqual},
					Attribute{Name: "attr2"},
					NewStaticString("val2"),
					&BinaryOperation{Op: OpNotEqual},
					&BinaryOperation{Op: OpAnd},
				},
			},
			{
				name:  "count",
				query: `{} | count() > 11`,
				visitOrder: []FieldExpression{
					NewStaticBool(true),
					nil,
					NewStaticInt(11),
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				expr, err := Parse(tc.query)
				require.NoError(t, err)

				var index int
				rw := &fieldExpressionRewriter{rewriteFunctions: []fieldExpressionRewriteFn{func(op FieldExpression) (FieldExpression, int) {
					if index < len(tc.visitOrder) {
						requireMatchingFieldExpression(t, op, tc.visitOrder[index])
					}
					index++
					return op, 0
				}}}
				rw.RewriteRoot(expr)
				require.Equal(t, len(tc.visitOrder), index, "visited element count mismatch")
			})
		}
	})
}

// requireMatchingFieldExpression compares field expressions for equality, without descending into subexpressions
func requireMatchingFieldExpression(t *testing.T, a, b FieldExpression) {
	t.Helper()
	switch opA := a.(type) {
	case *BinaryOperation:
		if opB, ok := b.(*BinaryOperation); ok {
			require.Equal(t, opA.Op, opB.Op)
		} else {
			require.Fail(t, "expected BinaryOperation, got %T", opB)
		}
	case *UnaryOperation:
		if opB, ok := b.(*UnaryOperation); ok {
			require.Equal(t, opA.Op, opB.Op)
		} else {
			require.Fail(t, "expected BinaryOperation, got %T", opB)
		}
	default:
		require.Equal(t, a, b)
	}
}
