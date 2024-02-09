package combiner

import (
	"io"

	"github.com/grafana/tempo/pkg/tempopb"
)

var _ Combiner = (*genericCombiner[*tempopb.SearchResponse])(nil)

// NewNoOp returns a combiner that doesn't is a no op, and doesn't combine.
// It is used in search streaming, in search streaming keeps trek of search progress
// and combines result on its own from the multi-tenant search.
func NewNoOp() Combiner {
	return &genericCombiner[*tempopb.SearchResponse]{
		code:  200,
		final: &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(body io.ReadCloser, final *tempopb.SearchResponse) error {
			// no op
			return nil
		},
		result: func(response *tempopb.SearchResponse) (string, error) {
			// no op
			return "", nil
		},
	}
}
