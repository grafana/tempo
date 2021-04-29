package encoding

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type recordIterator struct {
	indexReader common.IndexReader
	index       int

	objectRW common.ObjectReaderWriter
	dataR    common.DataReader

	currentIterator Iterator

	buffer []byte
}

// NewRecordIterator returns a recordIterator.  This iterator is used for iterating through
//  a series of objects by reading them one at a time from Records.
func NewRecordIterator(indexReader common.IndexReader, dataR common.DataReader, objectRW common.ObjectReaderWriter) Iterator {
	return &recordIterator{
		indexReader: indexReader,
		index:       0,
		objectRW:    objectRW,
		dataR:       dataR,
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
	if i.index < i.indexReader.Len() {
		var pages [][]byte
		var err error
		record, err := i.indexReader.At(ctx, i.index)
		if err != nil {
			return nil, nil, err
		}
		if record == nil {
			return nil, nil, fmt.Errorf("unexpected nil record at %d, %d", i, i.indexReader.Len())
		}

		pages, i.buffer, err = i.dataR.Read(ctx, []common.Record{*record}, i.buffer)
		if err != nil {
			return nil, nil, err
		}
		if len(pages) == 0 {
			return nil, nil, errors.New("unexpected 0 length pages from dataReader")
		}

		buff := pages[0]
		i.currentIterator = NewIterator(bytes.NewReader(buff), i.objectRW)
		i.index++

		return i.currentIterator.Next(ctx)
	}

	// done
	return nil, nil, io.EOF
}

func (i *recordIterator) Close() {
	i.dataR.Close()
}
