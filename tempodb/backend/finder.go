package backend

import (
	"bytes"
	"io"
	"sort"
)

type Finder interface {
	Find(id ID) ([]byte, error)
}

type finder struct {
	ra            io.ReaderAt
	sortedRecords []*Record
	combiner      ObjectCombiner
}

func NewFinder(sortedRecords []*Record, ra io.ReaderAt, combiner ObjectCombiner) Finder {
	return &finder{
		ra:            ra,
		sortedRecords: sortedRecords,
	}
}

func NewDedupingFinder(sortedRecords []*Record, ra io.ReaderAt, combiner ObjectCombiner) Finder {
	return &finder{
		ra:            ra,
		sortedRecords: sortedRecords,
		combiner:      combiner,
	}
}

func (f *finder) Find(id ID) ([]byte, error) {
	i := sort.Search(len(f.sortedRecords), func(idx int) bool {
		return bytes.Compare(f.sortedRecords[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(f.sortedRecords) {
		return nil, nil
	}

	record := f.sortedRecords[i]

	buff := make([]byte, record.Length)
	_, err := f.ra.ReadAt(buff, int64(record.Start))
	if err != nil {
		return nil, err
	}

	iter := NewIterator(bytes.NewReader(buff))

	if f.combiner != nil {
		iter, err = NewDedupingIterator(iter, f.combiner)
		if err != nil {
			return nil, err
		}
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
