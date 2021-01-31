package v0

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

// jpe - test, comment
type readerBytes struct {
	index []byte
}

// NewIndexReaderBytes returns an index reader for a byte slice of marshalled
// ordered records.
func NewIndexReaderBytes(index []byte) (common.IndexReader, error) {
	mod := len(index) % recordLength
	if mod != 0 {
		return nil, fmt.Errorf("records are an unexpected number of bytes %d", len(index))
	}

	return &readerBytes{
		index: index,
	}, nil
}

func (r *readerBytes) At(i int) *common.Record {
	if i < 0 || i >= len(r.index)/recordLength {
		return nil
	}

	buff := r.index[i*recordLength : (i+1)*recordLength]
	return unmarshalRecord(buff)
}

func (r *readerBytes) Find(id common.ID) (*common.Record, int) {
	numRecords := recordCount(r.index)
	var record *common.Record

	i := sort.Search(numRecords, func(i int) bool {
		buff := r.index[i*recordLength : (i+1)*recordLength]
		record = unmarshalRecord(buff)

		return bytes.Compare(record.ID, id) >= 0
	})

	if i >= 0 && i < numRecords {
		buff := r.index[i*recordLength : (i+1)*recordLength]
		record = unmarshalRecord(buff)

		return record, i
	}

	return nil, -1
}

type readerRecords struct {
	records []*common.Record
}

// NewIndexReaderRecords returns an index reader for an ordered slice
// of records
func NewIndexReaderRecords(records []*common.Record) common.IndexReader {
	return &readerRecords{
		records: records,
	}
}

func (r *readerRecords) At(i int) *common.Record {
	if i < 0 || i >= len(r.records) {
		return nil
	}

	return r.records[i]
}

func (r *readerRecords) Find(id common.ID) (*common.Record, int) {
	i := sort.Search(len(r.records), func(idx int) bool {
		return bytes.Compare(r.records[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(r.records) {
		return nil, -1
	}

	return r.records[i], i
}
