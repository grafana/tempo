package store

type Side string

const (
	Client Side = "client"
	Server Side = "server"
)

type Callback func(e *Edge)

// Store is an interface for building service graphs.
type Store interface {
	// UpsertEdge inserts or updates an edge.
	UpsertEdge(key string, side Side, update Callback) (isNew bool, err error)
	// AddDroppedSpanSide adds a dropped span side key.
	AddDroppedSpanSide(key string, side Side)
	// HasDroppedSpanSide checks if a dropped span side key exists.
	HasDroppedSpanSide(key string, side Side) bool
	// Expire evicts expired edges from the store.
	Expire()
}
