package util

import (
	"crypto/sha1"
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"go.uber.org/atomic"
)

var (
	ns   = uuid.MustParse("28840903-6eb5-4ffb-8880-93a4fa98dbcb") // Random UUID
	hash = sha1.New()
)

type IDGenerator interface {
	NewID() backend.UUID
}

var _ IDGenerator = (*DeterministicIDGenerator)(nil)

type DeterministicIDGenerator struct {
	seeds []int64
	seq   *atomic.Int64
}

func NewDeterministicIDGenerator(tenantID string, seeds ...int64) *DeterministicIDGenerator {
	seeds = append(seeds, int64(binary.LittleEndian.Uint64(stringToBytes(tenantID))))
	return &DeterministicIDGenerator{
		seeds: seeds,
		seq:   atomic.NewInt64(0),
	}
}

func (d *DeterministicIDGenerator) NewID() backend.UUID {
	seq := d.seq.Inc()
	seeds := append(d.seeds, seq)
	return backend.UUID(newDeterministicID(seeds))
}

func newDeterministicID(seeds []int64) uuid.UUID {
	b := int64ToBytes(seeds...)

	return uuid.NewHash(hash, ns, b, 5)
}

// TODO - Try to avoid allocs here
func stringToBytes(s string) []byte {
	return []byte(s)
}

func int64ToBytes(seeds ...int64) []byte {
	l := len(seeds)
	bytes := make([]byte, l*8)

	// Use binary.LittleEndian or binary.BigEndian depending on your requirement
	for i, seed := range seeds {
		binary.LittleEndian.PutUint64(bytes[i*8:], uint64(seed))
	}

	return bytes
}
