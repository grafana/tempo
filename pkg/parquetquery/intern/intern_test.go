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
	interned1, interned2 := i.internBytes([]byte("hello")), i.internBytes([]byte("hello"))
	if unsafe.SliceData(interned1) != unsafe.SliceData(interned2) {
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

	clone1Addr := unsafe.SliceData(clone1.ByteArray())
	clone2Addr := unsafe.SliceData(clone2.ByteArray())
	if clone1Addr != clone2Addr {
		t.Errorf("expected interned values to have same memory address, got %p and %p", clone1Addr, clone2Addr)
	}

	value1Addr := unsafe.SliceData(value1.ByteArray())
	value2Addr := unsafe.SliceData(value2.ByteArray())
	if value1Addr == value2Addr {
		t.Error("expected original values to have different memory addresses")
	}

	if string(clone1.ByteArray()) != string(clone2.ByteArray()) {
		t.Error("expected same byte values")
	}
	if string(value1.ByteArray()) != string(value2.ByteArray()) {
		t.Error("expected original values to have same content")
	}

	clone3 := i.UnsafeClone(&value1) // Clone the same value again
	clone3Addr := unsafe.SliceData(clone3.ByteArray())
	if clone1Addr != clone3Addr {
		t.Errorf("expected repeated interning to return same memory, got %p and %p", clone1Addr, clone3Addr)
	}

	differentValue := pq.ByteArrayValue([]byte("bar"))
	differentClone := i.UnsafeClone(&differentValue)
	differentAddr := unsafe.SliceData(differentClone.ByteArray())
	if clone1Addr == differentAddr {
		t.Error("expected different strings to have different interned memory")
	}
}

func TestPqValueMemoryLayout(t *testing.T) {
	var pqVal pq.Value
	var localVal pqValue

	pqSize := unsafe.Sizeof(pqVal)
	localSize := unsafe.Sizeof(localVal)

	if pqSize != localSize {
		t.Errorf("struct size mismatch: pq.Value=%d bytes, pqValue=%d bytes. "+
			"parquet-go may have changed pq.Value internal structure", pqSize, localSize)
	}

	pqAlign := unsafe.Alignof(pqVal)
	localAlign := unsafe.Alignof(localVal)

	if pqAlign != localAlign {
		t.Errorf("struct alignment mismatch: pq.Value=%d, pqValue=%d. "+
			"Memory layout assumptions may be invalid", pqAlign, localAlign)
	}
}

func TestPqValueConversion(t *testing.T) {
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
