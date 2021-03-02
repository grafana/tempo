package common

import "context"

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
	Next(context.Context) (ID, []byte, error)
	Close()
}

// Finder is capable of finding the requested ID
type Finder interface {
	Find(context.Context, ID) ([]byte, error)
}

// Appender is capable of tracking objects and ids that are added to it
type Appender interface {
	Append(ID, []byte) error
	Complete() error
	Records() []*Record
	Length() int
	DataLength() uint64
}

// ObjectCombiner is used to combine two objects in the backend
type ObjectCombiner interface {
	Combine(objA []byte, objB []byte) []byte
}

// PageReader returns a slice of pages in the encoding/v0 format referenced by
// the slice of *Records passed in.  The length of the returned slice is guaranteed
// to be equal to the length of the provided records unless error is non nil.
// PageReader is the primary abstraction point for supporting multiple data
// formats.
type PageReader interface {
	Read(context.Context, []*Record) ([][]byte, error)
	Close()
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

// PageWriter is used to write paged data to the backend
type PageWriter interface {
	// Write writes the passed ID/byte to the current page
	Write(ID, []byte) (int, error)
	// CutPage completes the current page and start a new one.  It
	//  returns the length in bytes of the cut page.
	CutPage() (int, error)
	// Complete must be called when the operation pagewriter is done.
	Complete() error
}

// IndexWriter is used to write paged indexes
type IndexWriter interface {
	// Write returns a byte representation of the provided Records
	Write([]*Record) ([]byte, error)
}
