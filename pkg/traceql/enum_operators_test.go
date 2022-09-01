package traceql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperatorIsBoolean(t *testing.T) {
	tt := []struct {
		op       Operator
		expected bool
	}{
		{OpAdd, false},
		{OpSub, false},
		{OpDiv, false},
		{OpMod, false},
		{OpMult, false},
		{OpEqual, true},
		{OpNotEqual, true},
		{OpRegex, true},
		{OpNotRegex, true},
		{OpGreater, true},
		{OpGreaterEqual, true},
		{OpLess, true},
		{OpLessEqual, true},
		{OpPower, false},
		{OpAnd, true},
		{OpOr, true},
		{OpNot, true},
		{OpSpansetChild, false},
		{OpSpansetDescendant, false},
		{OpSpansetAnd, false},
		{OpSpansetUnion, false},
		{OpSpansetSibling, false},
	}

	for _, tc := range tt {
		t.Run(tc.op.String(), func(t *testing.T) {
			actual := tc.op.isBoolean()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestOperatorBinaryTypesValid(t *testing.T) {
	tt := []struct {
		op       Operator
		t        StaticType
		expected bool
	}{
		// numeric
		{OpAdd, typeInt, true},
		{OpDiv, typeDuration, true},
		{OpMod, typeFloat, true},
		{OpMult, typeInt, true},
		{OpPower, typeDuration, true},
		{OpSub, typeAttribute, true},

		{OpAdd, typeString, false},
		{OpDiv, typeSpanset, false},
		{OpMod, typeStatus, false},
		{OpMult, typeNil, false},
		{OpPower, typeBoolean, false},
		// equality
		{OpEqual, typeDuration, true},
		{OpNotEqual, typeStatus, true},
		{OpEqual, typeString, true},
		{OpNotEqual, typeInt, true},
		{OpEqual, typeNil, true},
		{OpNotEqual, typeAttribute, true},
		{OpEqual, typeBoolean, true},
		{OpNotEqual, typeFloat, true},

		{OpEqual, typeSpanset, false},
		// range comparison
		{OpGreater, typeInt, true},
		{OpGreaterEqual, typeFloat, true},
		{OpLess, typeFloat, true},
		{OpLessEqual, typeDuration, true},

		{OpGreater, typeStatus, false},
		{OpGreaterEqual, typeNil, false},
		{OpLess, typeString, false},
		{OpLessEqual, typeBoolean, false},
		// string comparison
		{OpRegex, typeString, true},
		{OpNotRegex, typeAttribute, true},
		{OpRegex, typeString, true},

		{OpRegex, typeInt, false},
		{OpNotRegex, typeInt, false},
		// boolean
		{OpAnd, typeBoolean, true},
		{OpOr, typeAttribute, true},
		{OpAnd, typeAttribute, true},

		{OpAnd, typeDuration, false},
		{OpOr, typeDuration, false},
		// not
		{OpNot, typeBoolean, false},
	}

	for _, tc := range tt {
		t.Run(tc.op.String(), func(t *testing.T) {
			actual := tc.op.binaryTypesValid(tc.t, typeAttribute)
			assert.Equal(t, tc.expected, actual)
			actual = tc.op.binaryTypesValid(typeAttribute, tc.t)
			assert.Equal(t, tc.expected, actual)
			actual = tc.op.binaryTypesValid(tc.t, tc.t)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestOperatorUnaryTypesValid(t *testing.T) {
	tt := []struct {
		op       Operator
		t        StaticType
		expected bool
	}{
		{OpAdd, typeInt, false},
		{OpDiv, typeInt, false},
		{OpMod, typeInt, false},
		{OpMult, typeInt, false},
		{OpEqual, typeInt, false},
		{OpNotEqual, typeInt, false},
		{OpRegex, typeInt, false},
		{OpNotRegex, typeInt, false},
		{OpGreater, typeInt, false},
		{OpGreaterEqual, typeInt, false},
		{OpLess, typeInt, false},
		{OpLessEqual, typeInt, false},
		{OpPower, typeInt, false},
		{OpAnd, typeInt, false},
		{OpOr, typeInt, false},
		{OpSpansetChild, typeInt, false},
		{OpSpansetDescendant, typeInt, false},
		{OpSpansetAnd, typeInt, false},
		{OpSpansetUnion, typeInt, false},
		{OpSpansetSibling, typeInt, false},
		// not
		{OpNot, typeBoolean, true},
		{OpNot, typeInt, false},
		{OpNot, typeNil, false},
		{OpNot, typeString, false},
		// sub
		{OpSub, typeInt, true},
		{OpSub, typeFloat, true},
		{OpSub, typeDuration, true},
		{OpSub, typeBoolean, false},
		{OpSub, typeStatus, false},
		{OpSub, typeNil, false},
		{OpSub, typeSpanset, false},
	}

	for _, tc := range tt {
		t.Run(tc.op.String(), func(t *testing.T) {
			actual := tc.op.unaryTypesValid(tc.t)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
