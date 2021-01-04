package util

import (
	"encoding/hex"
	"hash/fnv"
)

// TokenFor generates a token used for finding ingesters from ring
func TokenFor(userID string, b []byte) uint32 {
	h := fnv.New32()
	_, _ = h.Write([]byte(userID))
	_, _ = h.Write(b)
	return h.Sum32()
}

// TokenForTraceID generates a hashed value for a trace id
func TokenForTraceID(b []byte) uint32 {
	h := fnv.New32()
	_, _ = h.Write(b)
	return h.Sum32()
}

func HexStringToTraceID(id string) ([]byte, error) {
	byteID, err := hex.DecodeString(id)
	if err != nil {
		return nil, err
	}

	size := len(byteID)
	if size < 16 {
		byteID = append(make([]byte, 16-size), byteID...)
	}

	return byteID, nil
}
