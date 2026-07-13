package store

import "time"

type ConnectionType string

const (
	Unknown         ConnectionType = ""
	MessagingSystem ConnectionType = "messaging_system"
	Database        ConnectionType = "database"
	VirtualNode     ConnectionType = "virtual_node"
)

// Edge is an Edge between two nodes in the graph.
//
// Edges are pool-allocated and pointers handed to update/onComplete/onExpire
// callbacks are only valid for the duration of the callback — the store may
// recycle the Edge after the callback returns. Do not retain *Edge or any
// substrings of its fields past the callback.
type Edge struct {
	key edgeKey
	// Intrusive list pointers; mutated only by *Store under Store.mtx.
	prev, next *Edge

	// traceID is an edge-owned copy of the raw span trace ID bytes (not hex).
	// The backing buffer is reused across pool recycles; it must only be
	// written through SetTraceID — assigning a caller's slice would make the
	// pool retain and later overwrite foreign memory.
	traceID                                        []byte
	ConnectionType                                 ConnectionType
	ServerService, ClientService                   string
	ServerLatencySec, ClientLatencySec             float64
	ServerStartTimeUnixNano, ClientEndTimeUnixNano uint64

	// If either the client or the server spans have status code error,
	// the Edge will be considered as failed.
	Failed bool

	// Additional dimension to add to the metrics
	Dimensions map[string]string

	// PeerNode is the attribute that will be used to create a peer edge
	PeerNode string

	// expiration is the time at which the Edge expires, expressed as Unix time
	expiration int64

	// Span multiplier is used for multiplying metrics
	SpanMultiplier float64
}

// resetEdge clears the Edge for reuse, keeping its reusable Dimensions and
// trace ID buffers and defaulting SpanMultiplier.
func resetEdge(e *Edge) {
	*e = Edge{
		Dimensions:     e.Dimensions,
		traceID:        e.traceID[:0],
		SpanMultiplier: 1,
	}
	clear(e.Dimensions)
}

// SetTraceID copies id into the Edge's reused trace ID buffer. The Edge must
// own the bytes: id typically aliases the decoded push request, which must
// not be retained past PushSpans, while the Edge lives until it completes or
// expires.
func (e *Edge) SetTraceID(id []byte) {
	e.traceID = append(e.traceID[:0], id...)
}

// TraceID returns the raw trace ID bytes. The slice aliases an edge-owned
// buffer that is reused after the callback returns; do not retain it.
func (e *Edge) TraceID() []byte {
	return e.traceID
}

// isComplete returns true if the corresponding client and server
// pair spans have been processed for the given Edge
func (e *Edge) isComplete() bool {
	return len(e.ClientService) != 0 && len(e.ServerService) != 0
}

func (e *Edge) isExpired() bool {
	return time.Now().Unix() >= e.expiration
}

// IsRoot reports whether the edge key has an empty parent span ID.
func (e *Edge) IsRoot() bool {
	return e.key.root
}
