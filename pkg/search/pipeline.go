package search

import (
	"github.com/grafana/tempo/pkg/tempopb"
)

const SecretExhaustiveSearchTag = "x-dbg-exhaustive" // jpe do we need this?

type Pipeline struct {
}

// jpe what do i really need here?
func NewSearchPipeline(req *tempopb.SearchRequest) Pipeline {
	// jpe gutted all the filter building?

	return Pipeline{}
}
