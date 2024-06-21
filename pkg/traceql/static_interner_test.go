package traceql

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestStaticInterner_StaticInt(t *testing.T) {
	interner := NewStaticInterner()

	tests := []int{-maxGlobalInt - 1, -maxGlobalInt, -1, 0, 1, 2, maxGlobalInt, maxGlobalInt + 1}

	for _, n := range tests {
		var a, b, c Static
		a = NewStaticInt(n)
		b = interner.StaticInt(n)
		c = interner.StaticInt(n) // interned version returned in second call

		assert.Equal(t, a, b)
		assert.Equal(t, b, c)
	}
}

func TestStaticInterner_StaticFloat(t *testing.T) {
	interner := NewStaticInterner()

	tests := []float64{-3.1415, -1, 0, 1, 3117.4}

	for _, f := range tests {
		var a, b, c Static
		a = NewStaticFloat(f)
		b = interner.StaticFloat(f)
		c = interner.StaticFloat(f) // interned version returned in second call

		assert.Equal(t, a, b)
		assert.Equal(t, b, c)
	}
}

func TestStaticInterner_StaticKind(t *testing.T) {
	interner := NewStaticInterner()

	for k := range globalKindCount {
		var a, b Static
		a = NewStaticKind(Kind(k))
		b = interner.StaticKind(Kind(k))

		assert.Equal(t, a, b)
	}
}

func TestStaticInterner_StaticStatus(t *testing.T) {
	interner := NewStaticInterner()

	for k := range globalStatusCount {
		var a, b Static
		a = NewStaticStatus(Status(k))
		b = interner.StaticStatus(Status(k))

		assert.Equal(t, a, b)
	}
}

func TestStaticInterner_StaticString(t *testing.T) {
	interner := NewStaticInterner()

	tests := []string{
		"",
		"foo",
		randomStr(maxInternStrLen - 1),
		randomStr(maxInternStrLen),
		randomStr(maxInternStrLen + 1),
	}

	for _, n := range tests {
		var a, b Static
		a = NewStaticString(n)
		b = interner.StaticString(n)

		assert.Equal(t, a, b)
	}
}

func TestStaticInterner_StaticBool(t *testing.T) {
	interner := NewStaticInterner()

	assert.Equal(t, NewStaticBool(true), interner.StaticBool(true))
	assert.Equal(t, NewStaticBool(false), interner.StaticBool(false))
}

func randomStr(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
