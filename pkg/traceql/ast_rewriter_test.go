package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinaryOpToArrayOpRewriter(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "empty",
			query: "{ }",
			want:  "{ true }",
		},
		// handle || to =
		{
			name:  "simple or",
			query: "{ .attr1 = `val1` || .attr1 = `val2` }",
			want:  "{ .attr1 = [`val1`, `val2`] }",
		},
		{
			name:  "simple or ints",
			query: "{ .attr1 = 1 || .attr1 = 2 }",
			want:  "{ .attr1 = [1, 2] }",
		},
		{
			name:  "multiple or",
			query: "{ .attr1 = `val1` || .attr1 = `val2` || .attr1 = `val3`}",
			want:  "{ .attr1 = [`val1`, `val2`, `val3`] }",
		},
		{
			name:  "mixed or and",
			query: "{ .attr1 = `val1` || .attr1 = `val2` && .attr2 = `val3`}",
			want:  "{ (.attr1 = [`val1`, `val2`]) && (.attr2 = `val3`) }",
		},
		{
			name:  "mixed or and interleaved",
			query: "{ .attr1 = `val1` || .attr1 = `val2` && .attr2 = `val3` || .attr1 = `val4`}",
			want:  "{ ((.attr1 = [`val1`, `val2`]) && (.attr2 = `val3`)) || (.attr1 = `val4`) }",
		},
		{
			name:  "multiple or wrong operator",
			query: "{ .attr1 = `val1` || .attr1 = `val2` || .attr1 != `val3`}",
			want:  "{ (.attr1 = [`val1`, `val2`]) || (.attr1 != `val3`) }",
		},
		{
			name:  "multiple or wrong type",
			query: "{ .attr1 = `val1` || .attr1 = `val2` || .attr1 = 1}",
			want:  "{ (.attr1 = [`val1`, `val2`]) || (.attr1 = 1) }",
		},
		// handle || to =~
		{
			name:  "regex simple or",
			query: "{ .attr1 =~ `val1` || .attr1 =~ `val2` }",
			want:  "{ .attr1 =~ [`val1`, `val2`] }",
		},
		{
			name:  "regex multiple or",
			query: "{ .attr1 =~ `val1` || .attr1 =~ `val2` || .attr1 =~ `val3`}",
			want:  "{ .attr1 =~ [`val1`, `val2`, `val3`] }",
		},
		{
			name:  "regex mixed or and",
			query: "{ .attr1 =~ `val1` || .attr1 =~ `val2` && .attr2 =~ `val3`}",
			want:  "{ (.attr1 =~ [`val1`, `val2`]) && (.attr2 =~ `val3`) }",
		},
		{
			name:  "regex mixed or and interleaved",
			query: "{ .attr1 =~ `val1` || .attr1 =~ `val2` && .attr2 =~ `val3` || .attr1 =~ `val4`}",
			want:  "{ ((.attr1 =~ [`val1`, `val2`]) && (.attr2 =~ `val3`)) || (.attr1 =~ `val4`) }",
		},
		// handle && to !=
		{
			name:  "simple and",
			query: "{ .attr1 != `val1` && .attr1 != `val2` }",
			want:  "{ .attr1 != [`val1`, `val2`] }",
		},
		{
			name:  "multiple and",
			query: "{ .attr1 != `val1` && .attr1 != `val2` && .attr1 != `val3`}",
			want:  "{ .attr1 != [`val1`, `val2`, `val3`] }",
		},
		{
			name:  "mixed and or",
			query: "{ .attr1 != `val1` && .attr1 != `val2` || .attr2 != `val3`}",
			want:  "{ (.attr1 != [`val1`, `val2`]) || (.attr2 != `val3`) }",
		},
		{
			name:  "multiple and wrong operator",
			query: "{ .attr1 != `val1` && .attr1 != `val2` && .attr1 = `val3`}",
			want:  "{ (.attr1 != [`val1`, `val2`]) && (.attr1 = `val3`) }",
		},
		// handle && to !~
		{
			name:  "regex simple and",
			query: "{ .attr1 !~ `val1` && .attr1 !~ `val2` }",
			want:  "{ .attr1 !~ [`val1`, `val2`] }",
		},
		{
			name:  "regex multiple and",
			query: "{ .attr1 !~ `val1` && .attr1 !~ `val2` && .attr1 !~ `val3`}",
			want:  "{ .attr1 !~ [`val1`, `val2`, `val3`] }",
		},
		{
			name:  "regex mixed and or",
			query: "{ .attr1 !~ `val1` && .attr1 !~ `val2` || .attr2 !~ `val3`}",
			want:  "{ (.attr1 !~ [`val1`, `val2`]) || (.attr2 !~ `val3`) }",
		},
	}

	rewriter := newBinaryOpToArrayOpRewriter()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := Parse(tc.query)
			require.NoError(t, err)

			rewrite := rewriter.RewriteRoot(expr)
			require.Equal(t, tc.want, rewrite.String())
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
				rw := &fieldExpressionRewriter{rewriteFunctions: []fieldExpressionRewriteFn{func(op FieldExpression) FieldExpression {
					if index < len(tc.visitOrder) {
						requireMatchingFieldExpression(t, op, tc.visitOrder[index])
					}
					index++
					return op
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
	switch a := a.(type) {
	case *BinaryOperation:
		if b, ok := b.(*BinaryOperation); ok {
			require.Equal(t, a.Op, b.Op)
		} else {
			require.Fail(t, "expected BinaryOperation, got %T", b)
		}
	case *UnaryOperation:
		if b, ok := b.(*UnaryOperation); ok {
			require.Equal(t, a.Op, b.Op)
		} else {
			require.Fail(t, "expected BinaryOperation, got %T", b)
		}
	default:
		require.Equal(t, a, b)
	}
}
