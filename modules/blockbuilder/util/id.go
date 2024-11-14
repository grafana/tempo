package util

import (
	"crypto/sha1"
	"encoding/binary"

	"github.com/google/uuid"
)

var (
	ns   = uuid.MustParse("28840903-6eb5-4ffb-8880-93a4fa98dbcb") // Random UUID
	hash = sha1.New()
)

func NewDeterministicID(ts, seq int64) uuid.UUID {
	b := int64ToBytes(ts, seq)

	return uuid.NewHash(hash, ns, b, 5)
}

func int64ToBytes(val1, val2 int64) []byte {
	// 16 bytes = 8 bytes (int64) + 8 bytes (int64)
	bytes := make([]byte, 16)

	// Use binary.LittleEndian or binary.BigEndian depending on your requirement
	binary.LittleEndian.PutUint64(bytes[0:8], uint64(val1))
	binary.LittleEndian.PutUint64(bytes[8:16], uint64(val2))

	return bytes
}
