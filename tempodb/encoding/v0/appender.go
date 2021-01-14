package v0

import (
	"bytes"
	"io"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding"
)

type appender struct {
	writer        io.Writer
	records       []*encoding.Record
	currentOffset int
}

// NewAppender returns an appender.  This appender simply appends new objects
//  to the provided io.Writer.
func NewAppender(writer io.Writer) encoding.Appender {
	return &appender{
		writer: writer,
	}
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *appender) Append(id encoding.ID, b []byte) error {
	length, err := marshalObjectToWriter(id, b, a.writer)
	if err != nil {
		return err
	}

	i := sort.Search(len(a.records), func(idx int) bool {
		return bytes.Compare(a.records[idx].ID, id) == 1
	})
	a.records = append(a.records, nil)
	copy(a.records[i+1:], a.records[i:])
	a.records[i] = &encoding.Record{
		ID:     id,
		Start:  uint64(a.currentOffset),
		Length: uint32(length),
	}

	a.currentOffset += length
	return nil
}

func (a *appender) Records() []*encoding.Record {
	return a.records
}

func (a *appender) Length() int {
	return len(a.records)
}

func (a *appender) Complete() {

}
