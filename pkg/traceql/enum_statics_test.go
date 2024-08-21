package traceql

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticType_isMatchingOperand(t *testing.T) {
	tests := []struct {
		t      StaticType
		otherT StaticType
		want   bool
	}{
		// same type on both sides
		{t: TypeNil, otherT: TypeNil, want: true},
		{t: TypeSpanset, otherT: TypeSpanset, want: true},
		{t: TypeAttribute, otherT: TypeAttribute, want: true},
		{t: TypeInt, otherT: TypeInt, want: true},
		{t: TypeFloat, otherT: TypeFloat, want: true},
		{t: TypeString, otherT: TypeString, want: true},
		{t: TypeBoolean, otherT: TypeBoolean, want: true},
		{t: TypeIntArray, otherT: TypeIntArray, want: true},
		{t: TypeFloatArray, otherT: TypeFloatArray, want: true},
		{t: TypeStringArray, otherT: TypeStringArray, want: true},
		{t: TypeBooleanArray, otherT: TypeBooleanArray, want: true},
		{t: TypeDuration, otherT: TypeDuration, want: true},
		{t: TypeStatus, otherT: TypeStatus, want: true},
		{t: TypeKind, otherT: TypeKind, want: true},

		// TypeAttribute with any other type
		{t: TypeAttribute, otherT: TypeNil, want: true},
		{t: TypeAttribute, otherT: TypeSpanset, want: true},
		{t: TypeAttribute, otherT: TypeInt, want: true},
		{t: TypeAttribute, otherT: TypeFloat, want: true},
		{t: TypeAttribute, otherT: TypeString, want: true},
		{t: TypeAttribute, otherT: TypeBoolean, want: true},
		{t: TypeAttribute, otherT: TypeIntArray, want: true},
		{t: TypeAttribute, otherT: TypeFloatArray, want: true},
		{t: TypeAttribute, otherT: TypeStringArray, want: true},
		{t: TypeAttribute, otherT: TypeBooleanArray, want: true},
		{t: TypeAttribute, otherT: TypeDuration, want: true},
		{t: TypeAttribute, otherT: TypeStatus, want: true},
		{t: TypeAttribute, otherT: TypeKind, want: true},

		// any type with TypeAttribute
		{t: TypeNil, otherT: TypeAttribute, want: true},
		{t: TypeSpanset, otherT: TypeAttribute, want: true},
		{t: TypeInt, otherT: TypeAttribute, want: true},
		{t: TypeFloat, otherT: TypeAttribute, want: true},
		{t: TypeString, otherT: TypeAttribute, want: true},
		{t: TypeBoolean, otherT: TypeAttribute, want: true},
		{t: TypeIntArray, otherT: TypeAttribute, want: true},
		{t: TypeFloatArray, otherT: TypeAttribute, want: true},
		{t: TypeStringArray, otherT: TypeAttribute, want: true},
		{t: TypeBooleanArray, otherT: TypeAttribute, want: true},
		{t: TypeDuration, otherT: TypeAttribute, want: true},
		{t: TypeStatus, otherT: TypeAttribute, want: true},
		{t: TypeKind, otherT: TypeAttribute, want: true},

		// both numeric
		{t: TypeInt, otherT: TypeFloat, want: true},
		{t: TypeFloat, otherT: TypeInt, want: true},
		{t: TypeNil, otherT: TypeIntArray, want: true},
		{t: TypeNil, otherT: TypeFloatArray, want: true},
		{t: TypeNil, otherT: TypeStringArray, want: true},
		{t: TypeNil, otherT: TypeBooleanArray, want: true},
		{t: TypeNil, otherT: TypeInt, want: false},
		{t: TypeNil, otherT: TypeFloat, want: false},
		{t: TypeNil, otherT: TypeString, want: false},
		{t: TypeNil, otherT: TypeBoolean, want: false},
		{t: TypeNil, otherT: TypeDuration, want: false},
		{t: TypeNil, otherT: TypeStatus, want: false},
		{t: TypeNil, otherT: TypeKind, want: false},

		// array types
		{t: TypeIntArray, otherT: TypeIntArray, want: true},
		{t: TypeIntArray, otherT: TypeFloatArray, want: true},
		{t: TypeFloatArray, otherT: TypeFloatArray, want: true},
		{t: TypeStringArray, otherT: TypeStringArray, want: true},
		{t: TypeBooleanArray, otherT: TypeBooleanArray, want: true},
		{t: TypeStringArray, otherT: TypeBooleanArray, want: false},
		{t: TypeIntArray, otherT: TypeStringArray, want: false},
		{t: TypeIntArray, otherT: TypeBooleanArray, want: false},
		{t: TypeFloatArray, otherT: TypeStringArray, want: false},
		{t: TypeFloatArray, otherT: TypeBooleanArray, want: false},

		// other edge cases
		{t: TypeInt, otherT: TypeString, want: false},
		{t: TypeBoolean, otherT: TypeString, want: false},
		{t: TypeDuration, otherT: TypeStatus, want: false},
		{t: TypeSpanset, otherT: TypeKind, want: false},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s with %s", tt.t, tt.otherT)
		t.Run(name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.t.isMatchingOperand(tt.otherT), "isMatchingOperand: %s", name)
		})
	}
}

func TestStaticType_isMatchingArrayElement(t *testing.T) {
	tests := []struct {
		t      StaticType
		otherT StaticType
		want   bool
	}{
		// IntArray cases
		{t: TypeIntArray, otherT: TypeInt, want: true},
		{t: TypeIntArray, otherT: TypeFloat, want: true},
		{t: TypeIntArray, otherT: TypeNil, want: true},
		{t: TypeIntArray, otherT: TypeAttribute, want: true},
		{t: TypeIntArray, otherT: TypeString, want: false},
		{t: TypeIntArray, otherT: TypeBoolean, want: false},

		// FloatArray cases
		{t: TypeFloatArray, otherT: TypeFloat, want: true},
		{t: TypeFloatArray, otherT: TypeInt, want: true},
		{t: TypeFloatArray, otherT: TypeNil, want: true},
		{t: TypeFloatArray, otherT: TypeAttribute, want: true},
		{t: TypeFloatArray, otherT: TypeString, want: false},
		{t: TypeFloatArray, otherT: TypeBoolean, want: false},

		// StringArray cases
		{t: TypeStringArray, otherT: TypeString, want: true},
		{t: TypeStringArray, otherT: TypeNil, want: true},
		{t: TypeStringArray, otherT: TypeAttribute, want: true},
		{t: TypeStringArray, otherT: TypeInt, want: false},
		{t: TypeStringArray, otherT: TypeFloat, want: false},
		{t: TypeStringArray, otherT: TypeBoolean, want: false},

		// BooleanArray cases
		{t: TypeBooleanArray, otherT: TypeBoolean, want: true},
		{t: TypeBooleanArray, otherT: TypeNil, want: true},
		{t: TypeBooleanArray, otherT: TypeAttribute, want: true},
		{t: TypeBooleanArray, otherT: TypeInt, want: false},
		{t: TypeBooleanArray, otherT: TypeFloat, want: false},
		{t: TypeBooleanArray, otherT: TypeString, want: false},

		// non array types on both sides
		{t: TypeInt, otherT: TypeInt, want: false},
		{t: TypeFloat, otherT: TypeFloat, want: false},
		{t: TypeString, otherT: TypeString, want: false},
		{t: TypeBoolean, otherT: TypeBoolean, want: false},
		{t: TypeNil, otherT: TypeBoolean, want: false},
		{t: TypeNil, otherT: TypeString, want: false},
		{t: TypeInt, otherT: TypeNil, want: false},
		{t: TypeFloat, otherT: TypeNil, want: false},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s with %s", tt.t, tt.otherT)
		t.Run(name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.t.isMatchingArrayElement(tt.otherT), "isMatchingArrayElement(%s)", name)
			// test symmetric case
			assert.Equalf(t, tt.want, tt.otherT.isMatchingArrayElement(tt.t), "isMatchingArrayElement(%s) [symmetric]", name)
		})
	}
}
