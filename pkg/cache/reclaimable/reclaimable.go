package reclaimable

import "sync"

// Cache is an in-memory key/value cache that is best-effort. It does not
// reserve dedicated space for values and can be reclaimed by the garbage collector as needed.
type Cache[TKey comparable, TValue any] struct {
	max  int
	f    func(TKey) TValue
	pool *sync.Pool
}

// New reclaimable cache.  Must provide the function to populate values. The maximum number of values
// can be optionally set (zero means unlimited).
func New[TKey comparable, TValue any](f func(TKey) TValue, maxLen int) Cache[TKey, TValue] {
	return Cache[TKey, TValue]{
		max: maxLen,
		f:   f,
		pool: &sync.Pool{
			New: func() any {
				return make(map[TKey]TValue)
			},
		},
	}
}

func (c Cache[TKey, TValue]) Get(key TKey) TValue {
	m := c.pool.Get().(map[TKey]TValue)
	defer c.pool.Put(m)

	if v, ok := m[key]; ok {
		return v
	}

	v := c.f(key)

	if c.max > 0 && len(m) < c.max {
		m[key] = v
	}
	return v
}
