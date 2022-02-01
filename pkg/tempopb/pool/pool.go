// Forked with love from: https://github.com/prometheus/prometheus/tree/c954cd9d1d4e3530be2939d39d8633c38b70913f/util/pool
// This package was forked to provide better protection against putting byte slices back into the pool that
// did not originate from it.

package pool

import (
	"sync"
)

// Pool is a bucketed pool for variably sized byte slices.
type Pool struct {
	buckets []sync.Pool
	sizes   []int
	// make is the function used to create an empty slice when none exist yet.
	make func(int) []byte
}

// New returns a new Pool with size buckets for minSize to maxSize
// increasing by the given factor.
func New(minSize, maxSize int, factor float64, makeFunc func(int) []byte) *Pool {
	if minSize < 1 {
		panic("invalid minimum pool size")
	}
	if maxSize < 1 {
		panic("invalid maximum pool size")
	}
	if factor < 1 {
		panic("invalid factor")
	}

	var sizes []int

	for s := minSize; s <= maxSize; s = int(float64(s) * factor) {
		sizes = append(sizes, s)
	}

	p := &Pool{
		buckets: make([]sync.Pool, len(sizes)),
		sizes:   sizes,
		make:    makeFunc,
	}

	return p
}

// Get returns a new byte slices that fits the given size.
func (p *Pool) Get(sz int) []byte {
	for i, bktSize := range p.sizes {
		if sz > bktSize {
			continue
		}
		b := p.buckets[i].Get()
		if b == nil {
			b = p.make(bktSize)
		}
		return b.([]byte)
	}
	return p.make(sz)
}

// Put adds a slice to the right bucket in the pool. This method has been adjusted from its initial
// implementation to ignore byte slices that dont have the correct size
func (p *Pool) Put(s []byte) {
	c := cap(s)
	for i, size := range p.sizes {
		if c == size {
			p.buckets[i].Put(s) // nolint: staticcheck
		}
		if c < size {
			return
		}
	}
}
