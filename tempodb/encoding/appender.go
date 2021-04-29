package encoding

import (
	"hash"

	"github.com/grafana/tempo/tempodb/encoding/common"

	"github.com/cespare/xxhash"
	"github.com/huandu/skiplist"
)

// Appender is capable of tracking objects and ids that are added to it
type Appender interface {
	Append(common.ID, []byte) error
	Complete() error
	Records() []common.Record
	Length() int
	DataLength() uint64
}

type appender struct {
	dataWriter    common.DataWriter
	records       *skiplist.SkipList
	hash          hash.Hash64
	currentOffset uint64
}

// NewAppender returns an appender.  This appender simply appends new objects
//  to the provided dataWriter.
func NewAppender(dataWriter common.DataWriter) Appender {
	return &appender{
		dataWriter: dataWriter,
		records:    skiplist.New(skiplist.Bytes),
		hash:       xxhash.New(),
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

	var sliceRecords []common.Record
	element := a.records.Get(id)
	if element != nil {
		sliceRecords = element.Value.([]common.Record)
	}
	sliceRecords = append(sliceRecords, common.Record{
		ID:     id,
		Start:  a.currentOffset,
		Length: uint32(bytesWritten),
	})
	a.records.Set(id, sliceRecords)

	a.currentOffset += uint64(bytesWritten)

	return nil
}

func (a *appender) Records() []common.Record {
	sliceRecords := make([]common.Record, 0, a.records.Len())

	elem := a.records.Front()
	for elem != nil {
		r := elem.Value.([]common.Record)
		sliceRecords = append(sliceRecords, r...)
		elem = elem.Next()
	}

	return sliceRecords
}

func (a *appender) Length() int {
	return a.records.Len()
}

func (a *appender) DataLength() uint64 {
	return a.currentOffset
}

func (a *appender) Complete() error {
	return a.dataWriter.Complete()
}
