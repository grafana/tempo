package v2

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type recordIterator struct {
	records []Record

	objectRW ObjectReaderWriter
	dataR    DataReader

	currentIterator BytesIterator

	buffer []byte
}

// newRecordIterator returns a recordIterator.  This iterator is used for iterating through
// a series of objects by reading them one at a time from Records.
func newRecordIterator(r []Record, dataR DataReader, objectRW ObjectReaderWriter) BytesIterator {
	return &recordIterator{
		records:  r,
		objectRW: objectRW,
		dataR:    dataR,
	}
}

func (i *recordIterator) NextBytes(ctx context.Context) (common.ID, []byte, error) {
	if i.currentIterator != nil {
		id, object, err := i.currentIterator.NextBytes(ctx)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, nil, err
		}
		if id != nil {
			return id, object, nil
		}
	}

	// read the next record and create an iterator
	if len(i.records) > 0 {
		var pages [][]byte
		var err error
		pages, i.buffer, err = i.dataR.Read(ctx, i.records[:1], pages, i.buffer)
		if err != nil {
			return nil, nil, err
		}
		if len(pages) == 0 {
			return nil, nil, errors.New("unexpected 0 length pages from dataReader")
		}

		buff := pages[0]
		i.currentIterator = NewIterator(bytes.NewReader(buff), i.objectRW)
		i.records = i.records[1:]

		return i.currentIterator.NextBytes(ctx)
	}

	// done
	return nil, nil, io.EOF
}

func (i *recordIterator) Close() {
	i.dataR.Close()
}
