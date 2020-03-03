package util

import (
	"encoding/binary"
	"encoding/hex"
	"hash/fnv"
	"math"

	"github.com/grafana/frigg/friggdb/backend"
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

func Float64fromID(ID backend.ID) float64 {
	bytes := []byte(ID)

	// binary representation
	bits := binary.LittleEndian.Uint64(bytes)

	// float64 from binary rep. Pretty cool
	float := math.Float64frombits(bits)
	return float
}
