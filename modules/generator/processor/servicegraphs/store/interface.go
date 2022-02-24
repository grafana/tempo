package store

type Callback func(e *Edge)

// Store is an interface for building service graphs.
type Store interface {
	// UpsertEdge inserts or updates an edge.
	UpsertEdge(key string, cb Callback) (*Edge, error)
	// EvictEdge removes an edge from the store.
	EvictEdge(key string)
	// Expire evicts all expired edges from the store.
	Expire()
}
