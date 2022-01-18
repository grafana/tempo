package store

import "time"

// Edge is an Edge between two nodes in the graph
type Edge struct {
	key string

	ServerService, ClientService string
	ServerLatency, ClientLatency float64

	// If either the client or the server spans have status code error,
	// the Edge will be considered as failed.
	Failed bool

	// expiration is the time at which the Edge expires, expressed as Unix time
	expiration int64
}

func NewEdge(key string, ttl time.Duration) *Edge {
	return &Edge{
		key: key,

		expiration: time.Now().Add(ttl).Unix(),
	}
}

// IsCompleted returns true if the corresponding client and server
// pair spans have been processed for the given Edge
func (e *Edge) IsCompleted() bool {
	return len(e.ClientService) != 0 && len(e.ServerService) != 0
}

func (e *Edge) IsExpired() bool {
	return time.Now().Unix() >= e.expiration
}
