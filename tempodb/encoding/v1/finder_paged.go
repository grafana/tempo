package v1

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// NewPagedFinder returns a v0.pagedFinder.  There are no changes
// to logic from the v0 finder and all compression changes are handled in the pageReader
func NewPagedFinder(index common.IndexReader, r common.PageReader, combiner common.ObjectCombiner) common.Finder {
	return v0.NewPagedFinder(index, r, combiner)
}
