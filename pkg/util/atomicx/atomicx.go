// Package atomicx provides atomic primitives that the standard library's
// sync/atomic does not ship typed wrappers for: Float64, Error, String.
//
// Other atomic types (Int32/Int64/Uint32/Uint64/Bool/Pointer) should use
// sync/atomic directly.
package atomicx

import (
	"math"
	"sync/atomic"
)

// Float64 is an atomic float64 backed by an atomic.Uint64 via math.Float64bits.
type Float64 struct {
	_ noCopy
	v atomic.Uint64
}

func NewFloat64(v float64) *Float64 {
	f := &Float64{}
	f.Store(v)
	return f
}

func (f *Float64) Load() float64 { return math.Float64frombits(f.v.Load()) }

func (f *Float64) Store(v float64) { f.v.Store(math.Float64bits(v)) }

func (f *Float64) Add(delta float64) float64 {
	for {
		oldBits := f.v.Load()
		newVal := math.Float64frombits(oldBits) + delta
		if f.v.CompareAndSwap(oldBits, math.Float64bits(newVal)) {
			return newVal
		}
	}
}

func (f *Float64) Sub(delta float64) float64 { return f.Add(-delta) }

func (f *Float64) CompareAndSwap(old, next float64) bool {
	return f.v.CompareAndSwap(math.Float64bits(old), math.Float64bits(next))
}

// String is an atomic string backed by atomic.Pointer[string].
type String struct {
	_ noCopy
	v atomic.Pointer[string]
}

func NewString(s string) *String {
	x := &String{}
	x.Store(s)
	return x
}

func (s *String) Load() string {
	if p := s.v.Load(); p != nil {
		return *p
	}
	return ""
}

func (s *String) Store(v string) { s.v.Store(&v) }

type Error struct {
	_ noCopy
	v atomic.Value
}

type packedError struct{ Value error }

func NewError(err error) *Error {
	e := &Error{}
	if err != nil {
		e.Store(err)
	}
	return e
}

func (e *Error) Load() error {
	v := e.v.Load()
	if v == nil {
		return nil
	}
	return v.(packedError).Value
}

func (e *Error) Store(err error) { e.v.Store(packedError{err}) }

func (e *Error) CompareAndSwap(old, next error) bool {
	if e.v.CompareAndSwap(packedError{old}, packedError{next}) {
		return true
	}
	// Before any Store, atomic.Value's internal state is the nil interface
	// rather than a zero packedError, so the first CAS misses. Retry against nil.
	if old == nil {
		return e.v.CompareAndSwap(nil, packedError{next})
	}
	return false
}

// Constructors for stdlib atomic types initialized with a non-zero value.
// For zero-value init, use the stdlib type directly (e.g. &atomic.Int64{}).

func NewInt32(v int32) *atomic.Int32 {
	x := &atomic.Int32{}
	x.Store(v)
	return x
}

func NewInt64(v int64) *atomic.Int64 {
	x := &atomic.Int64{}
	x.Store(v)
	return x
}

func NewUint32(v uint32) *atomic.Uint32 {
	x := &atomic.Uint32{}
	x.Store(v)
	return x
}

func NewUint64(v uint64) *atomic.Uint64 {
	x := &atomic.Uint64{}
	x.Store(v)
	return x
}

func NewBool(v bool) *atomic.Bool {
	x := &atomic.Bool{}
	x.Store(v)
	return x
}

// noCopy may be embedded into structs which must not be copied
// after the first use. Detected by `go vet`'s -copylocks check.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
