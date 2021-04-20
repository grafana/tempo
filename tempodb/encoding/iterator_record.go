package encoding

import (
	"bytes"
	"context"
	"errors"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type recordIterator struct {
	records []*common.Record

	objectRW common.ObjectReaderWriter
	dataR    common.DataReader

	currentIterator Iterator

	buffer []byte
}

// NewRecordIterator returns a recordIterator.  This iterator is used for iterating through
//  a series of objects by reading them one at a time from Records.
func NewRecordIterator(r []*common.Record, dataR common.DataReader, objectRW common.ObjectReaderWriter) Iterator {
	return &recordIterator{
		records:  r,
		objectRW: objectRW,
		dataR:    dataR,
	}
}

func (i *recordIterator) Next(ctx context.Context) (common.ID, []byte, error) {
	if i.currentIterator != nil {
		id, object, err := i.currentIterator.Next(ctx)
		if err != nil && err != io.EOF {
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
		pages, i.buffer, err = i.dataR.Read(ctx, i.records[:1], i.buffer)
		if err != nil {
			return nil, nil, err
		}
		if len(pages) == 0 {
			return nil, nil, errors.New("unexpected 0 length pages from dataReader")
		}

		buff := pages[0]
		i.currentIterator = NewIterator(bytes.NewReader(buff), i.objectRW)
		i.records = i.records[1:]

		return i.currentIterator.Next(ctx)
	}

	// done
	return nil, nil, io.EOF
}

func (i *recordIterator) Close() {
	i.dataR.Close()
}
