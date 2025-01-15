package util

import (
	"crypto/sha1"
	"encoding/binary"
	"hash"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"go.uber.org/atomic"
)

const (
	sha1Version5 = 5
)

var ns = uuid.MustParse("28840903-6eb5-4ffb-8880-93a4fa98dbcb") // Random UUID

type IDGenerator interface {
	NewID() backend.UUID
}

var _ IDGenerator = (*DeterministicIDGenerator)(nil)

type DeterministicIDGenerator struct {
	buf  []byte
	seq  *atomic.Uint64
	hash hash.Hash
}

func NewDeterministicIDGenerator(tenantID string, seeds ...uint64) *DeterministicIDGenerator {
	return &DeterministicIDGenerator{
		buf:  newBuf([]byte(tenantID), seeds),
		seq:  atomic.NewUint64(0),
		hash: sha1.New(),
	}
}

func newBuf(tenantID []byte, seeds []uint64) []byte {
	dl, sl := len(tenantID), len(seeds)
	data := make([]byte, dl+sl*8+8) // tenantID bytes + 8 bytes per uint64 + 8 bytes for seq
	copy(data, tenantID)

	for i, seed := range seeds {
		binary.LittleEndian.PutUint64(data[dl+i*8:], seed)
	}

	return data
}

func (d *DeterministicIDGenerator) NewID() backend.UUID {
	return backend.UUID(newDeterministicID(d.hash, d.buf, d.seq.Inc()))
}

func newDeterministicID(hash hash.Hash, data []byte, seq uint64) uuid.UUID {
	// update last 8 bytes of data with seq
	binary.LittleEndian.PutUint64(data[len(data)-8:], seq)

	return uuid.NewHash(hash, ns, data, sha1Version5)
}
