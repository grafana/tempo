package intern

import (
	"fmt"
	"testing"
	"unsafe"

	pq "github.com/parquet-go/parquet-go"
)

func TestInterner_internBytes(t *testing.T) {
	i := New()
	defer i.Close()

	words := []string{"hello", "world", "hello", "world", "hello", "world"}
	for _, w := range words {
		_ = i.internBytes([]byte(w))
	}
	if len(i.m) != 2 {
		// Values are interned, so there should be only 2 unique words
		t.Errorf("expected 2, got %d", len(i.m))
	}
	if i.internBytes([]byte("hello"))[0] != i.internBytes([]byte("hello"))[0] {
		// Values are interned, so the memory address should be the same
		t.Error("expected same memory address")
	}
}

func TestInterner_UnsafeClone(t *testing.T) {
	i := New()
	defer i.Close()

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

func Test_pqValue(t *testing.T) {
	// Test that conversion from pq.Value to pqValue and back to pq.Value
	// does not change the value.
	value := pq.ByteArrayValue([]byte("foo"))
	pqValue := *(*pqValue)(unsafe.Pointer(&value))
	back := *(*pq.Value)(unsafe.Pointer(&pqValue))

	if value.Kind() != back.Kind() {
		t.Error("expected same kind")
	}

	if string(value.ByteArray()) != string(back.ByteArray()) {
		t.Error("expected same value")
	}

	if value.String() != back.String() {
		t.Error("expected same value")
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
			defer interner.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				value := tc.valueFn(i)
				_ = interner.UnsafeClone(&value)
			}
		})
	}
}
