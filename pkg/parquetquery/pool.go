package parquetquery

import (
	"sync"

	"github.com/parquet-go/parquet-go"
)

var DefaultPool = NewResultPool(10)

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
	if x := p.pool.Get(); x != nil {
		return x.(*IteratorResult)
	}

	return &IteratorResult{
		Entries: make([]struct {
			Key    string
			Values []parquet.Value
		}, 0, p.cap),
		OtherEntries: make([]struct {
			Key   string
			Value any
		}, 0, p.cap),
	}
}

func (p *ResultPool) Release(r *IteratorResult) {
	r.Reset()
	p.pool.Put(r)
}
