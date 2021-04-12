package encoding

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

// Finder is capable of finding the requested ID
type Finder interface {
	Find(context.Context, common.ID) ([]byte, error)
}

type pagedFinder struct {
	r        common.DataReader
	index    common.IndexReader
	combiner common.ObjectCombiner
	objectRW common.ObjectReaderWriter
}

// NewPagedFinder returns a paged. This finder is used for searching
//  a set of records and returning an object. If a set of consecutive records has
//  matching ids they will be combined using the ObjectCombiner.
func NewPagedFinder(index common.IndexReader, r common.DataReader, combiner common.ObjectCombiner, objectRW common.ObjectReaderWriter) Finder {
	return &pagedFinder{
		r:        r,
		index:    index,
		combiner: combiner,
		objectRW: objectRW,
	}
}

func (f *pagedFinder) Find(ctx context.Context, id common.ID) ([]byte, error) {
	var bytesFound []byte
	record, i, err := f.index.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	if record == nil {
		return nil, nil
	}

	for {
		bytesOne, err := f.findOne(ctx, id, record)
		if err != nil {
			return nil, err
		}

		if f.combiner == nil {
			bytesFound = bytesOne
			break
		}

		bytesFound = f.combiner.Combine(bytesFound, bytesOne)

		// we need to check the next record to see if it also matches our id
		i++
		record, err = f.index.At(ctx, i)
		if err != nil {
			return nil, err
		}
		if record == nil {
			break
		}
		if !bytes.Equal(record.ID, id) {
			break
		}
	}

	return bytesFound, nil
}

func (f *pagedFinder) findOne(ctx context.Context, id common.ID, record *common.Record) ([]byte, error) {
	pages, err := f.r.Read(ctx, []*common.Record{record})
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, errors.New("unexpected 0 length pages in findOne")
	}

	// dataReader is expected to return pages in the v0 format.  so this works
	iter := NewIterator(bytes.NewReader(pages[0]), f.objectRW)
	if f.combiner != nil {
		iter, err = NewDedupingIterator(iter, f.combiner)
	}
	if err != nil {
		return nil, err
	}

	for {
		foundID, b, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if bytes.Equal(foundID, id) {
			return b, nil
		}
	}

	return nil, nil
}
