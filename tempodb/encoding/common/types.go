package common

import "go.uber.org/atomic"

// This file contains types that need to be referenced by both the ./encoding and ./encoding/vX packages.
// It primarily exists here to break dependency loops.

// ID in TempoDB
type ID []byte

// Record represents the location of an ID in an object file
type Record struct {
	ID     ID
	Start  uint64
	Length uint32
}

// Iterator is capable of iterating through a set of objects
type Iterator interface {
	Next() (ID, []byte, error)
}

// Finder is capable of finding the requested ID
type Finder interface {
	Find(id ID) ([]byte, error)
}

// Appender is capable of tracking objects and ids that are added to it
type Appender interface {
	Append(ID, []byte) error
	Complete()
	Records() []*Record
	Length() int
	DataLength() uint64
}

// ObjectCombiner is used to combine two objects in the backend
type ObjectCombiner interface {
	Combine(objA []byte, objB []byte) []byte
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

// NewFindMetrics constructs a FindMetrics
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

// PageReader returns a slice of pages in the encoding/v0 format referenced by
// the slice of *Records passed in.  The length of the returned slice is guaranteed
// to be equal to the length of the provided records unless error is non nil.
// PageReader is the primary abstraction point for supporting multiple data
// formats.
type PageReader interface {
	Read([]*Record) ([][]byte, error)
}

// IndexReader is used to abstract away the details of an index.  Currently
// only used in the paged finder, it could eventually provide a way to
// support multiple index formats.
// IndexReader is the primary abstraction point for supporting multiple index
// formats.
type IndexReader interface {
	At(i int) *Record
	Find(id ID) (*Record, int)
}
