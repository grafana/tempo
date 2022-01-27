package store

type Callback func(e *Edge)

// Store is an interface for building service graphs.
type Store interface {
	// UpsertEdge inserts or updates an edge.
	UpsertEdge(string, Callback) (*Edge, error)
	// EvictEdgeWithLock removes an edge from the store.
	EvictEdgeWithLock(string)
	// Expire evicts all expired edges from the store.
	Expire()
}
