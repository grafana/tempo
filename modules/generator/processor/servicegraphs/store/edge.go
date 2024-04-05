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

	// Additional labels for uninstrumented services
	VirtualNode                           string // Indicates which service is the virtual node (client or server)
	ClientDbSystem, ClientMessagingSystem string // If the client is a virtual node, indicates the client's db or messaging system
	ServerDbSystem, ServerMessagingSystem string // If the server is a virtual node, indicates the server's db or messaging system

	// PeerNode is the attribute that will be used to create a peer edge
	PeerNode string

	// expiration is the time at which the Edge expires, expressed as Unix time
	expiration int64

	// Span multiplier is used for multiplying metrics
	SpanMultiplier float64
}

func newEdge(key string, ttl time.Duration) *Edge {
	return &Edge{
		key:        key,
		Dimensions: make(map[string]string),
		expiration: time.Now().Add(ttl).Unix(),
	}
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
