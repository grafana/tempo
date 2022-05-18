package traceql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// func TestTest(t *testing.T) {
// 	tests := []struct {
// 		in  string
// 		err error
// 	}{
// 		{in: "max(duration) > 3s && { status = error || http.status = 500 }", err: newParseError("", 1, 12)},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.in, func(t *testing.T) {
// 			actual, err := Parse(tc.in)

// 			assert.Equal(t, tc.err, err)
// 			spew.Dump(actual)
// 		})
// 	}
// }

// jpe    - should `3 = 3` be valid but not `true`

func TestPipelineErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "{ a } | { b", err: newParseError("syntax error: unexpected $end", 1, 12)},
		{in: "{ a | b }", err: newParseError("syntax error: unexpected |", 1, 5)},
		{in: "({ a } | { b }", err: newParseError("syntax error: unexpected $end, expecting ) or |", 1, 15)},
		{in: "({ a } | { b }) + ({ a } | { b })", err: newParseError("syntax error: unexpected +", 1, 17)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestPipelineOperatorPrecedence(t *testing.T) {
	tests := []struct {
		in       string
		expected SpansetOperation
	}{
		{
			in: "({ a } | { b }) > ({ a } | { b }) && ({ a } | { b })",
			expected: newSpansetOperation(opSpansetChild,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newSpansetFilter(newStaticIdentifier("b")),
				),
				newSpansetOperation(opSpansetAnd,
					newPipeline(
						newSpansetFilter(newStaticIdentifier("a")),
						newSpansetFilter(newStaticIdentifier("b")),
					),
					newPipeline(
						newSpansetFilter(newStaticIdentifier("a")),
						newSpansetFilter(newStaticIdentifier("b")),
					),
				),
			),
		},
		{
			in: "(({ a } | { b }) > ({ a } | { b })) && ({ a } | { b })",
			expected: newSpansetOperation(opSpansetAnd,
				newSpansetOperation(opSpansetChild,
					newPipeline(
						newSpansetFilter(newStaticIdentifier("a")),
						newSpansetFilter(newStaticIdentifier("b")),
					),
					newPipeline(
						newSpansetFilter(newStaticIdentifier("a")),
						newSpansetFilter(newStaticIdentifier("b")),
					),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newSpansetFilter(newStaticIdentifier("b")),
				),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestPipelineSpansetOperators(t *testing.T) {
	tests := []struct {
		in       string
		expected SpansetOperation
	}{
		{
			in: "({ a } | { b }) > ({ a } | { b })",
			expected: newSpansetOperation(opSpansetChild,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newSpansetFilter(newStaticIdentifier("b")),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newSpansetFilter(newStaticIdentifier("b")),
				),
			),
		},
		{
			in: "({ a } | { b }) && ({ a } | { b })",
			expected: newSpansetOperation(opSpansetAnd,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newSpansetFilter(newStaticIdentifier("b")),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newSpansetFilter(newStaticIdentifier("b")),
				),
			),
		},
		{
			in: "({ a } | { b }) >> ({ a } | { b })",
			expected: newSpansetOperation(opSpansetDescendant,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newSpansetFilter(newStaticIdentifier("b")),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newSpansetFilter(newStaticIdentifier("b")),
				),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestPipelineScalarOperators(t *testing.T) {
	tests := []struct {
		in       string
		expected ScalarFilter
	}{
		{
			in: "({ a } | count()) = ({ a } | count())",
			expected: newScalarFilter(opEqual,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ a } | count()) != ({ a } | count())",
			expected: newScalarFilter(opNotEqual,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ a } | count()) < ({ a } | count())",
			expected: newScalarFilter(opLess,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ a } | count()) <= ({ a } | count())",
			expected: newScalarFilter(opLessEqual,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ a } | count()) >= ({ a } | count())",
			expected: newScalarFilter(opGreaterEqual,
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newStaticIdentifier("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestPipelines(t *testing.T) {
	tests := []struct {
		in       string
		expected Pipeline
	}{
		{
			in: "{ a } | { b }",
			expected: newPipeline(
				newSpansetFilter(newStaticIdentifier("a")),
				newSpansetFilter(newStaticIdentifier("b")),
			),
		},
		{
			in: "{ a } | count() > 1",
			expected: newPipeline(
				newSpansetFilter(newStaticIdentifier("a")),
				newScalarFilter(opGreater, newAggregate(aggregateCount, nil), newStaticInt(1)),
			),
		},
		{
			in: "{ a } | by(namespace) | coalesce() | avg(duration) = 1s ",
			expected: newPipeline(
				newSpansetFilter(newStaticIdentifier("a")),
				newGroupOperation(newStaticIdentifier("namespace")),
				newCoalesceOperation(),
				newScalarFilter(opEqual, newAggregate(aggregateAvg, newStaticIdentifier("duration")), newStaticDuration(time.Second)),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{tc.expected}, actual)
		})
	}
}

func TestGroupCoalesceErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "by(a) && { b }", err: newParseError("syntax error: unexpected &&", 0, 7)},
		{in: "group()", err: newParseError("syntax error: unexpected IDENTIFIER", 1, 1)},
		{in: "coalesce()", err: newParseError("syntax error: unexpected COALESCE", 1, 1)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGroupCoalesceOperation(t *testing.T) {
	tests := []struct {
		in       string
		expected Pipeline
	}{
		{in: "by(a) | coalesce()", expected: newPipeline(newGroupOperation(newStaticIdentifier("a")), newCoalesceOperation())},
		{in: "by(a + b)", expected: newPipeline(newGroupOperation(newBinaryOperation(opAdd, newStaticIdentifier("a"), newStaticIdentifier("b"))))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{tc.expected}, actual)
		})
	}
}

func TestSpansetExpressionErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "{ true } &&", err: newParseError("syntax error: unexpected $end, expecting { or (", 1, 12)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestSpansetExpressionPrecedence(t *testing.T) {
	tests := []struct {
		in       string
		expected SpansetOperation
	}{
		{
			in: "{ true } && { false } >> { `a` }",
			expected: SpansetOperation{
				op: opSpansetDescendant,
				lhs: SpansetOperation{
					op:  opSpansetAnd,
					lhs: newSpansetFilter(newStaticBool(true)),
					rhs: newSpansetFilter(newStaticBool(false)),
				},
				rhs: newSpansetFilter(newStaticString("a")),
			},
		},
		{
			in: "{ true } >> { false } && { `a` }",
			expected: SpansetOperation{
				op:  opSpansetDescendant,
				lhs: newSpansetFilter(newStaticBool(true)),
				rhs: SpansetOperation{
					op:  opSpansetAnd,
					lhs: newSpansetFilter(newStaticBool(false)),
					rhs: newSpansetFilter(newStaticString("a")),
				},
			},
		},
		{
			in: "({ true } >> { false }) && { `a` }",
			expected: SpansetOperation{
				op: opSpansetAnd,
				lhs: SpansetOperation{
					op:  opSpansetDescendant,
					lhs: newSpansetFilter(newStaticBool(true)),
					rhs: newSpansetFilter(newStaticBool(false)),
				},
				rhs: newSpansetFilter(newStaticString("a")),
			},
		},
		{
			in: "{ true } >> { false } ~ { `a` }",
			expected: SpansetOperation{
				op: opSpansetSibling,
				lhs: SpansetOperation{
					op:  opSpansetDescendant,
					lhs: newSpansetFilter(newStaticBool(true)),
					rhs: newSpansetFilter(newStaticBool(false)),
				},
				rhs: newSpansetFilter(newStaticString("a")),
			},
		},
		{
			in: "{ true } ~ { false } >> { `a` }",
			expected: SpansetOperation{
				op: opSpansetDescendant,
				lhs: SpansetOperation{
					op:  opSpansetSibling,
					lhs: newSpansetFilter(newStaticBool(true)),
					rhs: newSpansetFilter(newStaticBool(false)),
				},
				rhs: newSpansetFilter(newStaticString("a")),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestSpansetExpressionOperators(t *testing.T) {
	tests := []struct {
		in       string
		expected SpansetOperation
	}{
		{in: "{ true } && { false }", expected: newSpansetOperation(opSpansetAnd, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		{in: "{ true } > { false }", expected: newSpansetOperation(opSpansetChild, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		{in: "{ true } >> { false }", expected: newSpansetOperation(opSpansetDescendant, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		{in: "{ true } || { false }", expected: newSpansetOperation(opSpansetUnion, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		{in: "{ true } ~ { false }", expected: newSpansetOperation(opSpansetSibling, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestScalarExpressionErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "(avg(foo) > count()) + sum(bar)", err: newParseError("syntax error: unexpected +", 1, 22)},
		{in: "count(", err: newParseError("syntax error: unexpected $end, expecting )", 1, 7)},
		{in: "count(avg)", err: newParseError("syntax error: unexpected IDENTIFIER, expecting )", 1, 7)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestScalarExpressionPrecedence(t *testing.T) {
	tests := []struct {
		in       string
		expected ScalarFilter
	}{
		{
			in: "avg(foo) > count() + sum(bar)",
			expected: newScalarFilter(opGreater,
				newAggregate(aggregateAvg, newStaticIdentifier("foo")),
				newScalarOperation(opAdd,
					newAggregate(aggregateCount, nil),
					newAggregate(aggregateSum, newStaticIdentifier("bar")),
				),
			),
		},
		{
			in: "avg(foo) + count() > sum(bar)",
			expected: newScalarFilter(opGreater,
				newScalarOperation(opAdd,
					newAggregate(aggregateAvg, newStaticIdentifier("foo")),
					newAggregate(aggregateCount, nil),
				),
				newAggregate(aggregateSum, newStaticIdentifier("bar")),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestScalarExpressionOperators(t *testing.T) {
	tests := []struct {
		in       string
		expected ScalarFilter
	}{
		{in: "count() > 1", expected: newScalarFilter(opGreater, newAggregate(aggregateCount, nil), newStaticInt(1))},
		{in: "max(a) > 1", expected: newScalarFilter(opGreater, newAggregate(aggregateMax, newStaticIdentifier("a")), newStaticInt(1))},
		{in: "min(1) > 1", expected: newScalarFilter(opGreater, newAggregate(aggregateMin, newStaticInt(1)), newStaticInt(1))},
		{in: "sum(true) > 1", expected: newScalarFilter(opGreater, newAggregate(aggregateSum, newStaticBool(true)), newStaticInt(1))},
		{in: "avg(`c`) > 1", expected: newScalarFilter(opGreater, newAggregate(aggregateAvg, newStaticString("c")), newStaticInt(1))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestSpansetFilterErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "wharblgarbl", err: newParseError("syntax error: unexpected IDENTIFIER", 1, 1)},
		{in: "{ 2 <> 3}", err: newParseError("syntax error: unexpected >", 1, 6)},
		{in: "{ 2 = b ", err: newParseError("syntax error: unexpected $end", 1, 9)},
		{in: "{ + }", err: newParseError("syntax error: unexpected +", 1, 3)},
		{in: "{ foo.3 }", err: newParseError("syntax error: unexpected FLOAT", 1, 6)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestSpansetFilterOperatorPrecedence(t *testing.T) {
	tests := []struct {
		in       string
		expected FieldExpression
	}{
		{
			in: "{ a * b + c }",
			expected: BinaryOperation{
				op: opAdd,
				lhs: BinaryOperation{
					op:  opMult,
					lhs: newStaticIdentifier("a"),
					rhs: newStaticIdentifier("b"),
				},
				rhs: newStaticIdentifier("c"),
			},
		},
		{
			in: "{ a + b * c }",
			expected: BinaryOperation{
				op:  opAdd,
				lhs: newStaticIdentifier("a"),
				rhs: BinaryOperation{
					op:  opMult,
					lhs: newStaticIdentifier("b"),
					rhs: newStaticIdentifier("c"),
				},
			},
		},
		{
			in: "{ ( a + b ) * c }",
			expected: BinaryOperation{
				op: opMult,
				lhs: BinaryOperation{
					op:  opAdd,
					lhs: newStaticIdentifier("a"),
					rhs: newStaticIdentifier("b"),
				},
				rhs: newStaticIdentifier("c"),
			},
		},
		{
			in: "{ a + b ^ c }",
			expected: BinaryOperation{
				op:  opAdd,
				lhs: newStaticIdentifier("a"),
				rhs: BinaryOperation{
					op:  opPower,
					lhs: newStaticIdentifier("b"),
					rhs: newStaticIdentifier("c"),
				},
			},
		},
		{
			in: "{ a = b + c }",
			expected: BinaryOperation{
				op:  opEqual,
				lhs: newStaticIdentifier("a"),
				rhs: BinaryOperation{
					op:  opAdd,
					lhs: newStaticIdentifier("b"),
					rhs: newStaticIdentifier("c"),
				},
			},
		},
		{
			in: "{ a + b = c }",
			expected: BinaryOperation{
				op: opEqual,
				lhs: BinaryOperation{
					op:  opAdd,
					lhs: newStaticIdentifier("a"),
					rhs: newStaticIdentifier("b"),
				},
				rhs: newStaticIdentifier("c"),
			},
		},
		{
			in: "{ c - -a + b }",
			expected: BinaryOperation{
				op: opAdd,
				lhs: BinaryOperation{
					op:  opSub,
					lhs: newStaticIdentifier("c"),
					rhs: UnaryOperation{
						op: opSub,
						e:  newStaticIdentifier("a"),
					},
				},
				rhs: newStaticIdentifier("b"),
			},
		},
		{
			in: "{ c - -( a + b ) }",
			expected: BinaryOperation{
				op:  opSub,
				lhs: newStaticIdentifier("c"),
				rhs: UnaryOperation{
					op: opSub,
					e: BinaryOperation{
						op:  opAdd,
						lhs: newStaticIdentifier("a"),
						rhs: newStaticIdentifier("b"),
					},
				},
			},
		},
		{
			in: "{ a && b = c }",
			expected: BinaryOperation{
				op: opEqual,
				lhs: BinaryOperation{
					op:  opAnd,
					lhs: newStaticIdentifier("a"),
					rhs: newStaticIdentifier("b"),
				},
				rhs: newStaticIdentifier("c"),
			},
		},
		{
			in: "{ a = b && c }",
			expected: BinaryOperation{
				op:  opEqual,
				lhs: newStaticIdentifier("a"),
				rhs: BinaryOperation{
					op:  opAnd,
					lhs: newStaticIdentifier("b"),
					rhs: newStaticIdentifier("c"),
				},
			},
		},
		{
			in: "{ a = !b && c }",
			expected: BinaryOperation{
				op:  opEqual,
				lhs: newStaticIdentifier("a"),
				rhs: BinaryOperation{
					op: opAnd,
					lhs: UnaryOperation{
						op: opNot,
						e:  newStaticIdentifier("b"),
					},
					rhs: newStaticIdentifier("c"),
				},
			},
		},
		{
			in: "{ a = !( b && c ) }",
			expected: BinaryOperation{
				op:  opEqual,
				lhs: newStaticIdentifier("a"),
				rhs: UnaryOperation{
					op: opNot,
					e: BinaryOperation{
						op:  opAnd,
						lhs: newStaticIdentifier("b"),
						rhs: newStaticIdentifier("c"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)
		})
	}
}

func TestSpansetFilterStatics(t *testing.T) {
	tests := []struct {
		in       string
		expected FieldExpression
	}{
		{in: "{ true }", expected: newStaticBool(true)},
		{in: "{ false }", expected: newStaticBool(false)},
		{in: `{ "true" }`, expected: newStaticString("true")},
		{in: `{ "true\"" }`, expected: newStaticString("true\"")},
		{in: "{ `foo` }", expected: newStaticString("foo")},
		{in: "{ foo }", expected: newStaticIdentifier("foo")},
		{in: "{ foo.bar }", expected: newStaticIdentifier("foo.bar")},
		{in: "{ foo.bar.baz }", expected: newStaticIdentifier("foo.bar.baz")},
		{in: "{ 4321 }", expected: newStaticInt(4321)},
		{in: "{ 1.234 }", expected: newStaticFloat(1.234)},
		{in: "{ nil }", expected: newStaticNil()},
		{in: "{ 3h }", expected: newStaticDuration(3 * time.Hour)},
		{in: "{ }", expected: newStaticBool(true)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)
		})
	}
}

func TestSpansetFilterOperators(t *testing.T) {
	tests := []struct {
		in       string
		err      error
		expected FieldExpression
	}{
		{in: "{ a + b }", expected: newBinaryOperation(opAdd, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a - b }", expected: newBinaryOperation(opSub, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a / b }", expected: newBinaryOperation(opDiv, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a % b }", expected: newBinaryOperation(opMod, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a * b }", expected: newBinaryOperation(opMult, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a = b }", expected: newBinaryOperation(opEqual, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a != b }", expected: newBinaryOperation(opNotEqual, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a =~ b }", expected: newBinaryOperation(opRegex, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a !~ b }", expected: newBinaryOperation(opNotRegex, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a > b }", expected: newBinaryOperation(opGreater, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a >= b }", expected: newBinaryOperation(opGreaterEqual, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a < b }", expected: newBinaryOperation(opLess, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a <= b }", expected: newBinaryOperation(opLessEqual, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a ^ b }", expected: newBinaryOperation(opPower, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a && b }", expected: newBinaryOperation(opAnd, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ a || b }", expected: newBinaryOperation(opOr, newStaticIdentifier("a"), newStaticIdentifier("b"))},
		{in: "{ !b }", expected: newUnaryOperation(opNot, newStaticIdentifier("b"))},
		{in: "{ -b }", expected: newUnaryOperation(opSub, newStaticIdentifier("b"))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)
		})
	}
}
