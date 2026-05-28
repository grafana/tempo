// Package atomicx provides atomic primitives that the standard library's
// sync/atomic does not ship typed wrappers for: Float64, Error, String.
//
// Other atomic types (Int32/Int64/Uint32/Uint64/Bool/Pointer) should use
// sync/atomic directly.
package atomicx

import (
	"math"
	"sync"
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

func (f *Float64) CompareAndSwap(old, new float64) bool {
	return f.v.CompareAndSwap(math.Float64bits(old), math.Float64bits(new))
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

// Error is an atomic error. Uses a mutex; intended for low-frequency error
// propagation paths (first-writer-wins, completion handoff, etc.) — not hot
// loops.
type Error struct {
	_  noCopy
	mu sync.RWMutex
	v  error
}

func NewError(err error) *Error {
	return &Error{v: err}
}

func (e *Error) Load() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.v
}

func (e *Error) Store(err error) {
	e.mu.Lock()
	e.v = err
	e.mu.Unlock()
}

func (e *Error) CompareAndSwap(old, new error) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.v == old {
		e.v = new
		return true
	}
	return false
}

// noCopy may be embedded into structs which must not be copied
// after the first use. Detected by `go vet`'s -copylocks check.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
