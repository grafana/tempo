package traceql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPipelineErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "{ .a } | { .b", err: newParseError("syntax error: unexpected $end", 1, 14)},
		{in: "{ .a | .b }", err: newParseError("syntax error: unexpected |", 1, 6)},
		{in: "({ .a } | { .b }", err: newParseError("syntax error: unexpected $end, expecting ) or |", 1, 17)},
		{in: "({ .a } | { .b }) + ({ .a } | { .b })", err: newParseError("syntax error: unexpected +", 1, 19)},
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
			in: "({ .a } | { .b }) > ({ .a } | { .b }) && ({ .a } | { .b })",
			expected: newSpansetOperation(opSpansetChild,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
				),
				newSpansetOperation(opSpansetAnd,
					newPipeline(
						newSpansetFilter(newAttribute("a")),
						newSpansetFilter(newAttribute("b")),
					),
					newPipeline(
						newSpansetFilter(newAttribute("a")),
						newSpansetFilter(newAttribute("b")),
					),
				),
			),
		},
		{
			in: "(({ .a } | { .b }) > ({ .a } | { .b })) && ({ .a } | { .b })",
			expected: newSpansetOperation(opSpansetAnd,
				newSpansetOperation(opSpansetChild,
					newPipeline(
						newSpansetFilter(newAttribute("a")),
						newSpansetFilter(newAttribute("b")),
					),
					newPipeline(
						newSpansetFilter(newAttribute("a")),
						newSpansetFilter(newAttribute("b")),
					),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
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
			in: "({ .a } | { .b }) > ({ .a } | { .b })",
			expected: newSpansetOperation(opSpansetChild,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
				),
			),
		},
		{
			in: "({ .a } | { .b }) && ({ .a } | { .b })",
			expected: newSpansetOperation(opSpansetAnd,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
				),
			),
		},
		{
			in: "({ .a } | { .b }) >> ({ .a } | { .b })",
			expected: newSpansetOperation(opSpansetDescendant,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
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
			in: "({ .a } | count()) = ({ .a } | count())",
			expected: newScalarFilter(opEqual,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ .a } | count()) != ({ .a } | count())",
			expected: newScalarFilter(opNotEqual,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ .a } | count()) < ({ .a } | count())",
			expected: newScalarFilter(opLess,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ .a } | count()) <= ({ .a } | count())",
			expected: newScalarFilter(opLessEqual,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ .a } | count()) >= ({ .a } | count())",
			expected: newScalarFilter(opGreaterEqual,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(newAttribute("a")),
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
			in: "{ .a } | { .b }",
			expected: newPipeline(
				newSpansetFilter(newAttribute("a")),
				newSpansetFilter(newAttribute("b")),
			),
		},
		{
			in: "{ .a } | count() > 1",
			expected: newPipeline(
				newSpansetFilter(newAttribute("a")),
				newScalarFilter(opGreater, newAggregate(aggregateCount, nil), newStaticInt(1)),
			),
		},
		{
			in: "{ .a } | by(.namespace) | coalesce() | avg(duration) = 1s ",
			expected: newPipeline(
				newSpansetFilter(newAttribute("a")),
				newGroupOperation(newAttribute("namespace")),
				newCoalesceOperation(),
				newScalarFilter(opEqual, newAggregate(aggregateAvg, newIntrinsic(intrinsicDuration)), newStaticDuration(time.Second)),
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
		{in: "by(.a) && { .b }", err: newParseError("syntax error: unexpected &&", 0, 8)},
		{in: "by()", err: newParseError("syntax error: unexpected )", 1, 4)},
		{in: "coalesce()", err: newParseError("syntax error: unexpected coalesce", 1, 1)},
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
		{in: "by(.a) | coalesce()", expected: newPipeline(newGroupOperation(newAttribute("a")), newCoalesceOperation())},
		{in: "by(.a + .b)", expected: newPipeline(newGroupOperation(newBinaryOperation(opAdd, newAttribute("a"), newAttribute("b"))))},
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
			expected: newSpansetOperation(opSpansetDescendant,
				newSpansetOperation(opSpansetAnd, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false))),
				newSpansetFilter(newStaticString("a")),
			),
		},
		{
			in: "{ true } >> { false } && { `a` }",
			expected: newSpansetOperation(opSpansetDescendant,
				newSpansetFilter(newStaticBool(true)),
				newSpansetOperation(opSpansetAnd, newSpansetFilter(newStaticBool(false)), newSpansetFilter(newStaticString("a"))),
			),
		},
		{
			in: "({ true } >> { false }) && { `a` }",
			expected: newSpansetOperation(opSpansetAnd,
				newSpansetOperation(opSpansetDescendant, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false))),
				newSpansetFilter(newStaticString("a")),
			),
		},
		{
			in: "{ true } >> { false } ~ { `a` }",
			expected: newSpansetOperation(opSpansetSibling,
				newSpansetOperation(opSpansetDescendant, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false))),
				newSpansetFilter(newStaticString("a")),
			),
		},
		{
			in: "{ true } ~ { false } >> { `a` }",
			expected: newSpansetOperation(opSpansetDescendant,
				newSpansetOperation(opSpansetSibling, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false))),
				newSpansetFilter(newStaticString("a")),
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
		{in: "(avg(.foo) > count()) + sum(.bar)", err: newParseError("syntax error: unexpected +", 1, 23)},
		{in: "count(", err: newParseError("syntax error: unexpected $end, expecting )", 1, 7)},
		{in: "{ .a } | 3 = 3", err: newParseError("syntax error: unexpected INTEGER", 1, 14)},
		{in: "count(avg)", err: newParseError("syntax error: unexpected avg, expecting )", 1, 7)},
		{in: "count(.thing)", err: newParseError("syntax error: unexpected ., expecting )", 1, 7)},
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
			in: "avg(.foo) > count() + sum(.bar)",
			expected: newScalarFilter(opGreater,
				newAggregate(aggregateAvg, newAttribute("foo")),
				newScalarOperation(opAdd,
					newAggregate(aggregateCount, nil),
					newAggregate(aggregateSum, newAttribute("bar")),
				),
			),
		},
		{
			in: "avg(.foo) + count() > sum(.bar)",
			expected: newScalarFilter(opGreater,
				newScalarOperation(opAdd,
					newAggregate(aggregateAvg, newAttribute("foo")),
					newAggregate(aggregateCount, nil),
				),
				newAggregate(aggregateSum, newAttribute("bar")),
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
		{in: "max(.a) > 1", expected: newScalarFilter(opGreater, newAggregate(aggregateMax, newAttribute("a")), newStaticInt(1))},
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
		{in: "{ 2 = .b ", err: newParseError("syntax error: unexpected $end", 1, 10)},
		{in: "{ + }", err: newParseError("syntax error: unexpected +", 1, 3)},
		{in: "{ .foo.3 }", err: newParseError("syntax error: unexpected FLOAT", 1, 7)},
		{in: "{ parent. }", err: newParseError("syntax error: unexpected $end", 1, 12)},
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
			in: "{ .a * .b + .c }",
			expected: newBinaryOperation(opAdd,
				newBinaryOperation(opMult, newAttribute("a"), newAttribute("b")),
				newAttribute("c")),
		},
		{
			in: "{ .a + .b * .c }",
			expected: newBinaryOperation(opAdd,
				newAttribute("a"),
				newBinaryOperation(opMult, newAttribute("b"), newAttribute("c"))),
		},
		{
			in: "{ ( .a + .b ) * .c }",
			expected: newBinaryOperation(opMult,
				newBinaryOperation(opAdd, newAttribute("a"), newAttribute("b")),
				newAttribute("c")),
		},
		{
			in: "{ .a + .b ^ .c }",
			expected: newBinaryOperation(opAdd,
				newAttribute("a"),
				newBinaryOperation(opPower, newAttribute("b"), newAttribute("c"))),
		},
		{
			in: "{ .a = .b + .c }",
			expected: newBinaryOperation(opEqual,
				newAttribute("a"),
				newBinaryOperation(opAdd, newAttribute("b"), newAttribute("c"))),
		},
		{
			in: "{ .a + .b = .c }",
			expected: newBinaryOperation(opEqual,
				newBinaryOperation(opAdd, newAttribute("a"), newAttribute("b")),
				newAttribute("c")),
		},
		{
			in: "{ .c - -.a + .b }",
			expected: newBinaryOperation(opAdd,
				newBinaryOperation(opSub, newAttribute("c"), newUnaryOperation(opSub, newAttribute("a"))),
				newAttribute("b")),
		},
		{
			in: "{ .c - -( .a + .b ) }",
			expected: newBinaryOperation(opSub,
				newAttribute("c"),
				newUnaryOperation(opSub, newBinaryOperation(opAdd, newAttribute("a"), newAttribute("b")))),
		},
		{
			in: "{ .a && .b = .c }",
			expected: newBinaryOperation(opEqual,
				newBinaryOperation(opAnd, newAttribute("a"), newAttribute("b")),
				newAttribute("c")),
		},
		{
			in: "{ .a = .b && .c }",
			expected: newBinaryOperation(opEqual,
				newAttribute("a"),
				newBinaryOperation(opAnd, newAttribute("b"), newAttribute("c"))),
		},
		{
			in: "{ .a = !.b && .c }",
			expected: newBinaryOperation(opEqual,
				newAttribute("a"),
				newBinaryOperation(opAnd, newUnaryOperation(opNot, newAttribute("b")), newAttribute("c"))),
		},
		{
			in: "{ .a = !( .b && .c ) }",
			expected: newBinaryOperation(opEqual,
				newAttribute("a"),
				newUnaryOperation(opNot, newBinaryOperation(opAnd, newAttribute("b"), newAttribute("c")))),
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
		{in: "{ .max }", expected: newAttribute("max")},
		{in: "{ .status }", expected: newAttribute("status")},
		{in: "{ .foo }", expected: newAttribute("foo")},
		{in: "{ .foo.bar }", expected: newAttribute("foo.bar")},
		{in: "{ .foo.bar.baz }", expected: newAttribute("foo.bar.baz")},
		{in: "{ parent.foo.bar.baz }", expected: newScopedAttribute(attributeScopeParent, "foo.bar.baz")},
		{in: "{ resource.foo.bar.baz }", expected: newScopedAttribute(attributeScopeResource, "foo.bar.baz")},
		{in: "{ span.foo.bar.baz }", expected: newScopedAttribute(attributeScopeSpan, "foo.bar.baz")},
		{in: "{ start }", expected: newIntrinsic(intrinsicStart)},
		{in: "{ end }", expected: newIntrinsic(intrinsicEnd)},
		{in: "{ duration }", expected: newIntrinsic(intrinsicDuration)},
		{in: "{ childCount }", expected: newIntrinsic(intrinsicChildCount)},
		{in: "{ name }", expected: newIntrinsic(intrinsicName)},
		{in: "{ parent }", expected: newIntrinsic(intrinsicParent)},
		{in: "{ status }", expected: newIntrinsic(intrinsicStatus)}, // jpe new tests for attributes
		{in: "{ 4321 }", expected: newStaticInt(4321)},
		{in: "{ 1.234 }", expected: newStaticFloat(1.234)},
		{in: "{ nil }", expected: newStaticNil()},
		{in: "{ 3h }", expected: newStaticDuration(3 * time.Hour)},
		{in: "{ error }", expected: newStaticStatus(statusError)},
		{in: "{ ok }", expected: newStaticStatus(statusOk)},
		{in: "{ unset }", expected: newStaticStatus(statusUnset)},
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
		{in: "{ .a + .b }", expected: newBinaryOperation(opAdd, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a - .b }", expected: newBinaryOperation(opSub, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a / .b }", expected: newBinaryOperation(opDiv, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a % .b }", expected: newBinaryOperation(opMod, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a * .b }", expected: newBinaryOperation(opMult, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a = .b }", expected: newBinaryOperation(opEqual, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a != .b }", expected: newBinaryOperation(opNotEqual, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a =~ .b }", expected: newBinaryOperation(opRegex, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a !~ .b }", expected: newBinaryOperation(opNotRegex, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a > .b }", expected: newBinaryOperation(opGreater, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a >= .b }", expected: newBinaryOperation(opGreaterEqual, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a < .b }", expected: newBinaryOperation(opLess, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a <= .b }", expected: newBinaryOperation(opLessEqual, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a ^ .b }", expected: newBinaryOperation(opPower, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a && .b }", expected: newBinaryOperation(opAnd, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a || .b }", expected: newBinaryOperation(opOr, newAttribute("a"), newAttribute("b"))},
		{in: "{ !.b }", expected: newUnaryOperation(opNot, newAttribute("b"))},
		{in: "{ -.b }", expected: newUnaryOperation(opSub, newAttribute("b"))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)
		})
	}
}
