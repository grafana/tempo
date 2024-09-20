package collector

import "sync"

// GenericCollector is a generic collector using Go generics.
type GenericCollector[T comparable] struct {
	values []T        // A slice of generic type T to store the collected values
	mu     sync.Mutex // A mutex for thread-safe access to values
}

// NewGenericCollector initializes a new GenericCollector for type T.
func NewGenericCollector[T comparable]() *GenericCollector[T] {
	return &GenericCollector[T]{
		values: []T{},
	}
}

// Collect adds the values to the collector.
// Collect now works with generic types.
func (c *GenericCollector[T]) Collect(value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.values = append(c.values, value)
}

// Values returns the collected values.
func (c *GenericCollector[T]) Values() []T {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.values
}
