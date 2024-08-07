package traceql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticType_isMatchingOperand(t *testing.T) {
	tests := []struct {
		name   string
		t      StaticType
		otherT StaticType
		want   bool
	}{
		// same type on both sides
		{name: "both TypeNil", t: TypeNil, otherT: TypeNil, want: true},
		{name: "both TypeSpanset", t: TypeSpanset, otherT: TypeSpanset, want: true},
		{name: "both TypeAttribute", t: TypeAttribute, otherT: TypeAttribute, want: true},
		{name: "both TypeInt", t: TypeInt, otherT: TypeInt, want: true},
		{name: "both TypeFloat", t: TypeFloat, otherT: TypeFloat, want: true},
		{name: "both TypeString", t: TypeString, otherT: TypeString, want: true},
		{name: "both TypeBoolean", t: TypeBoolean, otherT: TypeBoolean, want: true},
		{name: "both TypeIntArray", t: TypeIntArray, otherT: TypeIntArray, want: true},
		{name: "both TypeFloatArray", t: TypeFloatArray, otherT: TypeFloatArray, want: true},
		{name: "both TypeStringArray", t: TypeStringArray, otherT: TypeStringArray, want: true},
		{name: "both TypeBooleanArray", t: TypeBooleanArray, otherT: TypeBooleanArray, want: true},
		{name: "both TypeDuration", t: TypeDuration, otherT: TypeDuration, want: true},
		{name: "both TypeStatus", t: TypeStatus, otherT: TypeStatus, want: true},
		{name: "both TypeKind", t: TypeKind, otherT: TypeKind, want: true},

		// TypeAttribute with any other type
		{name: "TypeAttribute with TypeNil", t: TypeAttribute, otherT: TypeNil, want: true},
		{name: "TypeAttribute with TypeSpanset", t: TypeAttribute, otherT: TypeSpanset, want: true},
		{name: "TypeAttribute with TypeInt", t: TypeAttribute, otherT: TypeInt, want: true},
		{name: "TypeAttribute with TypeFloat", t: TypeAttribute, otherT: TypeFloat, want: true},
		{name: "TypeAttribute with TypeString", t: TypeAttribute, otherT: TypeString, want: true},
		{name: "TypeAttribute with TypeBoolean", t: TypeAttribute, otherT: TypeBoolean, want: true},
		{name: "TypeAttribute with TypeIntArray", t: TypeAttribute, otherT: TypeIntArray, want: true},
		{name: "TypeAttribute with TypeFloatArray", t: TypeAttribute, otherT: TypeFloatArray, want: true},
		{name: "TypeAttribute with TypeStringArray", t: TypeAttribute, otherT: TypeStringArray, want: true},
		{name: "TypeAttribute with TypeBooleanArray", t: TypeAttribute, otherT: TypeBooleanArray, want: true},
		{name: "TypeAttribute with TypeDuration", t: TypeAttribute, otherT: TypeDuration, want: true},
		{name: "TypeAttribute with TypeStatus", t: TypeAttribute, otherT: TypeStatus, want: true},
		{name: "TypeAttribute with TypeKind", t: TypeAttribute, otherT: TypeKind, want: true},

		// any type with TypeAttribute
		{name: "TypeNil with TypeAttribute", t: TypeNil, otherT: TypeAttribute, want: true},
		{name: "TypeSpanset with TypeAttribute", t: TypeSpanset, otherT: TypeAttribute, want: true},
		{name: "TypeInt with TypeAttribute", t: TypeInt, otherT: TypeAttribute, want: true},
		{name: "TypeFloat with TypeAttribute", t: TypeFloat, otherT: TypeAttribute, want: true},
		{name: "TypeString with TypeAttribute", t: TypeString, otherT: TypeAttribute, want: true},
		{name: "TypeBoolean with TypeAttribute", t: TypeBoolean, otherT: TypeAttribute, want: true},
		{name: "TypeIntArray with TypeAttribute", t: TypeIntArray, otherT: TypeAttribute, want: true},
		{name: "TypeFloatArray with TypeAttribute", t: TypeFloatArray, otherT: TypeAttribute, want: true},
		{name: "TypeStringArray with TypeAttribute", t: TypeStringArray, otherT: TypeAttribute, want: true},
		{name: "TypeBooleanArray with TypeAttribute", t: TypeBooleanArray, otherT: TypeAttribute, want: true},
		{name: "TypeDuration with TypeAttribute", t: TypeDuration, otherT: TypeAttribute, want: true},
		{name: "TypeStatus with TypeAttribute", t: TypeStatus, otherT: TypeAttribute, want: true},
		{name: "TypeKind with TypeAttribute", t: TypeKind, otherT: TypeAttribute, want: true},

		// both numeric
		{name: "both numeric TypeInt and TypeFloat", t: TypeInt, otherT: TypeFloat, want: true},
		{name: "both numeric TypeFloat and TypeInt", t: TypeFloat, otherT: TypeInt, want: true},

		// TypeNil with any type
		{name: "TypeNil with TypeSpanset", t: TypeNil, otherT: TypeSpanset, want: false},
		{name: "TypeNil with TypeInt", t: TypeNil, otherT: TypeInt, want: false},
		{name: "TypeNil with TypeFloat", t: TypeNil, otherT: TypeFloat, want: false},
		{name: "TypeNil with TypeString", t: TypeNil, otherT: TypeString, want: false},
		{name: "TypeNil with TypeBoolean", t: TypeNil, otherT: TypeBoolean, want: false},
		{name: "TypeNil with TypeIntArray", t: TypeNil, otherT: TypeIntArray, want: false},
		{name: "TypeNil with TypeFloatArray", t: TypeNil, otherT: TypeFloatArray, want: false},
		{name: "TypeNil with TypeStringArray", t: TypeNil, otherT: TypeStringArray, want: false},
		{name: "TypeNil with TypeBooleanArray", t: TypeNil, otherT: TypeBooleanArray, want: false},
		{name: "TypeNil with TypeDuration", t: TypeNil, otherT: TypeDuration, want: false},
		{name: "TypeNil with TypeStatus", t: TypeNil, otherT: TypeStatus, want: false},
		{name: "TypeNil with TypeKind", t: TypeNil, otherT: TypeKind, want: false},

		// other edge cases
		{name: "TypeInt with TypeString", t: TypeInt, otherT: TypeString, want: false},
		{name: "TypeBoolean with TypeString", t: TypeBoolean, otherT: TypeString, want: false},
		{name: "TypeIntArray with TypeFloatArray", t: TypeIntArray, otherT: TypeFloatArray, want: false},
		{name: "TypeDuration with TypeStatus", t: TypeDuration, otherT: TypeStatus, want: false},
		{name: "TypeSpanset with TypeKind", t: TypeSpanset, otherT: TypeKind, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.t.isMatchingOperand(tt.otherT), "isMatchingOperand(%v)", tt.otherT)
		})
	}
}

func TestStaticType_isMatchingArrayElement(t *testing.T) {
	tests := []struct {
		name   string
		t      StaticType
		otherT StaticType
		want   bool
	}{
		// IntArray cases
		{name: "IntArray with TypeInt", t: TypeIntArray, otherT: TypeInt, want: true},
		{name: "IntArray with TypeFloat", t: TypeIntArray, otherT: TypeFloat, want: true},
		{name: "IntArray with TypeString", t: TypeIntArray, otherT: TypeString, want: false},
		{name: "IntArray with TypeBoolean", t: TypeIntArray, otherT: TypeBoolean, want: false},
		{name: "IntArray with TypeNil", t: TypeIntArray, otherT: TypeNil, want: true},
		{name: "IntArray with TypeAttribute", t: TypeIntArray, otherT: TypeAttribute, want: true},

		// FloatArray cases
		{name: "FloatArray with TypeFloat", t: TypeFloatArray, otherT: TypeFloat, want: true},
		{name: "FloatArray with TypeInt", t: TypeFloatArray, otherT: TypeInt, want: true},
		{name: "FloatArray with TypeString", t: TypeFloatArray, otherT: TypeString, want: false},
		{name: "FloatArray with TypeBoolean", t: TypeFloatArray, otherT: TypeBoolean, want: false},
		{name: "FloatArray with TypeNil", t: TypeFloatArray, otherT: TypeNil, want: true},
		{name: "FloatArray with TypeAttribute", t: TypeFloatArray, otherT: TypeAttribute, want: true},

		// StringArray cases
		{name: "StringArray with TypeString", t: TypeStringArray, otherT: TypeString, want: true},
		{name: "StringArray with TypeInt", t: TypeStringArray, otherT: TypeInt, want: false},
		{name: "StringArray with TypeFloat", t: TypeStringArray, otherT: TypeFloat, want: false},
		{name: "StringArray with TypeBoolean", t: TypeStringArray, otherT: TypeBoolean, want: false},
		{name: "StringArray with TypeNil", t: TypeStringArray, otherT: TypeNil, want: true},
		{name: "StringArray with TypeAttribute", t: TypeStringArray, otherT: TypeAttribute, want: true},

		// BooleanArray cases
		{name: "BooleanArray with TypeBoolean", t: TypeBooleanArray, otherT: TypeBoolean, want: true},
		{name: "BooleanArray with TypeInt", t: TypeBooleanArray, otherT: TypeInt, want: false},
		{name: "BooleanArray with TypeFloat", t: TypeBooleanArray, otherT: TypeFloat, want: false},
		{name: "BooleanArray with TypeString", t: TypeBooleanArray, otherT: TypeString, want: false},
		{name: "BooleanArray with TypeNil", t: TypeBooleanArray, otherT: TypeNil, want: true},
		{name: "BooleanArray with TypeAttribute", t: TypeBooleanArray, otherT: TypeAttribute, want: true},

		// non array types
		{name: "TypeInt with TypeInt", t: TypeInt, otherT: TypeInt, want: false},
		{name: "TypeFloat with TypeFloat", t: TypeFloat, otherT: TypeFloat, want: false},
		{name: "TypeString with TypeString", t: TypeString, otherT: TypeString, want: false},
		{name: "TypeBoolean with TypeBoolean", t: TypeBoolean, otherT: TypeBoolean, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.t.isMatchingArrayElement(tt.otherT), "isMatchingArrayElement(%v)", tt.otherT)
		})
	}
}
