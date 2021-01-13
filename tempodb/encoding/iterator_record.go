package encoding

import (
	"bytes"
	"io"
)

type recordIterator struct {
	records []*Record
	ra      io.ReaderAt

	currentIterator Iterator
}

// NewRecordIterator returns a recordIterator.  This iterator is used for iterating through
//  a series of objects by reading them one at a time from Records.
func NewRecordIterator(r []*Record, ra io.ReaderAt) Iterator {
	return &recordIterator{
		records: r,
		ra:      ra,
	}
}

func (i *recordIterator) Next() (ID, []byte, error) {
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
