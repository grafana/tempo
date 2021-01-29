package v1

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

// Deduping finder is used by the complete block
//  jpe - we should also transition to using it in the BackendBlock Find?
//  will basically work like v0 except will need to decompress before iterating through the data

type dedupingFinder struct {
}

// NewDedupingFinder returns a dedupingFinder. This finder is used for searching
//  a set of records and returning an object. If a set of consecutive records has
//  matching ids they will be combined using the ObjectCombiner.
func NewDedupingFinder(sortedRecords []*common.Record, ra io.ReaderAt, combiner common.ObjectCombiner) common.Finder {
	return &dedupingFinder{}
}

func (f *dedupingFinder) Find(id common.ID) ([]byte, error) {
	return nil, nil
}
