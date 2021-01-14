package v0

import (
	"bytes"
	"io"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding"
)

type finder struct {
	ra            io.ReaderAt
	sortedRecords []*encoding.Record
}

func NewFinder(sortedRecords []*encoding.Record, ra io.ReaderAt) encoding.Finder {
	return &finder{
		ra:            ra,
		sortedRecords: sortedRecords,
	}
}

func (f *finder) Find(id encoding.ID) ([]byte, error) {
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
