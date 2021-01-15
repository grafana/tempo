package v0

import (
	"bytes"
	"io"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding/index"
)

type dedupingFinder struct {
	ra            io.ReaderAt
	sortedRecords []*index.Record
	combiner      index.ObjectCombiner
}

// NewDedupingFinder returns a dedupingFinder. This finder is used for searching
//  a set of records and returning an object. If a set of consecutive records has
//  matching ids they will be combined using the ObjectCombiner.
func NewDedupingFinder(sortedRecords []*index.Record, ra io.ReaderAt, combiner index.ObjectCombiner) index.Finder {
	return &dedupingFinder{
		ra:            ra,
		sortedRecords: sortedRecords,
		combiner:      combiner,
	}
}

func (f *dedupingFinder) Find(id index.ID) ([]byte, error) {
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

func (f *dedupingFinder) findOne(id index.ID, record *index.Record) ([]byte, error) {
	buff := make([]byte, record.Length)
	_, err := f.ra.ReadAt(buff, int64(record.Start))
	if err != nil {
		return nil, err
	}

	iter := NewIterator(bytes.NewReader(buff))
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
