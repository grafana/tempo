package combiner

import (
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

var _ GRPCCombiner[*tempopb.QueryRangeResponse] = (*genericCombiner[*tempopb.QueryRangeResponse])(nil)

// NewQueryRange returns a query range combiner.
func NewQueryRange() Combiner {
	combiner := traceql.QueryRangeCombiner{}

	return &genericCombiner[*tempopb.QueryRangeResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.QueryRangeResponse { return &tempopb.QueryRangeResponse{} },
		current:        &tempopb.QueryRangeResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(partial *tempopb.QueryRangeResponse, _ *tempopb.QueryRangeResponse) error {
			if partial.Metrics != nil {
				// this is a coordination between the sharder and combiner. the sharder returns one response with summary metrics
				// only. the combiner correctly takes and accumulates that job. however, if the response has no jobs this is
				// an indicator this is a "real" response so we set CompletedJobs to 1 to increment in the combiner.
				if partial.Metrics.TotalJobs == 0 {
					partial.Metrics.CompletedJobs = 1
				}
			}

			combiner.Combine(partial)

			return nil
		},
		finalize: func(_ *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) {
			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}
			return resp, nil
		},
		// todo: the diff method still returns the full response every time. find a way to diff
		diff: func(_ *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) {
			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}
			return resp, nil
		},
	}
}

func NewTypedQueryRange() GRPCCombiner[*tempopb.QueryRangeResponse] {
	return NewQueryRange().(GRPCCombiner[*tempopb.QueryRangeResponse])
}

/* jpe - restore
res := c.Response()
res.Metrics.CompletedJobs = uint32(startedReqs)
res.Metrics.TotalBlocks = uint32(totalBlocks)
res.Metrics.TotalBlockBytes = uint64(totalBlockBytes)

// Sort all output, series alphabetically, samples by time
sort.SliceStable(res.Series, func(i, j int) bool {
	return strings.Compare(res.Series[i].PromLabels, res.Series[j].PromLabels) == -1
})
for _, series := range res.Series {
	sort.Slice(series.Samples, func(i, j int) bool {
		return series.Samples[i].TimestampMs < series.Samples[j].TimestampMs
	})
}
*/
