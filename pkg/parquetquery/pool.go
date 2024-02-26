package parquetquery

import (
	"sync"

	"github.com/parquet-go/parquet-go"
)

var DefaultResultPool = NewResultPool(10)

type (
	PoolFn    func() *IteratorResult
	ReleaseFn func(*IteratorResult)
)

type ResultPool struct {
	pool *sync.Pool
	cap  int
}

// NewResultPool creates a pool for reusing IteratorResults. New items are created
// with the given default capacity.  Using different pools is helpful to keep
// items of similar sizes together which reduces slice allocations.
func NewResultPool(defaultCapacity int) *ResultPool {
	return &ResultPool{
		pool: &sync.Pool{},
		cap:  defaultCapacity,
	}
}

func (p *ResultPool) Get() *IteratorResult {
	x := p.pool.Get()
	if x == nil {
		return &IteratorResult{
			Entries: make([]struct {
				Key   string
				Value parquet.Value
			}, 0, p.cap),
			OtherEntries: make([]struct {
				Key   string
				Value any
			}, 0, p.cap),
			ReleaseFn: p.Release,
		}
	}
	return x.(*IteratorResult)
}

func (p *ResultPool) Release(r *IteratorResult) {
	if r != nil {
		r.Reset()
		p.pool.Put(r)
	}
}
