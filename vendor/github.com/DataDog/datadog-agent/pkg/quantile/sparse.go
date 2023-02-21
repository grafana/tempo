// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package quantile

import (
	"math"
	"strings"
	"unsafe"

	"github.com/DataDog/datadog-agent/pkg/quantile/summary"
)

var _ memSized = (*Sketch)(nil)

// A Sketch for tracking quantiles
// The serialized JSON of Sketch contains the summary only
// Bins are not included.
type Sketch struct {
	sparseStore

	Basic summary.Summary `json:"summary"`
}

func (s *Sketch) String() string {
	var b strings.Builder
	printSketch(&b, s, Default())
	return b.String()
}

// MemSize returns memory use in bytes:
//
//	used: uses len(bins)
//	allocated: uses cap(bins)
func (s *Sketch) MemSize() (used, allocated int) {
	const (
		basicSize = int(unsafe.Sizeof(summary.Summary{}))
	)

	used, allocated = s.sparseStore.MemSize()
	used += basicSize
	allocated += basicSize
	return
}

// InsertMany values into the sketch.
func (s *Sketch) InsertMany(c *Config, values []float64) {
	keys := getKeyList()

	for _, v := range values {
		s.Basic.Insert(v)
		keys = append(keys, c.key(v))
	}

	s.insert(c, keys)
	putKeyList(keys)
}

// Reset sketch to its empty state.
func (s *Sketch) Reset() {
	s.Basic.Reset()
	s.count = 0
	s.bins = s.bins[:0] // TODO: just release to a size tiered pool.
}

// GetRawBins return raw bins information as string
func (s *Sketch) GetRawBins() (int, string) {
	return s.count, strings.Replace(s.bins.String(), "\n", "", -1)
}

// Insert a single value into the sketch.
// NOTE: InsertMany is much more efficient.
func (s *Sketch) Insert(c *Config, vals ...float64) {
	// TODO: remove this
	s.InsertMany(c, vals)
}

// Merge o into s, without mutating o.
func (s *Sketch) Merge(c *Config, o *Sketch) {
	s.Basic.Merge(o.Basic)
	s.merge(c, &o.sparseStore)
}

// Quantile returns v such that s.count*q items are <= v.
//
// Special cases are:
//
//		Quantile(c, q <= 0)  = min
//	 Quantile(c, q >= 1)  = max
func (s *Sketch) Quantile(c *Config, q float64) float64 {
	switch {
	case s.count == 0:
		return 0
	case q <= 0:
		return s.Basic.Min
	case q >= 1:
		return s.Basic.Max
	}

	var (
		n     float64
		rWant = rank(s.count, q)
	)

	for i, b := range s.bins {
		n += float64(b.n)
		if n <= rWant {
			continue
		}

		weight := (n - rWant) / float64(b.n)

		vLow := c.f64(b.k)
		vHigh := vLow * c.gamma.v

		switch i {
		case s.bins.Len():
			vHigh = s.Basic.Max
		case 0:
			vLow = s.Basic.Min
		}

		// TODO|PROD: Interpolate between bucket boundaries, correctly handling min, max,
		// negative numbers.
		// with a gamma of 1.02, interpolating to the center gives us a 1% abs
		// error bound.
		return (vLow*weight + vHigh*(1-weight))
		// return vLow
	}

	// this should never happen
	return math.NaN()
}

func rank(count int, q float64) float64 {
	return math.RoundToEven(q * float64(count-1))
}

// CopyTo makes a deep copy of this sketch into dst.
func (s *Sketch) CopyTo(dst *Sketch) {
	// TODO: pool slices here?
	dst.bins = dst.bins.ensureLen(s.bins.Len())
	copy(dst.bins, s.bins)
	dst.count = s.count
	dst.Basic = s.Basic
}

// Copy returns a deep copy
func (s *Sketch) Copy() *Sketch {
	dst := &Sketch{}
	s.CopyTo(dst)
	return dst
}

// Equals returns true if s and o are equivalent.
func (s *Sketch) Equals(o *Sketch) bool {
	if s.Basic != o.Basic {
		return false
	}

	if s.count != o.count {
		return false
	}

	if len(s.bins) != len(o.bins) {
		return false
	}

	for i := range s.bins {
		if o.bins[i] != s.bins[i] {
			return false
		}
	}

	return true
}

// ApproxEquals checks if s and o are equivalent, with e error allowed for Sum and Average
func (s *Sketch) ApproxEquals(o *Sketch, e float64) bool {
	if math.Abs(s.Basic.Sum-o.Basic.Sum) > e {
		return false
	}

	if math.Abs(s.Basic.Avg-o.Basic.Avg) > e {
		return false
	}

	if s.Basic.Min != o.Basic.Min {
		return false
	}

	if s.Basic.Max != o.Basic.Max {
		return false
	}

	if s.Basic.Cnt != o.Basic.Cnt {
		return false
	}

	if s.count != o.count {
		return false
	}

	if len(s.bins) != len(o.bins) {
		return false
	}

	for i := range s.bins {
		if o.bins[i] != s.bins[i] {
			return false
		}
	}

	return true
}
