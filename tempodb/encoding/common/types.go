package common

import (
	"context"
	"io"
)

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

// ObjectCombiner is used to combine two objects in the backend
type ObjectCombiner interface {
	Combine(objA []byte, objB []byte) []byte
}

// DataReader returns a slice of pages in the encoding/v0 format referenced by
// the slice of *Records passed in.  The length of the returned slice is guaranteed
// to be equal to the length of the provided records unless error is non nil.
// DataReader is the primary abstraction point for supporting multiple data
// formats.
type DataReader interface {
	Read(context.Context, []*Record) ([][]byte, error)
	Close()
}

// IndexReader is used to abstract away the details of an index.  Currently
// only used in the paged finder, it could eventually provide a way to
// support multiple index formats.
// IndexReader is the primary abstraction point for supporting multiple index
// formats.
type IndexReader interface {
	At(ctx context.Context, i int) (*Record, error)
	Find(ctx context.Context, id ID) (*Record, int, error)
}

// DataWriter is used to write paged data to the backend
type DataWriter interface {
	// Write writes the passed ID/byte to the current page
	Write(ID, []byte) (int, error)
	// CutPage completes the current page and start a new one.  It
	//  returns the length in bytes of the cut page.
	CutPage() (int, error)
	// Complete must be called when the operation DataWriter is done.
	Complete() error
}

// IndexWriter is used to write paged indexes
type IndexWriter interface {
	// Write returns a byte representation of the provided Records
	Write([]*Record) ([]byte, error)
}

// ObjectReaderWriter represents a library of methods to read and write
// at the object level
type ObjectReaderWriter interface {
	MarshalObjectToWriter(id ID, b []byte, w io.Writer) (int, error)
	UnmarshalObjectFromReader(r io.Reader) (ID, []byte, error)
	UnmarshalAndAdvanceBuffer(buffer []byte) ([]byte, ID, []byte, error)
}

// RecordReaderWriter represents a library of methods to read and write
// records
type RecordReaderWriter interface {
	MarshalRecords(records []*Record) ([]byte, error)
	MarshalRecordsToBuffer(records []*Record, buffer []byte) error
	RecordCount(b []byte) int
	UnmarshalRecord(buff []byte) *Record
}
