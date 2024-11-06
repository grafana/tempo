// Package intern is a utility for interning byte slices for pq.Value's.
//
// The Interner is used to intern byte slices for pq.Value's. This is useful
// for reducing memory usage and improving performance when working with
// large datasets with many repeated strings.
package intern

import (
	"sync"
	"unique"

	pq "github.com/parquet-go/parquet-go"
)

type Interner struct {
	h   map[unique.Handle[*pq.Value]]pq.Value
	mtx sync.Mutex
}

func New() *Interner {
	return &Interner{
		mtx: sync.Mutex{},
		h:   make(map[unique.Handle[*pq.Value]]pq.Value),
	}
}

// Clone returns a unique shalow copy of the input pq.Value or derefernces the
// received pointer.
func (i *Interner) Clone(v *pq.Value) pq.Value {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	switch v.Kind() {
	case pq.ByteArray, pq.FixedLenByteArray:
		if vv, ok := i.h[unique.Make(v)]; ok {
			return vv
		}
		vv := unique.Make(v)
		i.h[vv] = *vv.Value()
		return i.h[vv]
	default:
		return *v
	}
}

func (i *Interner) Close() {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	clear(i.h)
}
