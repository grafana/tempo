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
		{opAdd, false},
		{opSub, false},
		{opDiv, false},
		{opMod, false},
		{opMult, false},
		{opEqual, true},
		{opNotEqual, true},
		{opRegex, true},
		{opNotRegex, true},
		{opGreater, true},
		{opGreaterEqual, true},
		{opLess, true},
		{opLessEqual, true},
		{opPower, false},
		{opAnd, true},
		{opOr, true},
		{opNot, true},
		{opSpansetChild, false},
		{opSpansetDescendant, false},
		{opSpansetAnd, false},
		{opSpansetUnion, false},
		{opSpansetSibling, false},
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
		{opAdd, typeInt, true},
		{opDiv, typeDuration, true},
		{opMod, typeFloat, true},
		{opMult, typeInt, true},
		{opPower, typeDuration, true},
		{opSub, typeAttribute, true},

		{opAdd, typeString, false},
		{opDiv, typeSpanset, false},
		{opMod, typeStatus, false},
		{opMult, typeNil, false},
		{opPower, typeBoolean, false},
		// equality
		{opEqual, typeDuration, true},
		{opNotEqual, typeStatus, true},
		{opEqual, typeString, true},
		{opNotEqual, typeInt, true},
		{opEqual, typeNil, true},
		{opNotEqual, typeAttribute, true},
		{opEqual, typeBoolean, true},
		{opNotEqual, typeFloat, true},

		{opEqual, typeSpanset, false},
		// range comparison
		{opGreater, typeInt, true},
		{opGreaterEqual, typeFloat, true},
		{opLess, typeFloat, true},
		{opLessEqual, typeDuration, true},

		{opGreater, typeStatus, false},
		{opGreaterEqual, typeNil, false},
		{opLess, typeString, false},
		{opLessEqual, typeBoolean, false},
		// string comparison
		{opRegex, typeString, true},
		{opNotRegex, typeAttribute, true},
		{opRegex, typeString, true},

		{opRegex, typeInt, false},
		{opNotRegex, typeInt, false},
		// boolean
		{opAnd, typeBoolean, true},
		{opOr, typeAttribute, true},
		{opAnd, typeAttribute, true},

		{opAnd, typeDuration, false},
		{opOr, typeDuration, false},
		// not
		{opNot, typeBoolean, false},
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
		{opAdd, typeInt, false},
		{opDiv, typeInt, false},
		{opMod, typeInt, false},
		{opMult, typeInt, false},
		{opEqual, typeInt, false},
		{opNotEqual, typeInt, false},
		{opRegex, typeInt, false},
		{opNotRegex, typeInt, false},
		{opGreater, typeInt, false},
		{opGreaterEqual, typeInt, false},
		{opLess, typeInt, false},
		{opLessEqual, typeInt, false},
		{opPower, typeInt, false},
		{opAnd, typeInt, false},
		{opOr, typeInt, false},
		{opSpansetChild, typeInt, false},
		{opSpansetDescendant, typeInt, false},
		{opSpansetAnd, typeInt, false},
		{opSpansetUnion, typeInt, false},
		{opSpansetSibling, typeInt, false},
		// not
		{opNot, typeBoolean, true},
		{opNot, typeInt, false},
		{opNot, typeNil, false},
		{opNot, typeString, false},
		// sub
		{opSub, typeInt, true},
		{opSub, typeFloat, true},
		{opSub, typeDuration, true},
		{opSub, typeBoolean, false},
		{opSub, typeStatus, false},
		{opSub, typeNil, false},
		{opSub, typeSpanset, false},
	}

	for _, tc := range tt {
		t.Run(tc.op.String(), func(t *testing.T) {
			actual := tc.op.unaryTypesValid(tc.t)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
