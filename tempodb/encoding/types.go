package encoding

import (
	"context"

	"github.com/grafana/tempo/tempodb/backend"
	"go.uber.org/atomic"
)

type ID []byte

type Record struct {
	ID     ID
	Start  uint64
	Length uint32
}

type Iterator interface {
	Next() (ID, []byte, error)
}

type Finder interface {
	Find(id ID) ([]byte, error)
}

type Appender interface {
	Append(ID, []byte) error
	Complete()
	Records() []*Record
	Length() int
}

type ObjectCombiner interface {
	Combine(objA []byte, objB []byte) []byte
}

// BackendBlock defines an object that can be used to interact with a block in object storage
type BackendBlock interface {
	// Find searches for a given ID and returns the object if exists
	Find(ctx context.Context, r backend.Reader, id ID, metrics *FindMetrics) ([]byte, error)
	// Iterator returns an iterator that can be used to examine every object in the block
	Iterator(chunkSizeBytes uint32, r backend.Reader) (Iterator, error)
}

// FindMetrics is a threadsafe struct for tracking metrics related to a parallelized query
type FindMetrics struct {
	BloomFilterReads     *atomic.Int32
	BloomFilterBytesRead *atomic.Int32
	IndexReads           *atomic.Int32
	IndexBytesRead       *atomic.Int32
	BlockReads           *atomic.Int32
	BlockBytesRead       *atomic.Int32
}

func NewFindMetrics() FindMetrics {
	return FindMetrics{
		BloomFilterReads:     atomic.NewInt32(0),
		BloomFilterBytesRead: atomic.NewInt32(0),
		IndexReads:           atomic.NewInt32(0),
		IndexBytesRead:       atomic.NewInt32(0),
		BlockReads:           atomic.NewInt32(0),
		BlockBytesRead:       atomic.NewInt32(0),
	}
}
