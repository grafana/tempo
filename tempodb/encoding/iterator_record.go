package encoding

import (
	"bytes"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type recordIterator struct {
	records []*common.Record
	ra      io.ReaderAt

	currentIterator common.Iterator
}

// NewRecordIterator returns a recordIterator.  This iterator is used for iterating through
//  a series of objects by reading them one at a time from Records.
func NewRecordIterator(r []*common.Record, ra io.ReaderAt) common.Iterator {
	return &recordIterator{
		records: r,
		ra:      ra,
	}
}

func (i *recordIterator) Next() (common.ID, []byte, error) {
	if i.currentIterator != nil {
		id, object, err := i.currentIterator.Next()
		if err != nil {
			return nil, nil, err
		}
		if id != nil {
			return id, object, nil
		}
	}

	// read the next record and create an iterator
	if len(i.records) > 0 {
		record := i.records[0]

		buff := make([]byte, record.Length)
		_, err := i.ra.ReadAt(buff, int64(record.Start))
		if err != nil {
			return nil, nil, err
		}

		i.currentIterator = NewIterator(bytes.NewReader(buff))
		i.records = i.records[1:]

		return i.currentIterator.Next()
	}

	// done
	return nil, nil, nil
}

func (i *recordIterator) Close() {
}
