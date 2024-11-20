// Forked with love from: https://github.com/prometheus/prometheus/tree/c954cd9d1d4e3530be2939d39d8633c38b70913f/util/pool
// This package was forked to provide better protection against putting byte slices back into the pool that
// did not originate from it.

package pool

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metricMissOver prometheus.Counter
var metricMissUnder prometheus.Counter

func init() {
	metricAllocOutPool := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_prealloc_miss_bytes_total",
		Help:      "The total number of alloc'ed bytes that missed the sync pools.",
	}, []string{"direction"})

	metricMissOver = metricAllocOutPool.WithLabelValues("over")
	metricMissUnder = metricAllocOutPool.WithLabelValues("under")
}

// Pool is a linearly bucketed pool for variably sized byte slices.
type Pool struct {
	buckets   []sync.Pool
	bktSize   int
	minBucket int

	// make is the function used to create an empty slice when none exist yet.
	make func(int) []byte
}

// New returns a new Pool with size buckets for minSize to maxSize
func New(minBucket, numBuckets, bktSize int, makeFunc func(int) []byte) *Pool {
	if minBucket < 0 {
		panic("invalid min bucket size")
	}
	if bktSize < 1 {
		panic("invalid bucket size")
	}
	if numBuckets < 1 {
		panic("invalid num buckets")
	}

	return &Pool{
		buckets:   make([]sync.Pool, numBuckets),
		bktSize:   bktSize,
		minBucket: minBucket,
		make:      makeFunc,
	}
}

// Get returns a new byte slices that fits the given size.
func (p *Pool) Get(sz int) []byte {
	if sz < 0 {
		panic("requested negative size")
	}

	if sz < p.minBucket {
		metricMissUnder.Add(float64(sz))
		return p.make(sz)
	}

	// Find the right bucket.
	bkt := p.bucketFor(sz)

	if bkt >= len(p.buckets) {
		metricMissOver.Add(float64(sz))
		return p.make(sz)
	}

	b := p.buckets[bkt].Get()
	if b == nil {
		alignedSz := ((sz / p.bktSize) + 1) * p.bktSize // align to the next bucket up
		b = p.make(alignedSz)
	}
	return b.([]byte)
}

// Put adds a slice to the right bucket in the pool.
func (p *Pool) Put(s []byte) int {
	c := cap(s)

	// valid slice?
	if c%p.bktSize != 0 {
		return -1
	}
	bkt := p.bucketFor(c) - 1 // -1 puts the slice in the pool below. it will be larger than all requested slices for this bucket
	if bkt < 0 {
		return -1
	}
	if bkt >= len(p.buckets) {
		return -1
	}

	p.buckets[bkt].Put(s) // nolint: staticcheck

	return bkt
}

func (p *Pool) bucketFor(sz int) int {
	return (sz - p.minBucket) / p.bktSize
}
