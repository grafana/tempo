package v1

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

// Deduping finder is used by the complete block
//  jpe - we should also transition to using it in the BackendBlock Find?
//  will basically work like v0 except will need to decompress before iterating through the data

type pagedFinder struct {
}

// NewPagedFinder returns a pagedFinder. This finder is used for searching
//  a set of records and returning an object. If a set of consecutive records has
//  matching ids they will be combined using the ObjectCombiner.
func NewPagedFinder(sortedRecords []*common.Record, ra io.ReaderAt, combiner common.ObjectCombiner) common.Finder {
	return &pagedFinder{}
}

func (f *pagedFinder) Find(id common.ID) ([]byte, error) {
	return nil, nil
}
