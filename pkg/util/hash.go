package util

import (
	"hash/fnv"
)

// TokenFor generates a token used for finding ingesters from ring.
// Not suitable for in-memory hashing or deduping because it is only 32-bit.
// The collision rate is about 1 in 8000.
func TokenFor(userID string, b []byte) uint32 {
	h := fnv.New32()
	_, _ = h.Write([]byte(userID))
	_, _ = h.Write(b)
	return h.Sum32()
}

// TokenForTraceID generates a hashed value for a trace id.  Used for bloom lookups.
// Do not change because it will break lookups on existing bloom filters.
func TokenForTraceID(b []byte) uint32 {
	h := fnv.New32()
	_, _ = h.Write(b)
	return h.Sum32()
}

// HashForTraceID generates a generic hash for the trace ID, suitable for mapping and deduping.
func HashForTraceID(tid []byte) uint64 {
	h := fnv.New64()
	_, _ = h.Write(tid)
	return h.Sum64()
}
