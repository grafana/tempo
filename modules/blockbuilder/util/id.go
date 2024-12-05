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
	tenantBytes []byte
	seeds       []int64
	seq         *atomic.Int64
}

func NewDeterministicIDGenerator(tenantID string, seeds ...int64) *DeterministicIDGenerator {
	return &DeterministicIDGenerator{
		tenantBytes: []byte(tenantID),
		seeds:       seeds,
		seq:         atomic.NewInt64(0),
	}
}

func (d *DeterministicIDGenerator) NewID() backend.UUID {
	seq := d.seq.Inc()
	return backend.UUID(newDeterministicID(d.tenantBytes, append(d.seeds, seq)))
}

func newDeterministicID(b []byte, seeds []int64) uuid.UUID {
	sl, dl := len(seeds), len(b)
	data := make([]byte, dl+sl*8) // 8 bytes per int64
	copy(b, data)

	for i, seed := range seeds {
		binary.LittleEndian.PutUint64(data[dl+i*8:], uint64(seed))
	}

	return uuid.NewHash(hash, ns, data, 5)
}
