package traceql

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatic_Equals(t *testing.T) {
	areEqual := []struct {
		lhs, rhs Static
	}{
		{NewStaticInt(1), NewStaticInt(1)},
		{NewStaticFloat(1.5), NewStaticFloat(1.5)},
		{NewStaticString("foo"), NewStaticString("foo")},
		{NewStaticBool(true), NewStaticBool(true)},
		{NewStaticDuration(1 * time.Second), NewStaticDuration(1000 * time.Millisecond)},
		{NewStaticStatus(StatusOk), NewStaticStatus(StatusOk)},
		// Status and int comparison
		{NewStaticStatus(StatusError), NewStaticInt(0)},
		{NewStaticStatus(StatusOk), NewStaticInt(1)},
		{NewStaticStatus(StatusUnset), NewStaticInt(2)},
	}
	areNotEqual := []struct {
		lhs, rhs Static
	}{
		{NewStaticInt(1), NewStaticInt(2)},
		{NewStaticInt(1), NewStaticFloat(1)},
		{NewStaticBool(true), NewStaticInt(1)},
		{NewStaticString("foo"), NewStaticString("bar")},
		{NewStaticDuration(0), NewStaticInt(0)},
		{NewStaticStatus(StatusError), NewStaticStatus(StatusOk)},
		{NewStaticStatus(StatusOk), NewStaticInt(0)},
		{NewStaticStatus(StatusError), NewStaticFloat(0)},
	}
	for _, tt := range areEqual {
		t.Run(fmt.Sprintf("%v == %v", tt.lhs, tt.rhs), func(t *testing.T) {
			assert.True(t, tt.lhs.Equals(tt.rhs))
			assert.True(t, tt.rhs.Equals(tt.lhs))
		})
	}
	for _, tt := range areNotEqual {
		t.Run(fmt.Sprintf("%v != %v", tt.lhs, tt.rhs), func(t *testing.T) {
			assert.False(t, tt.lhs.Equals(tt.rhs))
			assert.False(t, tt.rhs.Equals(tt.lhs))
		})
	}
}
