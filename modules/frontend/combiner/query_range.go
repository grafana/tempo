package combiner

import (
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

var _ GRPCCombiner[*tempopb.QueryRangeResponse] = (*genericCombiner[*tempopb.QueryRangeResponse])(nil)

// NewQueryRange returns a query range combiner. It takes a isProm parameter to control the response format
func NewQueryRange(isProm bool) Combiner { // jpe - use isProm? or do in the handler?
	combiner := traceql.QueryRangeCombiner{}

	return &genericCombiner[*tempopb.QueryRangeResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.QueryRangeResponse { return &tempopb.QueryRangeResponse{} },
		current:        &tempopb.QueryRangeResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(partial *tempopb.QueryRangeResponse, final *tempopb.QueryRangeResponse) error { // jpe - final?
			// if partial.Metrics != nil { // jpe ?? totalJobs/completedJobs?
			// 	// there is a coordination with the search sharder here. normal responses
			// 	// will never have total jobs set, but they will have valid Inspected* values
			// 	// a special response is sent back from the sharder with no traces but valid Total* values
			// 	// if TotalJobs is nonzero then assume its the special response
			// 	if partial.Metrics.TotalJobs == 0 {
			// 		final.Metrics.CompletedJobs++

			// 		final.Metrics.InspectedBytes += partial.Metrics.InspectedBytes
			// 		final.Metrics.InspectedTraces += partial.Metrics.InspectedTraces
			// 	} else {
			// 		final.Metrics.TotalBlocks += partial.Metrics.TotalBlocks
			// 		final.Metrics.TotalJobs += partial.Metrics.TotalJobs
			// 		final.Metrics.TotalBlockBytes += partial.Metrics.TotalBlockBytes
			// 	}
			// }

			combiner.Combine(partial)

			return nil
		},
		finalize: func(final *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) {
			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}
			return resp, nil
		},
		diff: func(current *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) { // jpe - actually diff
			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}
			return resp, nil
		},
	}
}

func NewTypedQueryRange(isProm bool) GRPCCombiner[*tempopb.QueryRangeResponse] {
	return NewQueryRange(isProm).(GRPCCombiner[*tempopb.QueryRangeResponse])
}
