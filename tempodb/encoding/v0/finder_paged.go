package v0

import (
	"bytes"
	"errors"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type pagedFinder struct {
	r             common.PageReader
	sortedRecords []*common.Record
	combiner      common.ObjectCombiner
}

// NewPagedFinder returns a paged. This finder is used for searching
//  a set of records and returning an object. If a set of consecutive records has
//  matching ids they will be combined using the ObjectCombiner.
func NewPagedFinder(sortedRecords []*common.Record, r common.PageReader, combiner common.ObjectCombiner) common.Finder {
	return &pagedFinder{
		r:             r,
		sortedRecords: sortedRecords,
		combiner:      combiner,
	}
}

func (f *pagedFinder) Find(id common.ID) ([]byte, error) {
	i := sort.Search(len(f.sortedRecords), func(idx int) bool {
		return bytes.Compare(f.sortedRecords[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(f.sortedRecords) {
		return nil, nil
	}

	var bytesFound []byte

	for {
		record := f.sortedRecords[i]

		bytesOne, err := f.findOne(id, record)
		if err != nil {
			return nil, err
		}

		bytesFound = f.combiner.Combine(bytesFound, bytesOne)

		// we need to check the next record to see if it also matches our id
		i++
		if i >= len(f.sortedRecords) {
			break
		}

		if !bytes.Equal(f.sortedRecords[i].ID, id) {
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
	iter, err = NewDedupingIterator(iter, f.combiner)
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
