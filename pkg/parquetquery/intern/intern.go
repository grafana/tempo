// Package intern is a utility for interning byte slices for pq.Value's.
// It is not safe for concurrent use.
//
// The Interner is used to intern byte slices for pq.Value's. This is useful
// for reducing memory usage and improving performance when working with
// large datasets with many repeated strings.
package intern

import (
	"unsafe"

	pq "github.com/parquet-go/parquet-go"
)

type Interner struct {
	m map[string][]byte // TODO(mapno): Use swiss.Map (https://github.com/cockroachdb/swiss)
}

func New() *Interner {
	return NewWithSize(0)
}

func NewWithSize(size int) *Interner {
	return &Interner{m: make(map[string][]byte, size)}
}

func (i *Interner) UnsafeClone(v *pq.Value) pq.Value {
	switch v.Kind() {
	case pq.ByteArray, pq.FixedLenByteArray:
		// Look away, this is unsafe.
		a := *(*pqValue)(unsafe.Pointer(v))
		a.ptr = addressOfBytes(i.internBytes(a.byteArray()))
		return *(*pq.Value)(unsafe.Pointer(&a))
	default:
		return *v
	}
}

func (i *Interner) internBytes(b []byte) []byte {
	if x, ok := i.m[bytesToString(b)]; ok {
		return x
	}

	clone := make([]byte, len(b))
	copy(clone, b)
	i.m[bytesToString(clone)] = clone
	return clone
}

func (i *Interner) Close() {
	clear(i.m) // clear the map
	i.m = nil
}

// bytesToString converts a byte slice to a string.
// String shares the memory with the byte slice.
// The byte slice should not be modified after call.
func bytesToString(b []byte) string { return unsafe.String(unsafe.SliceData(b), len(b)) }

// addressOfBytes returns the address of the first byte in data.
// The data should not be modified after call.
func addressOfBytes(data []byte) *byte { return unsafe.SliceData(data) }

// bytes converts a pointer to a slice of bytes
func bytes(data *byte, size int) []byte { return unsafe.Slice(data, size) }

// pqValue is a slimmer version of github.com/parquet-go/parquet-go's pq.Value.
type pqValue struct {
	// data
	ptr *byte
	u64 uint64
	// type
	kind int8 // XOR(Kind) so the zero-value is <null>
}

func (v *pqValue) byteArray() []byte {
	return bytes(v.ptr, int(v.u64))
}
