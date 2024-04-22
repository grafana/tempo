package store

import "time"

type ConnectionType string

const (
	Unknown         ConnectionType = ""
	MessagingSystem ConnectionType = "messaging_system"
	Database        ConnectionType = "database"
	VirtualNode     ConnectionType = "virtual_node"
)

// Edge is an Edge between two nodes in the graph
type Edge struct {
	key string

	TraceID                            string
	ConnectionType                     ConnectionType
	ServerService, ClientService       string
	ServerLatencySec, ClientLatencySec float64

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

// zeroStateEdge resets the Edge to its zero state.
// Useful for reusing an Edge without allocating a new one.
func zeroStateEdge(e *Edge) {
	e.TraceID = ""
	e.ConnectionType = Unknown
	e.ServerService = ""
	e.ClientService = ""
	e.ServerLatencySec = 0
	e.ClientLatencySec = 0
	e.Failed = false
	for k := range e.Dimensions {
		// saves 30ns/op, 50 B/op, 1 allocs/op
		delete(e.Dimensions, k)
	}
	e.PeerNode = ""
	e.SpanMultiplier = 1
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
