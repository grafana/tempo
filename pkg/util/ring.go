package util

import "hash/fnv"

// TokenFor generates a token used for finding ingesters from ring
func TokenFor(b []byte) uint32 {
	h := fnv.New32()
	_, _ = h.Write(b)
	return h.Sum32()
}
