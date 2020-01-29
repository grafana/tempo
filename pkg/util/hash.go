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

// todo:  better alg?  just add high order uint64 to low order uint64?
func Fingerprint(b []byte) uint64 {
	h := fnv.New64()
	_, _ = h.Write(b)
	return h.Sum64()
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
