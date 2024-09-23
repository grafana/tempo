package collector

import "sync"

// GenericCollector is a safe collector to collect values
// and retrieve them later when needed.
type GenericCollector[T comparable] struct {
	values []T
	mtx    sync.Mutex
}

// NewGenericCollector initializes a new GenericCollector for type T.
func NewGenericCollector[T comparable]() *GenericCollector[T] {
	return &GenericCollector[T]{
		values: []T{},
	}
}

// Collect adds the values to the collector.
func (c *GenericCollector[T]) Collect(value T) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.values = append(c.values, value)
}

// Values returns the collected values.
func (c *GenericCollector[T]) Values() []T {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.values
}
