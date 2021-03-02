package v0

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type readerBytes struct {
	index []byte
}

// NewIndexReader returns an index reader for a byte slice of marshalled
// ordered records.
func NewIndexReader(r backend.ContextReader) (common.IndexReader, error) {
	index, err := r.ReadAll(context.Background())
	if err != nil {
		return nil, err
	}

	mod := len(index) % recordLength
	if mod != 0 {
		return nil, fmt.Errorf("records are an unexpected number of bytes %d", len(index))
	}

	return &readerBytes{
		index: index,
	}, nil
}

func (r *readerBytes) At(_ context.Context, i int) (*common.Record, error) {
	if i < 0 || i >= len(r.index)/recordLength {
		return nil, nil
	}

	buff := r.index[i*recordLength : (i+1)*recordLength]
	return unmarshalRecord(buff), nil
}

func (r *readerBytes) Find(_ context.Context, id common.ID) (*common.Record, int, error) {
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

		return record, i, nil
	}

	return nil, -1, nil
}
