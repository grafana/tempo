package v2

import (
	"bytes"
	"context"
	"io"
	"sort"

	"github.com/grafana/tempo/v2/tempodb/encoding/common"
)

// Record represents the location of an ID in an object file
type Record struct {
	ID     common.ID
	Start  uint64
	Length uint32
}

type recordSorter struct {
	records []Record
}

// SortRecords sorts a slice of record pointers
func SortRecords(records []Record) {
	sort.Sort(&recordSorter{
		records: records,
	})
}

func (t *recordSorter) Len() int {
	return len(t.records)
}

func (t *recordSorter) Less(i, j int) bool {
	a := t.records[i]
	b := t.records[j]

	return bytes.Compare(a.ID, b.ID) == -1
}

func (t *recordSorter) Swap(i, j int) {
	t.records[i], t.records[j] = t.records[j], t.records[i]
}

// Records is a slice of *Record
type Records []Record

// At implements IndexReader
func (r Records) At(_ context.Context, i int) (*Record, error) {
	if i < 0 || i >= len(r) {
		return nil, nil
	}

	return &r[i], nil
}

// Find implements IndexReader
func (r Records) Find(_ context.Context, id common.ID) (*Record, int, error) {
	i := sort.Search(len(r), func(idx int) bool {
		return bytes.Compare(r[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(r) {
		return nil, -1, nil
	}

	return &r[i], i, nil
}

// DataReader returns a slice of pages in the encoding/v0 format referenced by
// the slice of *Records passed in.  The length of the returned slice is guaranteed
// to be equal to the length of the provided records unless error is non nil.
// DataReader is the primary abstraction point for supporting multiple data
// formats.
type DataReader interface {
	Read(context.Context, []Record, [][]byte, []byte) ([][]byte, []byte, error)
	Close()

	// NextPage can be used to iterate at a page at a time. May return ErrUnsupported for older formats
	//  NextPage takes a reusable buffer to read the page into and returns it in case it needs to resize
	//  NextPage returns the uncompressed page buffer ready for object iteration and the length of the
	//    original page from the page header. len(page) might not equal page len!
	NextPage([]byte) ([]byte, uint32, error)
}

// IndexReader is used to abstract away the details of an index.  Currently
// only used in the paged finder, it could eventually provide a way to
// support multiple index formats.
// IndexReader is the primary abstraction point for supporting multiple index
// formats.
type IndexReader interface {
	At(ctx context.Context, i int) (*Record, error)
	Find(ctx context.Context, id common.ID) (*Record, int, error)
}

// DataWriter is used to write paged data to the backend
type DataWriter interface {
	// Write writes the passed ID/byte to the current page
	Write(common.ID, []byte) (int, error)
	// CutPage completes the current page and start a new one.  It
	//  returns the length in bytes of the cut page.
	CutPage() (int, error)
	// Complete must be called when the operation DataWriter is done.
	Complete() error
}

// DataWriterGeneric writes objects instead of byte slices
type DataWriterGeneric interface {
	// Write writes the passed ID/obj to the current page
	Write(context.Context, common.ID, interface{}) (int, error)

	// CutPage completes the current page and start a new one.  It
	//  returns the length in bytes of the cut page.
	CutPage(context.Context) (int, error)

	// Complete must be called when the operation DataWriter is done.
	Complete(context.Context) error
}

// IndexWriter is used to write paged indexes
type IndexWriter interface {
	// Write returns a byte representation of the provided Records
	Write([]Record) ([]byte, error)
}

// ObjectReaderWriter represents a library of methods to read and write
// at the object level
type ObjectReaderWriter interface {
	MarshalObjectToWriter(id common.ID, b []byte, w io.Writer) (int, error)
	UnmarshalObjectFromReader(r io.Reader) (common.ID, []byte, error)
	UnmarshalAndAdvanceBuffer(buffer []byte) ([]byte, common.ID, []byte, error)
}

// RecordReaderWriter represents a library of methods to read and write
// records
type RecordReaderWriter interface {
	MarshalRecords(records []Record) ([]byte, error)
	MarshalRecordsToBuffer(records []Record, buffer []byte) error
	RecordCount(b []byte) int
	UnmarshalRecord(buff []byte) Record
	RecordLength() int
}
