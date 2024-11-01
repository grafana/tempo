// Package intern is a utility for interning byte slices for pq.Value's.
// It is not safe for concurrent use.
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

func (i *Interner) UnsafeClone(v *pq.Value) pq.Value {
	switch v.Kind() {
	case pq.ByteArray, pq.FixedLenByteArray:
		return *unique.Make(v).Value()
	default:
		return *v
	}
}
