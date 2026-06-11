package store

type Side string

const (
	Client Side = "client"
	Server Side = "server"
)

// Callback is invoked by the store with a pooled *Edge while holding the
// store mutex. The Edge pointer is only valid for the duration of the call —
// the store may return it to the pool after the callback returns. Do not
// retain the pointer, send it to another goroutine, or store substrings of
// its fields beyond the callback's lifetime.
type Callback func(e *Edge)

// Store is an interface for building service graphs.
type Store interface {
	// UpsertEdge inserts or updates an edge.
	UpsertEdge(key string, side Side, update Callback) (isNew bool, err error)
	// UpsertEdgeFromBytes inserts or updates an edge from trace/span ID bytes.
	UpsertEdgeFromBytes(traceID, spanID []byte, side Side, update Callback) (isNew bool, err error)
	// AddDroppedSpanSide adds a dropped span side key and returns true if an
	// existing buffered counterpart edge was removed.
	AddDroppedSpanSide(key string, side Side) bool
	// HasDroppedSpanSide checks if a dropped span side key exists.
	HasDroppedSpanSide(key string, side Side) bool
	// Expire evicts expired edges from the store.
	Expire()
}
