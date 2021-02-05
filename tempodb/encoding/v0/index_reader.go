package v0

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type readerBytes struct {
	index []byte
}

// NewIndexReader returns an index reader for a byte slice of marshalled
// ordered records.
func NewIndexReader(index []byte) (common.IndexReader, error) {
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
