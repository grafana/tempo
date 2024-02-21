package intern

import (
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
