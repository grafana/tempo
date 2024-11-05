// Package intern is a utility for interning byte slices for pq.Value's.
//
// The Interner is used to intern byte slices for pq.Value's. This is useful
// for reducing memory usage and improving performance when working with
// large datasets with many repeated strings.
package intern

import (
	"unique"

	pq "github.com/parquet-go/parquet-go"
)

type Interner struct{}

func New() *Interner {
	return &Interner{}
}

// Clone returns a unique shalow copy of the input pq.Value or derefernces the
// received pointer.
func (i *Interner) Clone(v *pq.Value) pq.Value {
	switch v.Kind() {
	case pq.ByteArray, pq.FixedLenByteArray:
		return *unique.Make(v).Value()
	default:
		return *v
	}
}
