// Forked with love from: https://github.com/prometheus/prometheus/tree/c954cd9d1d4e3530be2939d39d8633c38b70913f/util/pool
// This package was forked to provide better protection against putting byte slices back into the pool that
// did not originate from it.

package pool

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricAllocOutPool = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_prealloc_miss_bytes_total",
		Help:      "The total number of alloc'ed bytes that missed the sync pools.",
	})
)

// Pool is a linearly bucketed pool for variably sized byte slices.
type Pool struct {
	buckets []sync.Pool
	bktSize int
	// make is the function used to create an empty slice when none exist yet.
	make func(int) []byte
}

// New returns a new Pool with size buckets for minSize to maxSize
func New(maxSize, bktSize int, makeFunc func(int) []byte) *Pool {
	if maxSize < 1 {
		panic("invalid maximum pool size")
	}
	if bktSize < 1 {
		panic("invalid factor")
	}
	if maxSize%bktSize != 0 {
		panic("invalid bucket size")
	}

	bkts := maxSize / bktSize

	p := &Pool{
		buckets: make([]sync.Pool, bkts),
		bktSize: bktSize,
		make:    makeFunc,
	}

	return p
}

// Get returns a new byte slices that fits the given size.
func (p *Pool) Get(sz int) []byte {
	if sz < 0 {
		sz = 0 // just panic?
	}

	// Find the right bucket.
	bkt := sz / p.bktSize

	if bkt >= len(p.buckets) {
		metricAllocOutPool.Add(float64(sz)) // track the number of bytes alloc'ed outside the pool for future tuning
		return p.make(sz)
	}

	b := p.buckets[bkt].Get()
	if b == nil {
		sz := (bkt + 1) * p.bktSize
		b = p.make(sz)
	}
	return b.([]byte)
}

// Put adds a slice to the right bucket in the pool.
func (p *Pool) Put(s []byte) {
	c := cap(s)

	if c%p.bktSize != 0 {
		return
	}
	bkt := (c / p.bktSize) - 1
	if bkt < 0 {
		return
	}
	if bkt >= len(p.buckets) {
		return
	}

	p.buckets[bkt].Put(s) // nolint: staticcheck
}
