package encoding

import (
	"hash"

	"github.com/cespare/xxhash"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// Appender is capable of tracking objects and ids that are added to it
type Appender interface {
	Append(common.ID, []byte) error
	Complete() error
	Records() []common.Record
	RecordsForID(common.ID) []common.Record
	Length() int
	DataLength() uint64
}

type appender struct {
	dataWriter    common.DataWriter
	records       map[uint64][]common.Record
	hash          hash.Hash64
	currentOffset uint64
}

// NewAppender returns an appender.  This appender simply appends new objects
//  to the provided dataWriter.
func NewAppender(dataWriter common.DataWriter) Appender {
	return &appender{
		dataWriter: dataWriter,
		records:    map[uint64][]common.Record{},
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

	a.hash.Reset()
	_, _ = a.hash.Write(id)
	hash := a.hash.Sum64()

	records := a.records[hash]
	records = append(records, common.Record{
		ID:     id,
		Start:  a.currentOffset,
		Length: uint32(bytesWritten),
	})
	a.records[hash] = records
	a.currentOffset += uint64(bytesWritten)

	return nil
}

func (a *appender) Records() []common.Record {
	sliceRecords := make([]common.Record, 0, len(a.records))
	for _, r := range a.records {
		sliceRecords = append(sliceRecords, r...)
	}

	common.SortRecords(sliceRecords)
	return sliceRecords
}

func (a *appender) RecordsForID(id common.ID) []common.Record {
	a.hash.Reset()
	_, _ = a.hash.Write(id)
	hash := a.hash.Sum64()

	return a.records[hash]
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
