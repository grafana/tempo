package encoding

import (
	"bytes"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

// Appender is capable of tracking objects and ids that are added to it
type Appender interface {
	Append(common.ID, []byte) error
	Complete() error
	Records() []*common.Record
	Length() int
	DataLength() uint64
}

type appender struct {
	dataWriter    common.DataWriter
	records       []*common.Record
	currentOffset uint64
}

// NewAppender returns an appender.  This appender simply appends new objects
//  to the provided dataWriter.
func NewAppender(dataWriter common.DataWriter) Appender {
	return &appender{
		dataWriter: dataWriter,
	}
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *appender) Append(id common.ID, b []byte) error {
	_, err := a.dataWriter.Write(id, b)
	if err != nil {
		return err
	}

	bytesWritten, err := a.dataWriter.CutPage()
	if err != nil {
		return err
	}

	i := sort.Search(len(a.records), func(idx int) bool {
		return bytes.Compare(a.records[idx].ID, id) == 1
	})
	a.records = append(a.records, nil)
	copy(a.records[i+1:], a.records[i:])
	a.records[i] = &common.Record{
		ID:     id,
		Start:  a.currentOffset,
		Length: uint32(bytesWritten),
	}

	a.currentOffset += uint64(bytesWritten)

	return nil
}

func (a *appender) Records() []*common.Record {
	return a.records
}

func (a *appender) Length() int {
	return len(a.records)
}

func (a *appender) DataLength() uint64 {
	return a.currentOffset
}

func (a *appender) Complete() error {
	return a.dataWriter.Complete()
}
