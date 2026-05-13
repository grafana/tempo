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
	// key is unsafe.String aliased over keyBuf and is only valid while the
	// Edge is in the store's map. keyBuf is preserved across pool recycles
	// and overwritten on the next grabEdgeFromBytes; substrings of key
	// must not outlive the Edge in the store.
	key    string
	keyBuf []byte
	// Intrusive list pointers; mutated only by *store under store.mtx.
	prev, next *Edge

	// TraceID is used when the trace ID is already a string or is too large to
	// keep in TraceIDBytes. Normal OTLP trace IDs use TraceIDBytes to avoid
	// hex-encoding before the exemplar is actually collected.
	TraceID                                        string
	TraceIDLen                                     int
	TraceIDRaw                                     [16]byte
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

// resetEdge resets the Edge to its zero state.
// Useful for reusing an Edge without allocating a new one.
func resetEdge(e *Edge) {
	*e = Edge{
		Dimensions:     e.Dimensions,
		keyBuf:         e.keyBuf,
		SpanMultiplier: 1,
	}
	clear(e.Dimensions)
}

// isComplete returns true if the corresponding client and server
// pair spans have been processed for the given Edge
func (e *Edge) isComplete() bool {
	return len(e.ClientService) != 0 && len(e.ServerService) != 0
}

func (e *Edge) isExpired() bool {
	return time.Now().Unix() >= e.expiration
}

func (e *Edge) Key() string {
	return e.key
}
