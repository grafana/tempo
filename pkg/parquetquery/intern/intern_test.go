package intern

import (
	"fmt"
	"testing"

	pq "github.com/parquet-go/parquet-go"
)

func TestInterner_UnsafeClone(t *testing.T) {
	i := New()

	value1 := pq.ByteArrayValue([]byte("foo"))
	value2 := pq.ByteArrayValue([]byte("foo"))

	clone1 := i.UnsafeClone(&value1)
	clone2 := i.UnsafeClone(&value2)

	if clone1.ByteArray()[0] != clone2.ByteArray()[0] {
		// Values are interned, so the memory address should be the same
		t.Error("expected same memory address")
	}

	if value1.ByteArray()[0] != value2.ByteArray()[0] {
		// Mutates the original value, so the memory address should be different as well
		t.Error("expected same memory address")
	}
}

func BenchmarkIntern(b *testing.B) {
	words := []string{"foo", "bar", "baz", "qux", "quux", "corge", "grault", "garply", "waldo", "fred", "plugh", "xyzzy", "thud"}
	testCases := []struct {
		name    string
		valueFn func(i int) pq.Value
	}{
		{
			name:    "byte_array",
			valueFn: func(i int) pq.Value { return pq.ByteArrayValue([]byte(words[i%len(words)])) },
		},
		{
			name:    "fixed_len_byte_array",
			valueFn: func(i int) pq.Value { return pq.FixedLenByteArrayValue([]byte(words[i%len(words)])) },
		},
		{
			name:    "bool",
			valueFn: func(i int) pq.Value { return pq.BooleanValue(i%2 == 0) },
		},
		{
			name:    "int32",
			valueFn: func(i int) pq.Value { return pq.Int32Value(int32(i)) },
		},
	}

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("no_interning: %s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				value := tc.valueFn(i)
				_ = value.Clone()
			}
		})

		b.Run(fmt.Sprintf("interning: %s", tc.name), func(b *testing.B) {
			interner := New()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				value := tc.valueFn(i)
				_ = interner.UnsafeClone(&value)
			}
		})
	}
}
