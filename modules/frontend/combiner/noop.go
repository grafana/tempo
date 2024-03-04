package combiner

import (
	"github.com/grafana/tempo/pkg/tempopb"
)

var _ GRPCCombiner[*tempopb.SearchResponse] = (*genericCombiner[*tempopb.SearchResponse])(nil)

// NewNoOp returns a combiner that doesn't is a no op, and doesn't combine.
// It is used in search streaming, in search streaming keeps trek of search progress
// and combines result on its own from the multi-tenant search.
func NewNoOp() Combiner {
	return &genericCombiner[*tempopb.SearchResponse]{
		httpStatusCode: 200,
		current:        &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
		new:            func() *tempopb.SearchResponse { return &tempopb.SearchResponse{} },
		combine: func(partial, final *tempopb.SearchResponse) error {
			// no op
			return nil
		},
		finalize: func(final *tempopb.SearchResponse) (*tempopb.SearchResponse, error) {
			// no op
			return nil, nil
		},
	}
}
