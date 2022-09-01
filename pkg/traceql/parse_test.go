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
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetOperation(OpSpansetChild,
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
		{
			in: "({ .a } | { .b }) > (({ .a } | { .b }) && ({ .a } | { .b }))",
			expected: newSpansetOperation(OpSpansetChild,
				newPipeline(
					newSpansetFilter(newAttribute("a")),
					newSpansetFilter(newAttribute("b")),
				),
				newSpansetOperation(OpSpansetAnd,
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
			expected: newSpansetOperation(OpSpansetChild,
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
			expected: newSpansetOperation(OpSpansetAnd,
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
			expected: newSpansetOperation(OpSpansetDescendant,
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
			expected: newScalarFilter(OpEqual,
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
			expected: newScalarFilter(OpNotEqual,
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
			expected: newScalarFilter(OpLess,
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
			expected: newScalarFilter(OpLessEqual,
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
			expected: newScalarFilter(OpGreaterEqual,
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
				newScalarFilter(OpGreater, newAggregate(aggregateCount, nil), newStaticInt(1)),
			),
		},
		{
			in: "{ .a } | by(.namespace) | coalesce() | avg(duration) = 1s ",
			expected: newPipeline(
				newSpansetFilter(newAttribute("a")),
				newGroupOperation(newAttribute("namespace")),
				newCoalesceOperation(),
				newScalarFilter(OpEqual, newAggregate(aggregateAvg, newIntrinsic(IntrinsicDuration)), newStaticDuration(time.Second)),
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
		{in: "by(.a + .b)", expected: newPipeline(newGroupOperation(newBinaryOperation(OpAdd, newAttribute("a"), newAttribute("b"))))},
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
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetFilter(newStaticBool(true)),
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(newStaticBool(false)), newSpansetFilter(newStaticString("a"))),
			),
		},
		{
			in: "{ true } >> { false } && { `a` }",
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false))),
				newSpansetFilter(newStaticString("a")),
			),
		},
		{
			in: "({ true } >> { false }) && { `a` }",
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false))),
				newSpansetFilter(newStaticString("a")),
			),
		},
		{
			in: "{ true } >> { false } ~ { `a` }",
			expected: newSpansetOperation(OpSpansetSibling,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false))),
				newSpansetFilter(newStaticString("a")),
			),
		},
		{
			in: "{ true } ~ { false } >> { `a` }",
			expected: newSpansetOperation(OpSpansetDescendant,
				newSpansetOperation(OpSpansetSibling, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false))),
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
		{in: "{ true } && { false }", expected: newSpansetOperation(OpSpansetAnd, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		{in: "{ true } > { false }", expected: newSpansetOperation(OpSpansetChild, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		{in: "{ true } >> { false }", expected: newSpansetOperation(OpSpansetDescendant, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		{in: "{ true } || { false }", expected: newSpansetOperation(OpSpansetUnion, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		{in: "{ true } ~ { false }", expected: newSpansetOperation(OpSpansetSibling, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
		// this test was added to highlight the one shift/reduce conflict in the grammar. this could also be parsed as two spanset pipelines &&ed together.
		{in: "({ true }) && ({ false })", expected: newSpansetOperation(OpSpansetAnd, newSpansetFilter(newStaticBool(true)), newSpansetFilter(newStaticBool(false)))},
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
			expected: newScalarFilter(OpGreater,
				newAggregate(aggregateAvg, newAttribute("foo")),
				newScalarOperation(OpAdd,
					newAggregate(aggregateCount, nil),
					newAggregate(aggregateSum, newAttribute("bar")),
				),
			),
		},
		{
			in: "avg(.foo) + count() > sum(.bar)",
			expected: newScalarFilter(OpGreater,
				newScalarOperation(OpAdd,
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
		{in: "count() > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateCount, nil), newStaticInt(1))},
		{in: "max(.a) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateMax, newAttribute("a")), newStaticInt(1))},
		{in: "min(1) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateMin, newStaticInt(1)), newStaticInt(1))},
		{in: "sum(true) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateSum, newStaticBool(true)), newStaticInt(1))},
		{in: "avg(`c`) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateAvg, newStaticString("c")), newStaticInt(1))},
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
			expected: newBinaryOperation(OpAdd,
				newBinaryOperation(OpMult, newAttribute("a"), newAttribute("b")),
				newAttribute("c")),
		},
		{
			in: "{ .a + .b * .c }",
			expected: newBinaryOperation(OpAdd,
				newAttribute("a"),
				newBinaryOperation(OpMult, newAttribute("b"), newAttribute("c"))),
		},
		{
			in: "{ ( .a + .b ) * .c }",
			expected: newBinaryOperation(OpMult,
				newBinaryOperation(OpAdd, newAttribute("a"), newAttribute("b")),
				newAttribute("c")),
		},
		{
			in: "{ .a + .b ^ .c }",
			expected: newBinaryOperation(OpAdd,
				newAttribute("a"),
				newBinaryOperation(OpPower, newAttribute("b"), newAttribute("c"))),
		},
		{
			in: "{ .a = .b + .c }",
			expected: newBinaryOperation(OpEqual,
				newAttribute("a"),
				newBinaryOperation(OpAdd, newAttribute("b"), newAttribute("c"))),
		},
		{
			in: "{ .a + .b = .c }",
			expected: newBinaryOperation(OpEqual,
				newBinaryOperation(OpAdd, newAttribute("a"), newAttribute("b")),
				newAttribute("c")),
		},
		{
			in: "{ .c - -.a + .b }",
			expected: newBinaryOperation(OpAdd,
				newBinaryOperation(OpSub, newAttribute("c"), newUnaryOperation(OpSub, newAttribute("a"))),
				newAttribute("b")),
		},
		{
			in: "{ .c - -( .a + .b ) }",
			expected: newBinaryOperation(OpSub,
				newAttribute("c"),
				newUnaryOperation(OpSub, newBinaryOperation(OpAdd, newAttribute("a"), newAttribute("b")))),
		},
		{
			in: "{ .a && .b = .c }",
			expected: newBinaryOperation(OpAnd,
				newAttribute("a"),
				newBinaryOperation(OpEqual, newAttribute("b"), newAttribute("c"))),
		},
		{
			in: "{ .a = .b && .c }",
			expected: newBinaryOperation(OpAnd,
				newBinaryOperation(OpEqual, newAttribute("a"), newAttribute("b")),
				newAttribute("c")),
		},
		{
			in: "{ .a = !.b && .c }",
			expected: newBinaryOperation(OpAnd,
				newBinaryOperation(OpEqual, newAttribute("a"), newUnaryOperation(OpNot, newAttribute("b"))),
				newAttribute("c")),
		},
		{
			in: "{ .a = !( .b && .c ) }",
			expected: newBinaryOperation(OpEqual,
				newAttribute("a"),
				newUnaryOperation(OpNot, newBinaryOperation(OpAnd, newAttribute("b"), newAttribute("c")))),
		},
		{
			in: "{ .a = .b || .c = .d}",
			expected: newBinaryOperation(OpOr,
				newBinaryOperation(OpEqual, newAttribute("a"), newAttribute("b")),
				newBinaryOperation(OpEqual, newAttribute("c"), newAttribute("d"))),
		},
		{
			in: "{ !.a = .b }",
			expected: newBinaryOperation(OpEqual,
				newUnaryOperation(OpNot, newAttribute("a")),
				newAttribute("b")),
		},
		{
			in: "{ !(.a = .b) }",
			expected: newUnaryOperation(OpNot, newBinaryOperation(OpEqual,
				newAttribute("a"),
				newAttribute("b"))),
		},
		{
			in: "{ -.a = .b }",
			expected: newBinaryOperation(OpEqual,
				newUnaryOperation(OpSub, newAttribute("a")),
				newAttribute("b")),
		},
		{
			in: "{ -(.a = .b) }",
			expected: newUnaryOperation(OpSub, newBinaryOperation(OpEqual,
				newAttribute("a"),
				newAttribute("b"))),
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
		{in: "{ .foo }", expected: newAttribute("foo")},
		{in: "{ duration }", expected: newIntrinsic(IntrinsicDuration)},
		{in: "{ childCount }", expected: newIntrinsic(IntrinsicChildCount)},
		{in: "{ name }", expected: newIntrinsic(IntrinsicName)},
		{in: "{ parent }", expected: newIntrinsic(IntrinsicParent)},
		{in: "{ status }", expected: newIntrinsic(IntrinsicStatus)},
		{in: "{ 4321 }", expected: newStaticInt(4321)},
		{in: "{ 1.234 }", expected: newStaticFloat(1.234)},
		{in: "{ nil }", expected: newStaticNil()},
		{in: "{ 3h }", expected: newStaticDuration(3 * time.Hour)},
		{in: "{ error }", expected: newStaticStatus(StatusError)},
		{in: "{ ok }", expected: newStaticStatus(StatusOk)},
		{in: "{ unset }", expected: newStaticStatus(StatusUnset)},
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
		{in: "{ .a + .b }", expected: newBinaryOperation(OpAdd, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a - .b }", expected: newBinaryOperation(OpSub, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a / .b }", expected: newBinaryOperation(OpDiv, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a % .b }", expected: newBinaryOperation(OpMod, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a * .b }", expected: newBinaryOperation(OpMult, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a = .b }", expected: newBinaryOperation(OpEqual, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a != .b }", expected: newBinaryOperation(OpNotEqual, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a =~ .b }", expected: newBinaryOperation(OpRegex, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a !~ .b }", expected: newBinaryOperation(OpNotRegex, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a > .b }", expected: newBinaryOperation(OpGreater, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a >= .b }", expected: newBinaryOperation(OpGreaterEqual, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a < .b }", expected: newBinaryOperation(OpLess, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a <= .b }", expected: newBinaryOperation(OpLessEqual, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a ^ .b }", expected: newBinaryOperation(OpPower, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a && .b }", expected: newBinaryOperation(OpAnd, newAttribute("a"), newAttribute("b"))},
		{in: "{ .a || .b }", expected: newBinaryOperation(OpOr, newAttribute("a"), newAttribute("b"))},
		{in: "{ !.b }", expected: newUnaryOperation(OpNot, newAttribute("b"))},
		{in: "{ -.b }", expected: newUnaryOperation(OpSub, newAttribute("b"))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)
		})
	}
}

func TestAttributeNameErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "{ . foo }", err: newParseError("syntax error: unexpected END_ATTRIBUTE, expecting IDENTIFIER", 1, 3)},
		{in: "{ .foo .bar }", err: newParseError("syntax error: unexpected .", 1, 8)},
		{in: "{ parent. }", err: newParseError("syntax error: unexpected END_ATTRIBUTE, expecting IDENTIFIER or resource. or span.", 0, 3)},
		{in: ".3foo", err: newParseError("syntax error: unexpected IDENTIFIER", 1, 3)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestAttributes(t *testing.T) {
	tests := []struct {
		in       string
		expected FieldExpression
	}{
		{in: "duration", expected: newIntrinsic(IntrinsicDuration)},
		{in: ".foo", expected: newAttribute("foo")},
		{in: ".max", expected: newAttribute("max")},
		{in: ".status", expected: newAttribute("status")},
		{in: ".foo.bar", expected: newAttribute("foo.bar")},
		{in: ".foo.bar.baz", expected: newAttribute("foo.bar.baz")},
		{in: ".foo.3", expected: newAttribute("foo.3")},
		{in: ".foo3", expected: newAttribute("foo3")},
		{in: ".http_status", expected: newAttribute("http_status")},
		{in: ".http-status", expected: newAttribute("http-status")},
		{in: ".http+", expected: newAttribute("http+")},
		{in: ".😝", expected: newAttribute("😝")},
		{in: ".http-other", expected: newAttribute("http-other")},
		{in: "parent.duration", expected: newScopedAttribute(AttributeScopeNone, true, "duration")},
		{in: "parent.foo.bar.baz", expected: newScopedAttribute(AttributeScopeNone, true, "foo.bar.baz")},
		{in: "resource.foo.bar.baz", expected: newScopedAttribute(AttributeScopeResource, false, "foo.bar.baz")},
		{in: "span.foo.bar", expected: newScopedAttribute(AttributeScopeSpan, false, "foo.bar")},
		{in: "parent.resource.foo", expected: newScopedAttribute(AttributeScopeResource, true, "foo")},
		{in: "parent.span.foo", expected: newScopedAttribute(AttributeScopeSpan, true, "foo")},
		{in: "parent.resource.foo.bar.baz", expected: newScopedAttribute(AttributeScopeResource, true, "foo.bar.baz")},
		{in: "parent.span.foo.bar", expected: newScopedAttribute(AttributeScopeSpan, true, "foo.bar")},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			s := "{ " + tc.in + " }"
			actual, err := Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)

			s = "{" + tc.in + "}"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)

			s = "{ (" + tc.in + ") }"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)

			s = "{ " + tc.in + " + " + tc.in + " }"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(newSpansetFilter(newBinaryOperation(OpAdd, tc.expected, tc.expected)))}, actual)
		})
	}
}

func TestIntrinsics(t *testing.T) {
	tests := []struct {
		in       string
		expected Intrinsic
	}{
		{in: "duration", expected: IntrinsicDuration},
		{in: "childCount", expected: IntrinsicChildCount},
		{in: "name", expected: IntrinsicName},
		{in: "status", expected: IntrinsicStatus},
		{in: "parent", expected: IntrinsicParent},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			// as intrinsic e.g. duration
			s := "{ " + tc.in + " }"
			actual, err := Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: tc.expected,
				}))}, actual)

			// as attribute e.g .duration
			s = "{ ." + tc.in + "}"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: tc.expected,
				}))}, actual)

			// as span scoped attribute e.g span.duration
			s = "{ span." + tc.in + "}"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeSpan,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as resource scoped attribute e.g resource.duration
			s = "{ resource." + tc.in + "}"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeResource,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as parent scoped intrinsic e.g parent.duration
			s = "{ parent." + tc.in + "}"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: tc.expected,
				}))}, actual)

			// as nested parent scoped intrinsic e.g. parent.duration.foo
			s = "{ parent." + tc.in + ".foo }"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    true,
					Name:      tc.in + ".foo",
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as parent resource scoped attribute e.g. parent.resource.duration
			s = "{ parent.resource." + tc.in + "}"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeResource,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as parent span scoped attribute e.g. praent.span.duration
			s = "{ parent.span." + tc.in + "}"
			actual, err = Parse(s)

			assert.NoError(t, err)
			assert.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeSpan,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)
		})
	}
}
