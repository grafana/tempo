package traceql

import (
	"fmt"
	"math"
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
		{in: "", err: newParseError("syntax error: unexpected $end", 0, 0)},
		{in: "{ .a } | { .b", err: newParseError("syntax error: unexpected $end", 1, 14)},
		{in: "{ .a | .b }", err: newParseError("syntax error: unexpected |", 1, 6)},
		{in: "({ .a } | { .b }", err: newParseError("syntax error: unexpected $end, expecting ) or |", 1, 17)},
		{in: "({ .a } | { .b }) + ({ .a } | { .b })", err: newParseError("syntax error: unexpected +, expecting with", 1, 19)},
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
		in          string
		expected    SpansetOperation
		expectedStr string
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
			expectedStr: "(({ .a }|{ .b }) > ({ .a }|{ .b })) && ({ .a }|{ .b })",
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
			expectedStr: "({ .a }|{ .b }) > (({ .a }|{ .b }) && ({ .a }|{ .b }))",
		},
		{
			in: "({ .a } | { .b }) < (({ .a } | { .b }) && ({ .a } | { .b }))",
			expected: newSpansetOperation(OpSpansetParent,
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
			expectedStr: "({ .a }|{ .b }) < (({ .a }|{ .b }) && ({ .a }|{ .b }))",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(tc.expected)), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestPipelineSpansetOperators(t *testing.T) {
	tests := []struct {
		in          string
		expected    SpansetOperation
		expectedStr string
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
			expectedStr: "({ .a }|{ .b }) > ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) < ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetParent,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) < ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) ~ ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetSibling,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) ~ ({ .a }|{ .b })",
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
			expectedStr: "({ .a }|{ .b }) && ({ .a }|{ .b })",
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
			expectedStr: "({ .a }|{ .b }) >> ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) << ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetAncestor,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) << ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) !> ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetNotChild,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) !> ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) !< ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetNotParent,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) !< ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) !~ ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetNotSibling,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) !~ ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) !>> ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetNotDescendant,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) !>> ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) !<< ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetNotAncestor,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) !<< ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) &> ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetUnionChild,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) &> ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) &< ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetUnionParent,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) &< ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) &~ ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetUnionSibling,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) &~ ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) &>> ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetUnionDescendant,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) &>> ({ .a }|{ .b })",
		},
		{
			in: "({ .a } | { .b }) &<< ({ .a } | { .b })",
			expected: newSpansetOperation(OpSpansetUnionAncestor,
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
				newPipeline(
					newSpansetFilter(NewAttribute("a")),
					newSpansetFilter(NewAttribute("b")),
				),
			),
			expectedStr: "({ .a }|{ .b }) &<< ({ .a }|{ .b })",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(tc.expected)), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestPipelineScalarOperators(t *testing.T) {
	tests := []struct {
		in          string
		expected    ScalarFilter
		expectedStr string
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
			expectedStr: "({ .a }|count()) = ({ .a }|count())",
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
			expectedStr: "({ .a }|count()) != ({ .a }|count())",
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
			expectedStr: "({ .a }|count()) < ({ .a }|count())",
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
			expectedStr: "({ .a }|count()) <= ({ .a }|count())",
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
			expectedStr: "({ .a }|count()) >= ({ .a }|count())",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(tc.expected)), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestPipelines(t *testing.T) {
	tests := []struct {
		in          string
		expected    Pipeline
		expectedStr string
	}{
		{
			in: "{ .a } | { .b }",
			expected: newPipeline(
				newSpansetFilter(NewAttribute("a")),
				newSpansetFilter(NewAttribute("b")),
			),
			expectedStr: "{ .a }|{ .b }",
		},
		{
			in: "{ .a } | count() > 1",
			expected: newPipeline(
				newSpansetFilter(NewAttribute("a")),
				newScalarFilter(OpGreater, newAggregate(aggregateCount, nil), NewStaticInt(1)),
			),
			expectedStr: "{ .a }|(count()) > 1",
		},
		{
			in: "{ .a } | by(.namespace) | coalesce() | avg(duration) = 1s ",
			expected: newPipeline(
				newSpansetFilter(NewAttribute("a")),
				newGroupOperation(NewAttribute("namespace")),
				newCoalesceOperation(),
				newScalarFilter(OpEqual, newAggregate(aggregateAvg, NewIntrinsic(IntrinsicDuration)), NewStaticDuration(time.Second)),
			),
			expectedStr: "{ .a }|by(.namespace)|coalesce()|(avg(duration)) = 1s",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(tc.expected), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestGroupCoalesceErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "by(.a) && { .b }", err: newParseError("syntax error: unexpected &&, expecting with", 0, 8)},
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
		in          string
		expected    Pipeline
		expectedStr string
	}{
		{in: "by(.a) | coalesce()", expected: newPipeline(newGroupOperation(NewAttribute("a")), newCoalesceOperation()), expectedStr: "by(.a)|coalesce()"},
		{in: "by(.a + .b)", expected: newPipeline(newGroupOperation(newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b")))), expectedStr: "by(.a + .b)"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(tc.expected), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestSelectErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "select(.a) && { .b }", err: newParseError("syntax error: unexpected &&, expecting with", 0, 12)},
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
		in          string
		expected    Pipeline
		expectedStr string
	}{
		{in: "select(.a)", expected: newPipeline(newSelectOperation([]Attribute{NewAttribute("a")})), expectedStr: "select(.a)"},
		{in: "select(.a,.b)", expected: newPipeline(newSelectOperation([]Attribute{NewAttribute("a"), NewAttribute("b")})), expectedStr: "select(.a, .b)"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(tc.expected), actual)
			require.Equal(t, tc.expectedStr, actual.String())
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
		in          string
		expected    SpansetOperation
		expectedStr string
	}{
		{
			in: "{ true } && { false } >> { `a` }",
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetFilter(NewStaticBool(true)),
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(false)), newSpansetFilter(NewStaticString("a"))),
			),
			expectedStr: "({ true }) && (({ false }) >> ({ `a` }))",
		},
		{
			in: "{ true } >> { false } && { `a` }",
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))),
				newSpansetFilter(NewStaticString("a")),
			),
			expectedStr: "(({ true }) >> ({ false })) && ({ `a` })",
		},
		{
			in: "({ true } >> { false }) && { `a` }",
			expected: newSpansetOperation(OpSpansetAnd,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))),
				newSpansetFilter(NewStaticString("a")),
			),
			expectedStr: "(({ true }) >> ({ false })) && ({ `a` })",
		},
		{
			in: "{ true } >> { false } ~ { `a` }",
			expected: newSpansetOperation(OpSpansetSibling,
				newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))),
				newSpansetFilter(NewStaticString("a")),
			),
			expectedStr: "(({ true }) >> ({ false })) ~ ({ `a` })",
		},
		{
			in: "{ true } ~ { false } >> { `a` }",
			expected: newSpansetOperation(OpSpansetDescendant,
				newSpansetOperation(OpSpansetSibling, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))),
				newSpansetFilter(NewStaticString("a")),
			),
			expectedStr: "(({ true }) ~ ({ false })) >> ({ `a` })",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(tc.expected)), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestSpansetExpressionOperators(t *testing.T) {
	tests := []struct {
		in          string
		expected    SpansetOperation
		expectedStr string
	}{
		{in: "{ true } && { false }", expected: newSpansetOperation(OpSpansetAnd, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) && ({ false })"},
		{in: "{ true } > { false }", expected: newSpansetOperation(OpSpansetChild, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) > ({ false })"},
		{in: "{ true } < { false }", expected: newSpansetOperation(OpSpansetParent, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) < ({ false })"},
		{in: "{ true } >> { false }", expected: newSpansetOperation(OpSpansetDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) >> ({ false })"},
		{in: "{ true } << { false }", expected: newSpansetOperation(OpSpansetAncestor, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) << ({ false })"},
		{in: "{ true } || { false }", expected: newSpansetOperation(OpSpansetUnion, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) || ({ false })"},
		{in: "{ true } ~ { false }", expected: newSpansetOperation(OpSpansetSibling, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) ~ ({ false })"},
		// this test was added to highlight the one shift/reduce conflict in the grammar. this could also be parsed as two spanset pipelines &&ed together.
		{in: "({ true }) && ({ false })", expected: newSpansetOperation(OpSpansetAnd, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) && ({ false })"},
		{in: "{ true } !> { false }", expected: newSpansetOperation(OpSpansetNotChild, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) !> ({ false })"},
		{in: "{ true } !< { false }", expected: newSpansetOperation(OpSpansetNotParent, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) !< ({ false })"},
		{in: "{ true } !>> { false }", expected: newSpansetOperation(OpSpansetNotDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) !>> ({ false })"},
		{in: "{ true } !<< { false }", expected: newSpansetOperation(OpSpansetNotAncestor, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) !<< ({ false })"},
		{in: "{ true } !~ { false }", expected: newSpansetOperation(OpSpansetNotSibling, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) !~ ({ false })"},
		{in: "{ true } &> { false }", expected: newSpansetOperation(OpSpansetUnionChild, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) &> ({ false })"},
		{in: "{ true } &< { false }", expected: newSpansetOperation(OpSpansetUnionParent, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) &< ({ false })"},
		{in: "{ true } &>> { false }", expected: newSpansetOperation(OpSpansetUnionDescendant, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) &>> ({ false })"},
		{in: "{ true } &<< { false }", expected: newSpansetOperation(OpSpansetUnionAncestor, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) &<< ({ false })"},
		{in: "{ true } &~ { false }", expected: newSpansetOperation(OpSpansetUnionSibling, newSpansetFilter(NewStaticBool(true)), newSpansetFilter(NewStaticBool(false))), expectedStr: "({ true }) &~ ({ false })"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(tc.expected)), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestScalarExpressionErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{in: "(avg(.foo) > count()) + sum(.bar)", err: newParseError("syntax error: unexpected +, expecting with", 1, 23)},
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
		in          string
		expected    ScalarFilter
		expectedStr string
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
			expectedStr: "(avg(.foo)) > ((count()) + (sum(.bar)))",
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
			expectedStr: "((avg(.foo)) + (count())) > (sum(.bar))",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(tc.expected)), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestScalarExpressionOperators(t *testing.T) {
	tests := []struct {
		in          string
		expected    ScalarFilter
		expectedStr string
	}{
		{in: "count() > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateCount, nil), NewStaticInt(1)), expectedStr: "(count()) > 1"},
		{in: "max(.a) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateMax, NewAttribute("a")), NewStaticInt(1)), expectedStr: "(max(.a)) > 1"},
		{in: "min(1) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateMin, NewStaticInt(1)), NewStaticInt(1)), expectedStr: "(min(1)) > 1"},
		{in: "sum(true) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateSum, NewStaticBool(true)), NewStaticInt(1)), expectedStr: "(sum(true)) > 1"},
		{in: "avg(`c`) > 1", expected: newScalarFilter(OpGreater, newAggregate(aggregateAvg, NewStaticString("c")), NewStaticInt(1)), expectedStr: "(avg(`c`)) > 1"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(tc.expected)), actual)
			require.Equal(t, tc.expectedStr, actual.String())
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
		in          string
		expected    FieldExpression
		expectedStr string
	}{
		{
			in: "{ .a * .b + .c }",
			expected: newBinaryOperation(OpAdd,
				newBinaryOperation(OpMult, NewAttribute("a"), NewAttribute("b")),
				NewAttribute("c")),
			expectedStr: "{ (.a * .b) + .c }",
		},
		{
			in: "{ .a + .b * .c }",
			expected: newBinaryOperation(OpAdd,
				NewAttribute("a"),
				newBinaryOperation(OpMult, NewAttribute("b"), NewAttribute("c"))),
			expectedStr: "{ .a + (.b * .c) }",
		},
		{
			in: "{ ( .a + .b ) * .c }",
			expected: newBinaryOperation(OpMult,
				newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b")),
				NewAttribute("c")),
			expectedStr: "{ (.a + .b) * .c }",
		},
		{
			in: "{ .a + .b ^ .c }",
			expected: newBinaryOperation(OpAdd,
				NewAttribute("a"),
				newBinaryOperation(OpPower, NewAttribute("b"), NewAttribute("c"))),
			expectedStr: "{ .a + (.b ^ .c) }",
		},
		{
			in: "{ .a = .b + .c }",
			expected: newBinaryOperation(OpEqual,
				NewAttribute("a"),
				newBinaryOperation(OpAdd, NewAttribute("b"), NewAttribute("c"))),
			expectedStr: "{ .a = (.b + .c) }",
		},
		{
			in: "{ .a + .b = .c }",
			expected: newBinaryOperation(OpEqual,
				newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b")),
				NewAttribute("c")),
			expectedStr: "{ (.a + .b) = .c }",
		},
		{
			in: "{ .c - -.a + .b }",
			expected: newBinaryOperation(OpAdd,
				newBinaryOperation(OpSub, NewAttribute("c"), newUnaryOperation(OpSub, NewAttribute("a"))),
				NewAttribute("b")),
			expectedStr: "{ (.c - (-.a)) + .b }",
		},
		{
			in: "{ .c - -( .a + .b ) }",
			expected: newBinaryOperation(OpSub,
				NewAttribute("c"),
				newUnaryOperation(OpSub, newBinaryOperation(OpAdd, NewAttribute("a"), NewAttribute("b")))),
			expectedStr: "{ .c - (-(.a + .b)) }",
		},
		{
			in: "{ .a && .b = .c }",
			expected: newBinaryOperation(OpAnd,
				NewAttribute("a"),
				newBinaryOperation(OpEqual, NewAttribute("b"), NewAttribute("c"))),
			expectedStr: "{ .a && (.b = .c) }",
		},
		{
			in: "{ .a = .b && .c }",
			expected: newBinaryOperation(OpAnd,
				newBinaryOperation(OpEqual, NewAttribute("a"), NewAttribute("b")),
				NewAttribute("c")),
			expectedStr: "{ (.a = .b) && .c }",
		},
		{
			in: "{ .a = !.b && .c }",
			expected: newBinaryOperation(OpAnd,
				newBinaryOperation(OpEqual, NewAttribute("a"), newUnaryOperation(OpNot, NewAttribute("b"))),
				NewAttribute("c")),
			expectedStr: "{ (.a = (!.b)) && .c }",
		},
		{
			in: "{ .a = !( .b && .c ) }",
			expected: newBinaryOperation(OpEqual,
				NewAttribute("a"),
				newUnaryOperation(OpNot, newBinaryOperation(OpAnd, NewAttribute("b"), NewAttribute("c")))),
			expectedStr: "{ .a = (!(.b && .c)) }",
		},
		{
			in: "{ .a = .b || .c = .d}",
			expected: newBinaryOperation(OpOr,
				newBinaryOperation(OpEqual, NewAttribute("a"), NewAttribute("b")),
				newBinaryOperation(OpEqual, NewAttribute("c"), NewAttribute("d"))),
			expectedStr: "{ (.a = .b) || (.c = .d) }",
		},
		{
			in: "{ !.a = .b }",
			expected: newBinaryOperation(OpEqual,
				newUnaryOperation(OpNot, NewAttribute("a")),
				NewAttribute("b")),
			expectedStr: "{ (!.a) = .b }",
		},
		{
			in: "{ !(.a = .b) }",
			expected: newUnaryOperation(OpNot, newBinaryOperation(OpEqual,
				NewAttribute("a"),
				NewAttribute("b"))),
			expectedStr: "{ !(.a = .b) }",
		},
		{
			in: "{ -.a = .b }",
			expected: newBinaryOperation(OpEqual,
				newUnaryOperation(OpSub, NewAttribute("a")),
				NewAttribute("b")),
			expectedStr: "{ (-.a) = .b }",
		},
		{
			in: "{ -(.a = .b) }",
			expected: newUnaryOperation(OpSub, newBinaryOperation(OpEqual,
				NewAttribute("a"),
				NewAttribute("b"))),
			expectedStr: "{ -(.a = .b) }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(tc.expected))), actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestSpansetFilterStatics(t *testing.T) {
	tests := []struct {
		in          string
		expected    FieldExpression
		expectedStr string
	}{
		{in: "{ true }", expected: NewStaticBool(true), expectedStr: "{ true }"},
		{in: "{ false }", expected: NewStaticBool(false), expectedStr: "{ false }"},
		{in: `{ "true" }`, expected: NewStaticString("true"), expectedStr: "{ `true` }"},
		{in: `{ "true\"" }`, expected: NewStaticString("true\""), expectedStr: "{ `true\"` }"},
		{in: "{ `foo` }", expected: NewStaticString("foo"), expectedStr: "{ `foo` }"},
		{in: "{ .foo }", expected: NewAttribute("foo"), expectedStr: "{ .foo }"},
		{in: "{ duration }", expected: NewIntrinsic(IntrinsicDuration), expectedStr: "{ duration }"},
		{in: "{ name }", expected: NewIntrinsic(IntrinsicName), expectedStr: "{ name }"},
		{in: "{ parent }", expected: NewIntrinsic(IntrinsicParent), expectedStr: "{ parent }"},
		{in: "{ status }", expected: NewIntrinsic(IntrinsicStatus), expectedStr: "{ status }"},
		{in: "{ statusMessage }", expected: NewIntrinsic(IntrinsicStatusMessage), expectedStr: "{ statusMessage }"},
		{in: "{ 4321 }", expected: NewStaticInt(4321), expectedStr: "{ 4321 }"},
		{in: "{ maxInt }", expected: NewStaticInt(math.MaxInt), expectedStr: "{ 9223372036854775807 }"},
		{in: "{ minInt }", expected: NewStaticInt(math.MinInt), expectedStr: "{ -9223372036854775808 }"},
		{in: "{ 1.234 }", expected: NewStaticFloat(1.234), expectedStr: "{ 1.234 }"},
		{in: "{ 3h }", expected: NewStaticDuration(3 * time.Hour), expectedStr: "{ 3h0m0s }"},
		{in: "{ 1.5m }", expected: NewStaticDuration(1*time.Minute + 30*time.Second), expectedStr: "{ 1m30s }"},
		{in: "{ error }", expected: NewStaticStatus(StatusError), expectedStr: "{ error }"},
		{in: "{ ok }", expected: NewStaticStatus(StatusOk), expectedStr: "{ ok }"},
		{in: "{ unset }", expected: NewStaticStatus(StatusUnset), expectedStr: "{ unset }"},
		{in: "{ unspecified }", expected: NewStaticKind(KindUnspecified), expectedStr: "{ unspecified }"},
		{in: "{ internal }", expected: NewStaticKind(KindInternal), expectedStr: "{ internal }"},
		{in: "{ client }", expected: NewStaticKind(KindClient), expectedStr: "{ client }"},
		{in: "{ server }", expected: NewStaticKind(KindServer), expectedStr: "{ server }"},
		{in: "{ producer }", expected: NewStaticKind(KindProducer), expectedStr: "{ producer }"},
		{in: "{ consumer }", expected: NewStaticKind(KindConsumer), expectedStr: "{ consumer }"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(tc.expected))), actual)
			require.Equal(t, tc.expectedStr, actual.String())
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
		{in: "{ .a != nil }", expected: newUnaryOperation(OpExists, NewAttribute("a")), alsoTestWithoutSpace: true},
		{in: "{ .a = nil }", expected: newUnaryOperation(OpNotExists, NewAttribute("a")), alsoTestWithoutSpace: true},

		// nil comparisons
		{in: "{ nil != nil }", expected: NewStaticBool(false)},
		{in: "{ nil = nil }", expected: NewStaticBool(false)},
	}

	test := func(q string, expected FieldExpression) {
		actual, err := Parse(q)
		require.NoError(t, err, q)
		require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(expected))), actual, q)
	}

	for _, tc := range tests {
		t.Run(tc.in, func(_ *testing.T) {
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
		{in: `{ . "foo" }`, err: newParseError("syntax error: unexpected END_ATTRIBUTE, expecting IDENTIFIER", 1, 3)},
		{in: "{ .foo .bar }", err: newParseError("syntax error: unexpected .", 1, 8)},
		{in: "{ parent. }", err: newParseError("syntax error: unexpected END_ATTRIBUTE, expecting IDENTIFIER or resource. or span.", 0, 3)},
		{in: ".3foo", err: newParseError("syntax error: unexpected IDENTIFIER", 1, 3)},
		{in: `{ ."foo }`, err: newParseError(`unexpected EOF, expecting "`, 0, 3)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			require.Equal(t, tc.err, err)
		})
	}
}

// TestBinaryAndUnaryOperationsRewrites tests code in the newBinaryOperation and newUnaryOperation functions
// that attempts to simplify combinations of static values where possible.
func TestBinaryAndUnaryOperationsRewrites(t *testing.T) {
	tests := []struct {
		in       string
		expected FieldExpression
	}{
		// collapse to statics
		{in: "{ duration > 1 + 2}", expected: newBinaryOperation(OpGreater, NewIntrinsic(IntrinsicDuration), NewStaticInt(3))},
		{in: "{ -1 }", expected: NewStaticInt(-1)},
		{in: "{ 1 + 1 > 1 }", expected: NewStaticBool(true)},
		{in: "{ `foo` = `bar` }", expected: NewStaticBool(false)},
		{in: "{ 1 = 1. }", expected: NewStaticBool(true)}, // this is an interesting case, it returns true even though { span.foo = 1 } would be false if span.foo had the float value 1.0
		{in: "{ .1 + 1 }", expected: NewStaticFloat(1.1)},
		{in: "{ 1 * -1 = -1 }", expected: NewStaticBool(true)},
		{in: "{ .foo * -1. = -1 }", expected: newBinaryOperation(OpEqual, newBinaryOperation(OpMult, NewAttribute("foo"), NewStaticFloat(-1)), NewStaticInt(-1))},
		// rewrite != nil to existence
		{in: "{ .foo != nil }", expected: newUnaryOperation(OpExists, NewAttribute("foo"))},
		{in: "{ .foo = nil }", expected: newUnaryOperation(OpNotExists, NewAttribute("foo"))},
		{in: "{ nil != .foo }", expected: newUnaryOperation(OpExists, NewAttribute("foo"))},
		{in: "{ nil = .foo }", expected: newUnaryOperation(OpNotExists, NewAttribute("foo"))},
	}

	test := func(t *testing.T, q string, expected FieldExpression) {
		actual, err := Parse(q)
		require.NoError(t, err, q)
		require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(expected))), actual, q)
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			test(t, tc.in, tc.expected)
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
		{in: "event.foo.bar", expected: NewScopedAttribute(AttributeScopeEvent, false, "foo.bar")},
		{in: "link.foo.bar", expected: NewScopedAttribute(AttributeScopeLink, false, "foo.bar")},
		{in: "instrumentation.foo.bar", expected: NewScopedAttribute(AttributeScopeInstrumentation, false, "foo.bar")},
		{in: "parent.resource.foo", expected: NewScopedAttribute(AttributeScopeResource, true, "foo")},
		{in: "parent.span.foo", expected: NewScopedAttribute(AttributeScopeSpan, true, "foo")},
		{in: "parent.resource.foo.bar.baz", expected: NewScopedAttribute(AttributeScopeResource, true, "foo.bar.baz")},
		{in: "parent.span.foo.bar", expected: NewScopedAttribute(AttributeScopeSpan, true, "foo.bar")},
		{in: `."bar z".foo`, expected: NewAttribute("bar z.foo")},
		{in: `span."bar z".foo`, expected: NewScopedAttribute(AttributeScopeSpan, false, "bar z.foo")},
		{in: `."bar z".foo."bar"`, expected: NewAttribute("bar z.foo.bar")},
		{in: `.foo."bar baz"`, expected: NewAttribute("foo.bar baz")},
		{in: `.foo."bar baz".bar`, expected: NewAttribute("foo.bar baz.bar")},
		{in: `.foo."bar \" baz"`, expected: NewAttribute(`foo.bar " baz`)},
		{in: `.foo."bar \\ baz"`, expected: NewAttribute(`foo.bar \ baz`)},
		{in: `.foo."bar \\"." baz"`, expected: NewAttribute(`foo.bar \. baz`)},
		{in: `."foo.bar"`, expected: NewAttribute(`foo.bar`)},
		{in: `."🤘"`, expected: NewAttribute(`🤘`)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			s := "{ " + tc.in + " }"
			actual, err := Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(tc.expected))), actual)

			s = "{" + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(tc.expected))), actual)

			s = "{ (" + tc.in + ") }"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(tc.expected))), actual)

			s = "{ " + tc.in + " + " + tc.in + " }"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(newBinaryOperation(OpAdd, tc.expected, tc.expected)))), actual)
		})
	}
}

func TestIntrinsics(t *testing.T) {
	tests := []struct {
		in       string
		expected Intrinsic
	}{
		{in: "duration", expected: IntrinsicDuration},
		{in: "name", expected: IntrinsicName},
		{in: "status", expected: IntrinsicStatus},
		{in: "statusMessage", expected: IntrinsicStatusMessage},
		{in: "kind", expected: IntrinsicKind},
		{in: "parent", expected: IntrinsicParent},
		{in: "traceDuration", expected: IntrinsicTraceDuration},
		{in: "rootServiceName", expected: IntrinsicTraceRootService},
		{in: "rootName", expected: IntrinsicTraceRootSpan},
		{in: "nestedSetLeft", expected: IntrinsicNestedSetLeft},
		{in: "nestedSetRight", expected: IntrinsicNestedSetRight},
		{in: "nestedSetParent", expected: IntrinsicNestedSetParent},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			// as intrinsic e.g. duration
			s := "{ " + tc.in + " }"
			actual, err := Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: tc.expected,
				}))), actual)

			// as attribute e.g .duration
			s = "{ ." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))), actual)

			// as span scoped attribute e.g span.duration
			s = "{ span." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeSpan,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))), actual)

			// as resource scoped attribute e.g resource.duration
			s = "{ resource." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeResource,
					Parent:    false,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))), actual)

			// as parent scoped intrinsic e.g parent.duration
			s = "{ parent." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))), actual)

			// as nested parent scoped intrinsic e.g. parent.duration.foo
			// this becomes lookup on attribute named "duration.foo"
			s = "{ parent." + tc.in + ".foo }"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeNone,
					Parent:    true,
					Name:      tc.in + ".foo",
					Intrinsic: IntrinsicNone,
				}))), actual)

			// as parent resource scoped attribute e.g. parent.resource.duration
			s = "{ parent.resource." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeResource,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))), actual)

			// as parent span scoped attribute e.g. praent.span.duration
			s = "{ parent.span." + tc.in + "}"
			actual, err = Parse(s)

			require.NoError(t, err)
			require.Equal(t, newRootExpr(newPipeline(
				newSpansetFilter(Attribute{
					Scope:     AttributeScopeSpan,
					Parent:    true,
					Name:      tc.in,
					Intrinsic: IntrinsicNone,
				}))), actual)
		})
	}
}

func TestScopedIntrinsics(t *testing.T) {
	tests := []struct {
		in          string
		expected    Intrinsic
		shouldError bool
	}{
		{in: "trace:duration", expected: IntrinsicTraceDuration},
		{in: "trace:rootName", expected: IntrinsicTraceRootSpan},
		{in: "trace:rootService", expected: IntrinsicTraceRootService},
		{in: "trace:id", expected: IntrinsicTraceID},
		{in: "span:duration", expected: IntrinsicDuration},
		{in: "span:kind", expected: IntrinsicKind},
		{in: "span:name", expected: IntrinsicName},
		{in: "span:status", expected: IntrinsicStatus},
		{in: "span:statusMessage", expected: IntrinsicStatusMessage},
		{in: "span:id", expected: IntrinsicSpanID},
		{in: "span:parentID", expected: IntrinsicParentID},
		{in: "span:childCount", expected: IntrinsicChildCount},
		{in: "event:name", expected: IntrinsicEventName},
		{in: "event:timeSinceStart", expected: IntrinsicEventTimeSinceStart},
		{in: "link:traceID", expected: IntrinsicLinkTraceID},
		{in: "link:spanID", expected: IntrinsicLinkSpanID},
		{in: "instrumentation:name", expected: IntrinsicInstrumentationName},
		{in: "instrumentation:version", expected: IntrinsicInstrumentationVersion},
		{in: ":duration", shouldError: true},
		{in: ":statusMessage", shouldError: true},
		{in: "trace:name", shouldError: true},
		{in: "trace:rootServiceName", shouldError: true},
		{in: "span:rootServiceName", shouldError: true},
		{in: "parent:id", shouldError: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			// as scoped intrinsic e.g :duration
			s := "{ " + tc.in + "}"
			actual, err := Parse(s)

			if tc.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, newRootExpr(newPipeline(
					newSpansetFilter(Attribute{
						Scope:     AttributeScopeNone,
						Parent:    false,
						Name:      tc.expected.String(),
						Intrinsic: tc.expected,
					}))), actual)
			}
		})
	}
}

func TestParseIdentifier(t *testing.T) {
	testCases := map[string]Attribute{
		// Basic intrinsics
		"name":            NewIntrinsic(IntrinsicName),
		"status":          NewIntrinsic(IntrinsicStatus),
		"statusMessage":   NewIntrinsic(IntrinsicStatusMessage),
		"kind":            NewIntrinsic(IntrinsicKind),
		"duration":        NewIntrinsic(IntrinsicDuration),
		"parent":          NewIntrinsic(IntrinsicParent),
		"traceDuration":   NewIntrinsic(IntrinsicTraceDuration),
		"rootName":        NewIntrinsic(IntrinsicTraceRootSpan),
		"rootServiceName": NewIntrinsic(IntrinsicTraceRootService),
		"nestedSetLeft":   NewIntrinsic(IntrinsicNestedSetLeft),
		"nestedSetRight":  NewIntrinsic(IntrinsicNestedSetRight),
		"nestedSetParent": NewIntrinsic(IntrinsicNestedSetParent),

		// Scoped intrinsics - trace:
		"trace:duration":    NewIntrinsic(IntrinsicTraceDuration),
		"trace:rootName":    NewIntrinsic(IntrinsicTraceRootSpan),
		"trace:rootService": NewIntrinsic(IntrinsicTraceRootService),
		"trace:id":          NewIntrinsic(IntrinsicTraceID),

		// Scoped intrinsics - span:
		"span:duration":      NewIntrinsic(IntrinsicDuration),
		"span:name":          NewIntrinsic(IntrinsicName),
		"span:kind":          NewIntrinsic(IntrinsicKind),
		"span:status":        NewIntrinsic(IntrinsicStatus),
		"span:statusMessage": NewIntrinsic(IntrinsicStatusMessage),
		"span:id":            NewIntrinsic(IntrinsicSpanID),
		"span:parentID":      NewIntrinsic(IntrinsicParentID),
		"span:childCount":    NewIntrinsic(IntrinsicChildCount),

		// Scoped intrinsics - event:
		"event:name":           NewIntrinsic(IntrinsicEventName),
		"event:timeSinceStart": NewIntrinsic(IntrinsicEventTimeSinceStart),

		// Scoped intrinsics - link:
		"link:traceID": NewIntrinsic(IntrinsicLinkTraceID),
		"link:spanID":  NewIntrinsic(IntrinsicLinkSpanID),

		// Scoped intrinsics - instrumentation:
		"instrumentation:name":    NewIntrinsic(IntrinsicInstrumentationName),
		"instrumentation:version": NewIntrinsic(IntrinsicInstrumentationVersion),

		// Simple attributes
		".name":        NewAttribute("name"),
		".status":      NewAttribute("status"),
		".foo":         NewAttribute("foo"),
		".foo.bar":     NewAttribute("foo.bar"),
		".foo.bar.baz": NewAttribute("foo.bar.baz"),
		".http_status": NewAttribute("http_status"),
		".http-status": NewAttribute("http-status"),
		".http+":       NewAttribute("http+"),
		".foo3":        NewAttribute("foo3"),
		".foo.3":       NewAttribute("foo.3"),
		".\"0\"":       NewAttribute("0"),

		// Scoped attributes - resource
		"resource.foo":         NewScopedAttribute(AttributeScopeResource, false, "foo"),
		"resource.foo.bar":     NewScopedAttribute(AttributeScopeResource, false, "foo.bar"),
		"resource.foo.bar.baz": NewScopedAttribute(AttributeScopeResource, false, "foo.bar.baz"),

		// Scoped attributes - span
		"span.foo":         NewScopedAttribute(AttributeScopeSpan, false, "foo"),
		"span.foo.bar":     NewScopedAttribute(AttributeScopeSpan, false, "foo.bar"),
		"span.foo.bar.baz": NewScopedAttribute(AttributeScopeSpan, false, "foo.bar.baz"),

		// Scoped attributes - event
		"event.foo":     NewScopedAttribute(AttributeScopeEvent, false, "foo"),
		"event.foo.bar": NewScopedAttribute(AttributeScopeEvent, false, "foo.bar"),

		// Scoped attributes - link
		"link.foo":     NewScopedAttribute(AttributeScopeLink, false, "foo"),
		"link.foo.bar": NewScopedAttribute(AttributeScopeLink, false, "foo.bar"),

		// Scoped attributes - instrumentation
		"instrumentation.foo":     NewScopedAttribute(AttributeScopeInstrumentation, false, "foo"),
		"instrumentation.foo.bar": NewScopedAttribute(AttributeScopeInstrumentation, false, "foo.bar"),

		// Parent-scoped intrinsics
		"parent.duration": NewScopedAttribute(AttributeScopeNone, true, "duration"),
		"parent.name":     NewScopedAttribute(AttributeScopeNone, true, "name"),
		"parent.foo":      NewScopedAttribute(AttributeScopeNone, true, "foo"),
		"parent.foo.bar":  NewScopedAttribute(AttributeScopeNone, true, "foo.bar"),

		// Parent-scoped with resource/span
		"parent.resource.foo":     NewScopedAttribute(AttributeScopeResource, true, "foo"),
		"parent.resource.foo.bar": NewScopedAttribute(AttributeScopeResource, true, "foo.bar"),
		"parent.span.foo":         NewScopedAttribute(AttributeScopeSpan, true, "foo"),
		"parent.span.foo.bar":     NewScopedAttribute(AttributeScopeSpan, true, "foo.bar"),

		// Quoted identifiers with spaces
		".\"foo bar\"":            NewAttribute("foo bar"),
		".\"bar z\".foo":          NewAttribute("bar z.foo"),
		".\"bar z\".foo.\"bar\"":  NewAttribute("bar z.foo.bar"),
		".foo.\"bar baz\"":        NewAttribute("foo.bar baz"),
		".foo.\"bar baz\".bar":    NewAttribute("foo.bar baz.bar"),
		"span.\"foo bar\"":        NewScopedAttribute(AttributeScopeSpan, false, "foo bar"),
		"span.\"bar z\".foo":      NewScopedAttribute(AttributeScopeSpan, false, "bar z.foo"),
		"resource.\"foo bar\"":    NewScopedAttribute(AttributeScopeResource, false, "foo bar"),
		"parent.\"foo bar\"":      NewScopedAttribute(AttributeScopeNone, true, "foo bar"),
		"parent.resource.\"foo\"": NewScopedAttribute(AttributeScopeResource, true, "foo"),

		// Quoted identifiers with escape sequences
		".foo.\"bar \\\" baz\"":      NewAttribute("foo.bar \" baz"),
		".foo.\"bar \\\\ baz\"":      NewAttribute("foo.bar \\ baz"),
		".foo.\"bar \\\\\".\" baz\"": NewAttribute("foo.bar \\. baz"),
		".\"foo.bar\"":               NewAttribute("foo.bar"),

		// Unicode identifiers
		".😝":     NewAttribute("😝"),
		".\"🤘\"": NewAttribute("🤘"),
	}
	for input, expected := range testCases {
		t.Run(input, func(t *testing.T) {
			actual, err := ParseIdentifier(input)
			require.NoError(t, err, "input: %s", input)
			require.Equal(t, expected, actual, "input: %s", input)
		})
	}
}

func TestParseIdentifierErrors(t *testing.T) {
	errorCases := []struct {
		input       string
		description string
	}{
		// Empty and whitespace
		{"", "empty string"},
		{"   ", "only whitespace"},

		// Invalid syntax
		{"{ .foo }", "curly braces (already a filter)"},
		{".foo &&", "incomplete expression with operator"},
		{".foo && .bar", "multiple identifiers with AND"},
		{".foo || .bar", "multiple identifiers with OR"},
		{".foo > .bar", "comparison operator"},

		// Expressions that aren't identifiers
		{".foo = \"bar\"", "comparison expression"},
		{".foo + .bar", "arithmetic expression"},
		{".foo != nil", "existence check"},
		{"true", "boolean literal"},
		{"123", "numeric literal"},
		{"\"string\"", "string literal"},
		{"`backtick`", "backtick string literal"},

		// Aggregate functions
		{"count()", "aggregate function"},
		{"max(.foo)", "aggregate with attribute"},
		{"avg(duration)", "aggregate with intrinsic"},

		// Invalid scoped intrinsics
		{":duration", "missing scope prefix"},
		{"trace:name", "invalid trace intrinsic"},
		{"parent:id", "invalid parent scope"},

		// Incomplete attributes
		{".", "dot without identifier"},
		{"span.", "scope without attribute"},
		{"resource.", "resource without attribute"},
		{"parent.", "parent without attribute"},

		// Invalid quoted strings
		{".\"foo", "unclosed quote"},
		{".\"foo\\", "unclosed quote with escape"},

		// Complex queries
		{"{ .foo } | { .bar }", "pipeline expression"},
		{"({ .foo })", "wrapped spanset"},
	}

	for _, tc := range errorCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := ParseIdentifier(tc.input)
			require.Error(t, err, "expected error for input: %s", tc.input)
			require.Contains(t, err.Error(), "failed to parse identifier", "error should have context")
		})
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
			require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(NewStaticBool(true)))), actual, tc.in)
			require.Equal(t, "{ true }", actual.String())
		})
	}
}

func TestHints(t *testing.T) {
	tests := []struct {
		in          string
		expected    *RootExpr
		expectedStr string
	}{
		{
			in: `{ } | rate() with(foo="bar")`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
			).withHints(newHints([]*Hint{
				newHint("foo", NewStaticString("bar")),
			})),
			expectedStr: "{ true } | rate() with(foo=`bar`)",
		},
		{
			in: `{ } | rate() with(foo=0.5)`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
			).withHints(newHints([]*Hint{
				newHint("foo", NewStaticFloat(0.5)),
			})),
			expectedStr: "{ true } | rate() with(foo=0.5)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestReallyLongQuery(t *testing.T) {
	for i := 1000; i < 1050; i++ {
		longVal := strings.Repeat("a", i)

		// static value
		query := fmt.Sprintf("{ .a = `%s` }", longVal)
		expected := newBinaryOperation(OpEqual, NewAttribute("a"), NewStaticString(longVal))

		actual, err := Parse(query)

		require.NoError(t, err, "i=%d", i)
		require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(expected))), actual, "i=%d", i)

		// attr name
		query = fmt.Sprintf("{ .%s = `foo` }", longVal)
		expected = newBinaryOperation(OpEqual, NewAttribute(longVal), NewStaticString("foo"))

		actual, err = Parse(query)

		require.NoError(t, err, "i=%d", i)
		require.Equal(t, newRootExpr(newPipeline(newSpansetFilter(expected))), actual, "i=%d", i)
	}
}

func TestMetrics(t *testing.T) {
	tests := []struct {
		in          string
		expected    *RootExpr
		expectedStr string
	}{
		{
			in: `{ } | rate()`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
			),
			expectedStr: `{ true } | rate()`,
		},
		{
			in: `{ } | count_over_time() by(name, span.http.status_code)`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateCountOverTime, []Attribute{
					NewIntrinsic(IntrinsicName),
					NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"),
				}),
			),
			expectedStr: `{ true } | count_over_time()by(name,span.http.status_code)`,
		},
		{
			in: `{ } | min_over_time(duration) by(name, span.http.status_code)`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregateWithAttr(metricsAggregateMinOverTime,
					NewIntrinsic(IntrinsicDuration),
					[]Attribute{
						NewIntrinsic(IntrinsicName),
						NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"),
					}),
			),
			expectedStr: `{ true } | min_over_time(duration)by(name,span.http.status_code)`,
		},
		{
			in: `{ } | max_over_time(duration) by(name, span.http.status_code)`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregateWithAttr(metricsAggregateMaxOverTime,
					NewIntrinsic(IntrinsicDuration),
					[]Attribute{
						NewIntrinsic(IntrinsicName),
						NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"),
					}),
			),
			expectedStr: `{ true } | max_over_time(duration)by(name,span.http.status_code)`,
		},
		{
			in: `{ } | avg_over_time(duration) by(name, span.http.status_code)`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newAverageOverTimeMetricsAggregator(
					NewIntrinsic(IntrinsicDuration),
					[]Attribute{
						NewIntrinsic(IntrinsicName),
						NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"),
					}),
			),
			expectedStr: `{ true } | avg_over_time(duration)by(name,span.http.status_code)`,
		},
		{
			in: `{ } | sum_over_time(duration) by(name, span.http.status_code)`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregateWithAttr(metricsAggregateSumOverTime,
					NewIntrinsic(IntrinsicDuration),
					[]Attribute{
						NewIntrinsic(IntrinsicName),
						NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"),
					}),
			),
			expectedStr: `{ true } | sum_over_time(duration)by(name,span.http.status_code)`,
		},
		{
			in: `{ } | quantile_over_time(duration, 0, 0.90, 0.95, 1) by(name, span.http.status_code)`,
			expected: newRootExprWithMetrics(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregateQuantileOverTime(
					NewIntrinsic(IntrinsicDuration),
					[]float64{0, 0.9, 0.95, 1.0},
					[]Attribute{
						NewIntrinsic(IntrinsicName),
						NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"),
					}),
			),
			expectedStr: `{ true } | quantile_over_time(duration,0.00000,0.90000,0.95000,1.00000)by(name,span.http.status_code)`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestMetricsSecondStage(t *testing.T) {
	tests := []struct {
		in          string
		expected    *RootExpr
		expectedStr string
	}{
		{
			in: `{ } | rate() | topk(10)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newTopKBottomK(OpTopK, 10),
			),
			expectedStr: `{ true } | rate() | topk(10)`,
		},
		{
			in: `{ } | count_over_time() by(name, span.http.status_code) | topk(10)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateCountOverTime, []Attribute{
					NewIntrinsic(IntrinsicName),
					NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"),
				}),
				newTopKBottomK(OpTopK, 10),
			),
			expectedStr: `{ true } | count_over_time()by(name,span.http.status_code) | topk(10)`,
		},
		{
			in: `{ } | rate() | topk(10) with(foo="bar")`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newTopKBottomK(OpTopK, 10),
			).withHints(newHints([]*Hint{
				newHint("foo", NewStaticString("bar")),
			})),
			expectedStr: "{ true } | rate() | topk(10) with(foo=`bar`)",
		},
		{
			in: `{ } | rate() | bottomk(10)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newTopKBottomK(OpBottomK, 10),
			),
			expectedStr: `{ true } | rate() | bottomk(10)`,
		},
		{
			in: `{ } | count_over_time() by(name, span.http.status_code) | bottomk(10)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateCountOverTime, []Attribute{
					NewIntrinsic(IntrinsicName),
					NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"),
				}),
				newTopKBottomK(OpBottomK, 10),
			),
			expectedStr: `{ true } | count_over_time()by(name,span.http.status_code) | bottomk(10)`,
		},
		{
			in: `{ } | rate() | bottomk(10) with(foo="bar")`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newTopKBottomK(OpBottomK, 10),
			).withHints(newHints([]*Hint{
				newHint("foo", NewStaticString("bar")),
			})),
			expectedStr: "{ true } | rate() | bottomk(10) with(foo=`bar`)",
		},
		// Combined second stage tests (topk/bottomk with filter)
		{
			in: `{ } | rate() | topk(5) > 10`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newChainedSecondStage(
					newTopKBottomK(OpTopK, 5),
					newMetricsFilter(OpGreater, 10),
				),
			),
			expectedStr: `{ true } | rate() | topk(5) > 10`,
		},
		{
			in: `{ } | rate() by(name) | bottomk(3) >= 5.5`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, []Attribute{
					NewIntrinsic(IntrinsicName),
				}),
				newChainedSecondStage(
					newTopKBottomK(OpBottomK, 3),
					newMetricsFilter(OpGreaterEqual, 5.5),
				),
			),
			expectedStr: `{ true } | rate()by(name) | bottomk(3) >= 5.5`,
		},
		{
			in: `{ } | count_over_time() | topk(10) != 0`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateCountOverTime, nil),
				newChainedSecondStage(
					newTopKBottomK(OpTopK, 10),
					newMetricsFilter(OpNotEqual, 0),
				),
			),
			expectedStr: `{ true } | count_over_time() | topk(10) != 0`,
		},
		{
			in: `{ } | count_over_time() | topk(10) != 0 with (sample=0.1)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateCountOverTime, nil),
				newChainedSecondStage(
					newTopKBottomK(OpTopK, 10),
					newMetricsFilter(OpNotEqual, 0),
				),
			).withHints(newHints([]*Hint{
				newHint("sample", NewStaticFloat(0.1)),
			})),
			expectedStr: `{ true } | count_over_time() | topk(10) != 0 with(sample=0.1)`,
		},
		// Combined: filter then topk/bottomk (e.g. {} | rate() > 10 | topk(5))
		{
			in: `{ } | rate() > 10 | topk(5)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newChainedSecondStage(
					newMetricsFilter(OpGreater, 10),
					newTopKBottomK(OpTopK, 5),
				),
			),
			expectedStr: `{ true } | rate() > 10 | topk(5)`,
		},
		{
			in: `{ } | rate() by(name) >= 5.5 | bottomk(3)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, []Attribute{
					NewIntrinsic(IntrinsicName),
				}),
				newChainedSecondStage(
					newMetricsFilter(OpGreaterEqual, 5.5),
					newTopKBottomK(OpBottomK, 3),
				),
			),
			expectedStr: `{ true } | rate()by(name) >= 5.5 | bottomk(3)`,
		},
		{
			in: `{ } | count_over_time() != 0 | topk(10)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateCountOverTime, nil),
				newChainedSecondStage(
					newMetricsFilter(OpNotEqual, 0),
					newTopKBottomK(OpTopK, 10),
				),
			),
			expectedStr: `{ true } | count_over_time() != 0 | topk(10)`,
		},
		{
			in: `{ } | count_over_time() | topk(1) != 0 | topk(10)`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateCountOverTime, nil),
				newChainedSecondStage(
					newTopKBottomK(OpTopK, 1),
					newMetricsFilter(OpNotEqual, 0),
					newTopKBottomK(OpTopK, 10),
				),
			),
			expectedStr: `{ true } | count_over_time() | topk(1) != 0 | topk(10)`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
			require.Equal(t, tc.expectedStr, actual.String())
		})
	}
}

func TestMetricsSecondStageErrors(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{
			in:  "{} | topk(10)",
			err: newParseError("syntax error: unexpected topk", 1, 6),
		},
		{
			in:  "{} | topk(10) with(sample=0.1)",
			err: newParseError("syntax error: unexpected topk", 1, 6),
		},
		{
			in:  "{} | rate() | topk(-1)",
			err: newParseError("syntax error: unexpected -, expecting INTEGER", 1, 20),
		},
		{
			in:  "{} | bottomk(10)",
			err: newParseError("syntax error: unexpected bottomk", 1, 6),
		},
		{
			in:  "{} | bottomk(10) with(sample=0.1)",
			err: newParseError("syntax error: unexpected bottomk", 1, 6),
		},
		{
			in:  "{} | rate() | bottomk(-1)",
			err: newParseError("syntax error: unexpected -, expecting INTEGER", 1, 23),
		},
		{
			in:  "{} > 10",
			err: newParseError("syntax error: unexpected INTEGER, expecting { or (", 1, 6),
		},
		{
			in:  "{} = 10",
			err: newParseError("syntax error: unexpected =, expecting with", 1, 4),
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := Parse(tc.in)

			require.Equal(t, tc.err, err)
		})
	}
}

func TestParseRewrites(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "no rewrites",
			query: "{ .attr = `foo` }",
			want:  "{ .attr = `foo` }",
		},
		{
			name:  "query with rewrite OR",
			query: "{ .attr = `foo` || .attr = `bar` } | rate()",
			want:  "{ .attr IN [`foo`, `bar`] } | rate()",
		},
		{
			name:  "query with rewrite AND",
			query: "{ .attr != `foo` && .attr != `bar` }",
			want:  "{ .attr NOT IN [`foo`, `bar`] }",
		},
		{
			name:  "query with rewrite OR regex",
			query: "{ .attr =~ `foo` || .attr =~ `bar` }",
			want:  "{ .attr MATCH ANY [`foo`, `bar`] }",
		},
		{
			name:  "query with rewrite AND regex",
			query: "{ .attr !~ `foo` && .attr !~ `bar` }",
			want:  "{ .attr MATCH NONE [`foo`, `bar`] }",
		},
		{
			name:  "skip rewrites hint",
			query: "{ .attr = `foo` || .attr = `bar` } | rate() with(skip_optimization=true)",
			want:  "{ (.attr = `foo`) || (.attr = `bar`) } | rate() with(skip_optimization=true)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := Parse(tc.query)
			require.NoError(t, err)
			require.Equal(t, tc.want, actual.String())
		})
	}
}

func TestMetricsFilter(t *testing.T) {
	tests := []struct {
		in          string
		expected    *RootExpr
		expectedStr string
	}{
		{
			in: `{ } | rate() > 10`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newMetricsFilter(OpGreater, 10),
			),
			expectedStr: `{ true } | rate() > 10`,
		},
		{
			in: `{ } | rate() by(name) >= 5.5`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, []Attribute{
					NewIntrinsic(IntrinsicName),
				}),
				newMetricsFilter(OpGreaterEqual, 5.5),
			),
			expectedStr: `{ true } | rate()by(name) >= 5.5`,
		},
		{
			in: `{ } | rate() > 10s`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newMetricsFilter(OpGreater, 10),
			),
			expectedStr: `{ true } | rate() > 10`,
		},
		{
			in: `{ } | count_over_time() < 100`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateCountOverTime, nil),
				newMetricsFilter(OpLess, 100),
			),
			expectedStr: `{ true } | count_over_time() < 100`,
		},
		{
			in: `{ } | rate() <= 0.5`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newMetricsFilter(OpLessEqual, 0.5),
			),
			expectedStr: `{ true } | rate() <= 0.5`,
		},
		{
			in: `{ } | rate() = 42`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newMetricsFilter(OpEqual, 42),
			),
			expectedStr: `{ true } | rate() = 42`,
		},
		{
			in: `{ } | rate() != 0`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newMetricsFilter(OpNotEqual, 0),
			),
			expectedStr: `{ true } | rate() != 0`,
		},
		{
			in: `{ } | rate() > -3`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newMetricsFilter(OpGreater, -3),
			),
			expectedStr: `{ true } | rate() > -3`,
		},
		{
			in: `{ } | rate() > -2.5`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newMetricsFilter(OpGreater, -2.5),
			),
			expectedStr: `{ true } | rate() > -2.5`,
		},
		{
			in: `{ } | rate() > 10 with(foo="bar")`,
			expected: newRootExprWithMetricsTwoStage(
				newPipeline(newSpansetFilter(NewStaticBool(true))),
				newMetricsAggregate(metricsAggregateRate, nil),
				newMetricsFilter(OpGreater, 10),
			).withHints(newHints([]*Hint{
				newHint("foo", NewStaticString("bar")),
			})),
			expectedStr: "{ true } | rate() > 10 with(foo=`bar`)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
			q := actual.String()
			require.Equal(t, tc.expectedStr, q, "stringified query should match expected string")
		})
	}
}
