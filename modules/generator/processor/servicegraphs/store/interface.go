package store

type Callback func(e *Edge)

// Store is an interface for building service graphs.
type Store interface {
	// UpsertEdge inserts or updates an edge.
	UpsertEdge(key string, update Callback) (isNew bool, err error)
	// Expire evicts expired edges from the store.
	Expire()
}
