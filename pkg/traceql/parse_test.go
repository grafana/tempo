package traceql

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

			require.Equal(t, tc.err, err)
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
						newSpansetFilter(NewAttribute("a")),
						newSpansetFilter(NewAttribute("b")),
					),
					newPipeline(
						newSpansetFilter(NewAttribute("a")),
						newSpansetFilter(NewAttribute("b")),
					),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
		},
		{
			in: "({ .a } | { .b }) > (({ .a } | { .b }) && ({ .a } | { .b }))",
			expected: newSpansetOperation(OpSpansetChild,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newSpansetOperation(OpSpansetAnd,
					newPipeline(
						newSpansetFilter(NewAttribute("a")),
						newSpansetFilter(NewAttribute("b")),
					),
					newPipeline(
						newSpansetFilter(NewAttribute("a")),
						newSpansetFilter(NewAttribute("b")),
					),
				),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
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
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
		},
		{
			in: "({ .a } | { .b }) && ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetAnd,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
		},
		{
			in: "({ .a } | { .b }) >> ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetDescendant,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
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
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ .a } | count()) != ({ .a } | count())",
			expected: newScalarFilter(OpNotEqual,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ .a } | count()) < ({ .a } | count())",
			expected: newScalarFilter(OpLess,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ .a } | count()) <= ({ .a } | count())",
			expected: newScalarFilter(OpLessEqual,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
		{
			in: "({ .a } | count()) >= ({ .a } | count())",
			expected: newScalarFilter(OpGreaterEqual,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newAggregate(aggregateCount, nil),
				),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
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
				newSpansetFilter(NewAttribute("a")),
				newSpansetFilter(NewAttribute("b")),
			),
		},
		{
			in: "{ .a } | count() > 1",
			expected: newPipeline(
				newSpansetFilter(NewAttribute("a")),
				newScalarFilter(OpGreater, newAggregate(aggregateCount, nil), NewStaticInt(1)),
			),
		},
		{
			in: "{ .a } | by(.namespace) | coalesce() | avg(duration) = 1s ",
			expected: newPipeline(
				newSpansetFilter(NewAttribute("a")),
				newGroupOperation(NewAttribute("namespace")),
				newCoalesceOperation(),
				newScalarFilter(OpEqual, newAggregate(aggregateAvg, NewIntrinsic(IntrinsicDuration)), NewStaticDuration(time.Second)),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{tc.expected}, actual)
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

			require.Equal(t, tc.err, err)
		})
	}
}

func TestGroupCoalesceOperation(t *testing.T) {
	tests := []struct {
		in       string
		expected Pipeline
	}{
		{in: "by(.a) | coalesce()", expected: newPipeline(newGroupOperation(NewAttribute("a")), newCoalesceOperation())},
		{in: "by(.a + .b)", expected: newPipeline(newGroupOperation(newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b"))))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{tc.expected}, actual)
		})
	}
}

func TestSelectErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "select(.a) && { .b }", err: newParseError("syntax error: unexpected &&", 0, 12)},
		{in: "select()", err: newParseError("syntax error: unexpected )", 1, 8)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			require.Equal(t, tc.err, err)
		})
	}
}

func TestSelectOperation(t *testing.T) {
	tests := []struct {
		in       string
		expected Pipeline
	}{
		{in: "select(.a)", expected: newPipeline(newSelectOperation([]FieldExpression{NewAttribute("a")}))},
		{in: "select(.a,.b)", expected: newPipeline(newSelectOperation([]FieldExpression{NewAttribute("a"), NewAttribute("b")}))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{tc.expected}, actual)
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

			require.Equal(t, tc.err, err)
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
				newSpansetFilter(NewStaticBool(true)),
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(false)), newSpansetFilter(NewStaticString("a"))),
			),
		},
		{
			in: "{ true } >> { false } && { `a` }",
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))),
				newSpansetFilter(NewStaticString("a")),
			),
		},
		{
			in: "({ true } >> { false }) && { `a` }",
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))),
				newSpansetFilter(NewStaticString("a")),
			),
		},
		{
			in: "{ true } >> { false } ~ { `a` }",
			expected: newSpansetOperation(OpSpansetSibling,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))),
				newSpansetFilter(NewStaticString("a")),
			),
		},
		{
			in: "{ true } ~ { false } >> { `a` }",
			expected: newSpansetOperation(OpSpansetDescendant,
				newSpansetOperation(OpSpansetSibling, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))),
				newSpansetFilter(NewStaticString("a")),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestSpansetExpressionOperators(t *testing.T) {
	tests := []struct {
		in       string
		expected SpansetOperation
	}{
		{in: "{ true } && { false }", expected: newSpansetOperation(OpSpansetAnd, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false)))},
		{in: "{ true } > { false }", expected: newSpansetOperation(OpSpansetChild, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false)))},
		{in: "{ true } >> { false }", expected: newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false)))},
		{in: "{ true } || { false }", expected: newSpansetOperation(OpSpansetUnion, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false)))},
		{in: "{ true } ~ { false }", expected: newSpansetOperation(OpSpansetSibling, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false)))},
		// this test was added to highlight the one shift/reduce conflict in the grammar. this could also be parsed as two spanset pipelines &&ed together.
		{in: "({ true }) && ({ false })", expected: newSpansetOperation(OpSpansetAnd, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false)))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
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

			require.Equal(t, tc.err, err)
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
				newAggregate(aggregateAvg, NewAttribute("foo")),
				newScalarOperation(OpAdd,
					newAggregate(aggregateCount, nil),
					newAggregate(aggregateSum, NewAttribute("bar")),
				),
			),
		},
		{
			in: "avg(.foo) + count() > sum(.bar)",
			expected: newScalarFilter(OpGreater,
				newScalarOperation(OpAdd,
					newAggregate(aggregateAvg, NewAttribute("foo")),
					newAggregate(aggregateCount, nil),
				),
				newAggregate(aggregateSum, NewAttribute("bar")),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
		})
	}
}

func TestScalarExpressionOperators(t *testing.T) {
	tests := []struct {
		in       string
		expected ScalarFilter
	}{
		{in: "count() > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateCount, nil), NewStaticInt(1))},
		{in: "max(.a) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateMax, NewAttribute("a")), NewStaticInt(1))},
		{in: "min(1) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateMin, NewStaticInt(1)), NewStaticInt(1))},
		{in: "sum(true) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateSum, NewStaticBool(true)), NewStaticInt(1))},
		{in: "avg(`c`) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateAvg, NewStaticString("c")), NewStaticInt(1))},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(tc.expected)}, actual)
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

			require.Equal(t, tc.err, err)
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
				newBinaryOperation(OpMult, NewAttribute("a"), NewAttribute("b")),
				NewAttribute("c")),
		},
		{
			in: "{ .a + .b * .c }",
			expected: newBinaryOperation(OpAdd,
				NewAttribute("a"),
				newBinaryOperation(OpMult, NewAttribute("b"), NewAttribute("c"))),
		},
		{
			in: "{ ( .a + .b ) * .c }",
			expected: newBinaryOperation(OpMult,
				newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b")),
				NewAttribute("c")),
		},
		{
			in: "{ .a + .b ^ .c }",
			expected: newBinaryOperation(OpAdd,
				NewAttribute("a"),
				newBinaryOperation(OpPower, NewAttribute("b"), NewAttribute("c"))),
		},
		{
			in: "{ .a = .b + .c }",
			expected: newBinaryOperation(OpEqual,
				NewAttribute("a"),
				newBinaryOperation(OpAdd, NewAttribute("b"), NewAttribute("c"))),
		},
		{
			in: "{ .a + .b = .c }",
			expected: newBinaryOperation(OpEqual,
				newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b")),
				NewAttribute("c")),
		},
		{
			in: "{ .c - -.a + .b }",
			expected: newBinaryOperation(OpAdd,
				newBinaryOperation(OpSub, NewAttribute("c"), newUnaryOperation(OpSub, NewAttribute("a"))),
				NewAttribute("b")),
		},
		{
			in: "{ .c - -( .a + .b ) }",
			expected: newBinaryOperation(OpSub,
				NewAttribute("c"),
				newUnaryOperation(OpSub, newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b")))),
		},
		{
			in: "{ .a && .b = .c }",
			expected: newBinaryOperation(OpAnd,
				NewAttribute("a"),
				newBinaryOperation(OpEqual, NewAttribute("b"), NewAttribute("c"))),
		},
		{
			in: "{ .a = .b && .c }",
			expected: newBinaryOperation(OpAnd,
				newBinaryOperation(OpEqual, NewAttribute("a"), NewAttribute("b")),
				NewAttribute("c")),
		},
		{
			in: "{ .a = !.b && .c }",
			expected: newBinaryOperation(OpAnd,
				newBinaryOperation(OpEqual, NewAttribute("a"), newUnaryOperation(OpNot, NewAttribute("b"))),
				NewAttribute("c")),
		},
		{
			in: "{ .a = !( .b && .c ) }",
			expected: newBinaryOperation(OpEqual,
				NewAttribute("a"),
				newUnaryOperation(OpNot, newBinaryOperation(OpAnd, NewAttribute("b"), NewAttribute("c")))),
		},
		{
			in: "{ .a = .b || .c = .d}",
			expected: newBinaryOperation(OpOr,
				newBinaryOperation(OpEqual, NewAttribute("a"), NewAttribute("b")),
				newBinaryOperation(OpEqual, NewAttribute("c"), NewAttribute("d"))),
		},
		{
			in: "{ !.a = .b }",
			expected: newBinaryOperation(OpEqual,
				newUnaryOperation(OpNot, NewAttribute("a")),
				NewAttribute("b")),
		},
		{
			in: "{ !(.a = .b) }",
			expected: newUnaryOperation(OpNot, newBinaryOperation(OpEqual,
				NewAttribute("a"),
				NewAttribute("b"))),
		},
		{
			in: "{ -.a = .b }",
			expected: newBinaryOperation(OpEqual,
				newUnaryOperation(OpSub, NewAttribute("a")),
				NewAttribute("b")),
		},
		{
			in: "{ -(.a = .b) }",
			expected: newUnaryOperation(OpSub, newBinaryOperation(OpEqual,
				NewAttribute("a"),
				NewAttribute("b"))),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)
		})
	}
}

func TestSpansetFilterStatics(t *testing.T) {
	tests := []struct {
		in       string
		expected FieldExpression
	}{
		{in: "{ true }", expected: NewStaticBool(true)},
		{in: "{ false }", expected: NewStaticBool(false)},
		{in: `{ "true" }`, expected: NewStaticString("true")},
		{in: `{ "true\"" }`, expected: NewStaticString("true\"")},
		{in: "{ `foo` }", expected: NewStaticString("foo")},
		{in: "{ .foo }", expected: NewAttribute("foo")},
		{in: "{ duration }", expected: NewIntrinsic(IntrinsicDuration)},
		{in: "{ childCount }", expected: NewIntrinsic(IntrinsicChildCount)},
		{in: "{ name }", expected: NewIntrinsic(IntrinsicName)},
		{in: "{ parent }", expected: NewIntrinsic(IntrinsicParent)},
		{in: "{ status }", expected: NewIntrinsic(IntrinsicStatus)},
		{in: "{ statusMessage }", expected: NewIntrinsic(IntrinsicStatusMessage)},
		{in: "{ 4321 }", expected: NewStaticInt(4321)},
		{in: "{ 1.234 }", expected: NewStaticFloat(1.234)},
		{in: "{ nil }", expected: NewStaticNil()},
		{in: "{ 3h }", expected: NewStaticDuration(3 * time.Hour)},
		{in: "{ 1.5m }", expected: NewStaticDuration(1*time.Minute + 30*time.Second)},
		{in: "{ error }", expected: NewStaticStatus(StatusError)},
		{in: "{ ok }", expected: NewStaticStatus(StatusOk)},
		{in: "{ unset }", expected: NewStaticStatus(StatusUnset)},
		{in: "{ unspecified }", expected: NewStaticKind(KindUnspecified)},
		{in: "{ internal }", expected: NewStaticKind(KindInternal)},
		{in: "{ client }", expected: NewStaticKind(KindClient)},
		{in: "{ server }", expected: NewStaticKind(KindServer)},
		{in: "{ producer }", expected: NewStaticKind(KindProducer)},
		{in: "{ consumer }", expected: NewStaticKind(KindConsumer)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)
		})
	}
}

func TestSpansetFilterOperators(t *testing.T) {
	tests := []struct {
		in                   string
		expected             FieldExpression
		alsoTestWithoutSpace bool
	}{
		{in: "{ .a + .b }", expected: newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b"))},
		{in: "{ .a - .b }", expected: newBinaryOperation(OpSub, NewAttribute("a"), NewAttribute("b"))},
		{in: "{ .a / .b }", expected: newBinaryOperation(OpDiv, NewAttribute("a"), NewAttribute("b"))},
		{in: "{ .a % .b }", expected: newBinaryOperation(OpMod, NewAttribute("a"), NewAttribute("b"))},
		{in: "{ .a * .b }", expected: newBinaryOperation(OpMult, NewAttribute("a"), NewAttribute("b"))},
		{in: "{ .a = .b }", expected: newBinaryOperation(OpEqual, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a != .b }", expected: newBinaryOperation(OpNotEqual, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a =~ .b }", expected: newBinaryOperation(OpRegex, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a !~ .b }", expected: newBinaryOperation(OpNotRegex, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a > .b }", expected: newBinaryOperation(OpGreater, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a >= .b }", expected: newBinaryOperation(OpGreaterEqual, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a < .b }", expected: newBinaryOperation(OpLess, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a <= .b }", expected: newBinaryOperation(OpLessEqual, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a ^ .b }", expected: newBinaryOperation(OpPower, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a && .b }", expected: newBinaryOperation(OpAnd, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ .a || .b }", expected: newBinaryOperation(OpOr, NewAttribute("a"), NewAttribute("b")), alsoTestWithoutSpace: true},
		{in: "{ !.b }", expected: newUnaryOperation(OpNot, NewAttribute("b"))},
		{in: "{ -.b }", expected: newUnaryOperation(OpSub, NewAttribute("b"))},

		// Against statics
		{in: "{ .a = `foo` }", expected: newBinaryOperation(OpEqual, NewAttribute("a"), NewStaticString("foo")), alsoTestWithoutSpace: true},
		{in: "{ .a = 3 }", expected: newBinaryOperation(OpEqual, NewAttribute("a"), NewStaticInt(3)), alsoTestWithoutSpace: true},
		{in: "{ .a = 3.0 }", expected: newBinaryOperation(OpEqual, NewAttribute("a"), NewStaticFloat(3)), alsoTestWithoutSpace: true},
		{in: "{ .a = true }", expected: newBinaryOperation(OpEqual, NewAttribute("a"), NewStaticBool(true)), alsoTestWithoutSpace: true},

		// existence
		{in: "{ .a != nil }", expected: newBinaryOperation(OpNotEqual, NewAttribute("a"), NewStaticNil()), alsoTestWithoutSpace: true},
		{in: "{ .a = nil }", expected: newBinaryOperation(OpEqual, NewAttribute("a"), NewStaticNil()), alsoTestWithoutSpace: true},
	}

	test := func(q string, expected FieldExpression) {
		actual, err := Parse(q)
		require.NoError(t, err, q)
		require.Equal(t, &RootExpr{newPipeline(newSpansetFilter(expected))}, actual, q)
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			test(tc.in, tc.expected)
			if tc.alsoTestWithoutSpace {
				test(strings.ReplaceAll(tc.in, " ", ""), tc.expected)
			}
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

			require.Equal(t, tc.err, err)
		})
	}
}

func TestAttributes(t *testing.T) {
	tests := []struct {
		in       string
		expected FieldExpression
	}{
		{in: "duration", expected: NewIntrinsic(IntrinsicDuration)},
		{in: "kind", expected: NewIntrinsic(IntrinsicKind)},
		{in: ".foo", expected: NewAttribute("foo")},
		{in: ".max", expected: NewAttribute("max")},
		{in: ".status", expected: NewAttribute("status")},
		{in: ".kind", expected: NewAttribute("kind")},
		{in: ".foo.bar", expected: NewAttribute("foo.bar")},
		{in: ".foo.bar.baz", expected: NewAttribute("foo.bar.baz")},
		{in: ".foo.3", expected: NewAttribute("foo.3")},
		{in: ".foo3", expected: NewAttribute("foo3")},
		{in: ".http_status", expected: NewAttribute("http_status")},
		{in: ".http-status", expected: NewAttribute("http-status")},
		{in: ".http+", expected: NewAttribute("http+")},
		{in: ".😝", expected: NewAttribute("😝")},
		{in: ".http-other", expected: NewAttribute("http-other")},
		{in: "parent.duration", expected: NewScopedAttribute(AttributeScopeNone, true, "duration")},
		{in: "parent.foo.bar.baz", expected: NewScopedAttribute(AttributeScopeNone, true, "foo.bar.baz")},
		{in: "resource.foo.bar.baz", expected: NewScopedAttribute(AttributeScopeResource, false, "foo.bar.baz")},
		{in: "span.foo.bar", expected: NewScopedAttribute(AttributeScopeSpan, false, "foo.bar")},
		{in: "parent.resource.foo", expected: NewScopedAttribute(AttributeScopeResource, true, "foo")},
		{in: "parent.span.foo", expected: NewScopedAttribute(AttributeScopeSpan, true, "foo")},
		{in: "parent.resource.foo.bar.baz", expected: NewScopedAttribute(AttributeScopeResource, true, "foo.bar.baz")},
		{in: "parent.span.foo.bar", expected: NewScopedAttribute(AttributeScopeSpan, true, "foo.bar")},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			s := "{ " + tc.in + " }"
			actual, err := Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)

			s = "{" + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)

			s = "{ (" + tc.in + ") }"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(newSpansetFilter(tc.expected))}, actual)

			s = "{ " + tc.in + " + " + tc.in + " }"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(newSpansetFilter(newBinaryOperation(OpAdd, tc.expected, tc.expected)))}, actual)
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
		{in: "statusMessage", expected: IntrinsicStatusMessage},
		{in: "kind", expected: IntrinsicKind},
		{in: "parent", expected: IntrinsicParent},
		{in: "traceDuration", expected: IntrinsicTraceDuration},
		{in: "rootServiceName", expected: IntrinsicTraceRootService},
		{in: "rootName", expected: IntrinsicTraceRootSpan},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			// as intrinsic e.g. duration
			s := "{ " + tc.in + " }"
			actual, err := Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: tc.expected,
				}))}, actual)

			// as attribute e.g .duration
			s = "{ ." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as span scoped attribute e.g span.duration
			s = "{ span." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeSpan,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as resource scoped attribute e.g resource.duration
			s = "{ resource." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeResource,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as parent scoped intrinsic e.g parent.duration
			s = "{ parent." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: tc.expected,
				}))}, actual)

			// as nested parent scoped intrinsic e.g. parent.duration.foo
			// this becomes lookup on attribute named "duration.foo"
			s = "{ parent." + tc.in + ".foo }"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    true,
					Name:      tc.in + ".foo",
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as parent resource scoped attribute e.g. parent.resource.duration
			s = "{ parent.resource." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeResource,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)

			// as parent span scoped attribute e.g. praent.span.duration
			s = "{ parent.span." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, &RootExpr{newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeSpan,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))}, actual)
		})
	}
}

func TestParseIdentifier(t *testing.T) {
	testCases := map[string]Attribute{
		"name":             NewIntrinsic(IntrinsicName),
		"status":           NewIntrinsic(IntrinsicStatus),
		"statusMessage":    NewIntrinsic(IntrinsicStatusMessage),
		"kind":             NewIntrinsic(IntrinsicKind),
		".name":            NewAttribute("name"),
		".status":          NewAttribute("status"),
		".foo.bar":         NewAttribute("foo.bar"),
		"resource.foo.bar": NewScopedAttribute(AttributeScopeResource, false, "foo.bar"),
		"span.foo.bar":     NewScopedAttribute(AttributeScopeSpan, false, "foo.bar"),
	}
	for i, expected := range testCases {
		actual, err := ParseIdentifier(i)
		require.NoError(t, err, i)
		require.Equal(t, expected, actual, i)
	}
}

func TestEmptyQuery(t *testing.T) {
	tests := []struct {
		in string
	}{
		{in: "{}"},
		{in: "{ }"},
		{in: "{                    }"},
		{in: "{ true }"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)
			require.NoError(t, err, tc.in)
			require.Equal(t, &RootExpr{newPipeline(newSpansetFilter(NewStaticBool(true)))}, actual, tc.in)
		})
	}
}
