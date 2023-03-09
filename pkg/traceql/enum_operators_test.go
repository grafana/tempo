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
		{OpAdd, TypeInt, true},
		{OpDiv, TypeDuration, true},
		{OpMod, TypeFloat, true},
		{OpMult, TypeInt, true},
		{OpPower, TypeDuration, true},
		{OpSub, TypeAttribute, true},

		{OpAdd, TypeString, false},
		{OpDiv, TypeSpanset, false},
		{OpMod, TypeStatus, false},
		{OpMult, TypeNil, false},
		{OpPower, TypeBoolean, false},
		{OpMult, TypeKind, false},
		// equality
		{OpEqual, TypeDuration, true},
		{OpNotEqual, TypeStatus, true},
		{OpEqual, TypeString, true},
		{OpNotEqual, TypeInt, true},
		{OpEqual, TypeNil, true},
		{OpNotEqual, TypeAttribute, true},
		{OpEqual, TypeBoolean, true},
		{OpNotEqual, TypeFloat, true},
		{OpEqual, TypeKind, true},
		{OpNotEqual, TypeKind, true},

		{OpEqual, TypeSpanset, false},
		// range comparison
		{OpGreater, TypeInt, true},
		{OpGreaterEqual, TypeFloat, true},
		{OpLess, TypeFloat, true},
		{OpLessEqual, TypeDuration, true},

		{OpGreater, TypeStatus, false},
		{OpLessEqual, TypeKind, false},
		{OpGreaterEqual, TypeNil, false},
		{OpLess, TypeString, false},
		{OpLessEqual, TypeBoolean, false},
		// string comparison
		{OpRegex, TypeString, true},
		{OpNotRegex, TypeAttribute, true},
		{OpRegex, TypeString, true},

		{OpRegex, TypeKind, false},
		{OpRegex, TypeInt, false},
		{OpNotRegex, TypeInt, false},
		// boolean
		{OpAnd, TypeBoolean, true},
		{OpOr, TypeAttribute, true},
		{OpAnd, TypeAttribute, true},

		{OpAnd, TypeDuration, false},
		{OpOr, TypeDuration, false},
		// not
		{OpNot, TypeBoolean, false},
	}

	for _, tc := range tt {
		t.Run(tc.op.String(), func(t *testing.T) {
			actual := tc.op.binaryTypesValid(tc.t, TypeAttribute)
			assert.Equal(t, tc.expected, actual)
			actual = tc.op.binaryTypesValid(TypeAttribute, tc.t)
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
		{OpAdd, TypeInt, false},
		{OpDiv, TypeInt, false},
		{OpMod, TypeInt, false},
		{OpMult, TypeInt, false},
		{OpEqual, TypeInt, false},
		{OpNotEqual, TypeInt, false},
		{OpRegex, TypeInt, false},
		{OpNotRegex, TypeInt, false},
		{OpGreater, TypeInt, false},
		{OpGreaterEqual, TypeInt, false},
		{OpLess, TypeInt, false},
		{OpLessEqual, TypeInt, false},
		{OpPower, TypeInt, false},
		{OpAnd, TypeInt, false},
		{OpOr, TypeInt, false},
		{OpSpansetChild, TypeInt, false},
		{OpSpansetDescendant, TypeInt, false},
		{OpSpansetAnd, TypeInt, false},
		{OpSpansetUnion, TypeInt, false},
		{OpSpansetSibling, TypeInt, false},
		// not
		{OpNot, TypeBoolean, true},
		{OpNot, TypeInt, false},
		{OpNot, TypeNil, false},
		{OpNot, TypeString, false},
		// sub
		{OpSub, TypeInt, true},
		{OpSub, TypeFloat, true},
		{OpSub, TypeDuration, true},
		{OpSub, TypeBoolean, false},
		{OpSub, TypeStatus, false},
		{OpSub, TypeNil, false},
		{OpSub, TypeSpanset, false},
	}

	for _, tc := range tt {
		t.Run(tc.op.String(), func(t *testing.T) {
			actual := tc.op.unaryTypesValid(tc.t)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
