package v0

import (
	"bytes"
	"errors"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type pagedFinder struct {
	r        common.PageReader
	index    common.IndexReader
	combiner common.ObjectCombiner
}

// NewPagedFinder returns a paged. This finder is used for searching
//  a set of records and returning an object. If a set of consecutive records has
//  matching ids they will be combined using the ObjectCombiner.
func NewPagedFinder(index common.IndexReader, r common.PageReader, combiner common.ObjectCombiner) common.Finder {
	return &pagedFinder{
		r:        r,
		index:    index,
		combiner: combiner,
	}
}

func (f *pagedFinder) Find(id common.ID) ([]byte, error) {
	var bytesFound []byte
	record, i := f.index.Find(id)

	if record == nil {
		return nil, nil
	}

	for {
		bytesOne, err := f.findOne(id, record)
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
		record = f.index.At(i)
		if record == nil {
			break
		}
		if !bytes.Equal(record.ID, id) {
			break
		}
	}

	return bytesFound, nil
}

func (f *pagedFinder) findOne(id common.ID, record *common.Record) ([]byte, error) {
	pages, err := f.r.Read([]*common.Record{record})
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, errors.New("unexpected 0 length pages in findOne")
	}

	iter := NewIterator(bytes.NewReader(pages[0]))
	if f.combiner != nil {
		iter, err = NewDedupingIterator(iter, f.combiner)
	}
	if err != nil {
		return nil, err
	}

	for {
		foundID, b, err := iter.Next()
		if foundID == nil {
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
