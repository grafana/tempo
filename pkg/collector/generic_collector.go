package collector

import "sync"

// FIXME: make this collector generic using the golang generics instead of the interface{}

type GenericCollector[T comparable] struct {
	values map[T]struct{}
	mu     sync.Mutex
}

// FIXME: do we need a GenericCollector for this? this is only used by SearchRecent??
// SearchRecent is the searching for only in ingesters??
func NewGenericCollector[T comparable]() *GenericCollector[T] {
	return &GenericCollector[T]{
		values: make(map[T]struct{}),
	}
}

func (c *GenericCollector) Collect(response interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.responses = append(c.responses, response)
}

func (c *GenericCollector) Results() interface{} {
	return c.responses
}

// FIXME: impliment this method
func (c *GenericCollector) Exceeded() bool {
	return false
}
