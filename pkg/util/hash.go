package util

import (
	"encoding/binary"
	"encoding/hex"
	"hash/fnv"

	"github.com/grafana/tempo/tempodb/encoding"
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

func BlockIDRange(maxID encoding.ID, minID encoding.ID) float64 {
	if len(maxID) > 8 {
		maxIDHighBytes := []byte(maxID)[8:]
		minIDHighBytes := []byte(minID)[8:]

		maxIDHigh := float64(binary.LittleEndian.Uint64(maxIDHighBytes))
		minIDHigh := float64(binary.LittleEndian.Uint64(minIDHighBytes))

		if maxIDHigh-minIDHigh > 0 {
			return (2 ^ 64) - 1
		}
	}
	maxIDLowBytes := []byte(maxID)[0 : len(maxID)-1]
	minIDLowBytes := []byte(minID)[0 : len(minID)-1]

	maxIDLow := float64(binary.LittleEndian.Uint64(maxIDLowBytes))
	minIDLow := float64(binary.LittleEndian.Uint64(minIDLowBytes))

	return maxIDLow - minIDLow
}
