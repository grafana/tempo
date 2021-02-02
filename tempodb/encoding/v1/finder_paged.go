package v1

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// jpe comment
// NewPagedFinder returns a pagedFinder. This finder is used for searching
//  a set of records and returning an object. If a set of consecutive records has
//  matching ids they will be combined using the ObjectCombiner.
func NewPagedFinder(index common.IndexReader, r common.PageReader, combiner common.ObjectCombiner) common.Finder {
	return v0.NewPagedFinder(index, r, combiner)
}
