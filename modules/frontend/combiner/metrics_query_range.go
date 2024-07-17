package combiner

import (
	"sort"
	"strings"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

var _ GRPCCombiner[*tempopb.QueryRangeResponse] = (*genericCombiner[*tempopb.QueryRangeResponse])(nil)

// NewQueryRange returns a query range combiner.
func NewQueryRange(req *tempopb.QueryRangeRequest, trackDiffs bool) (Combiner, error) {
	combiner, err := traceql.QueryRangeCombinerFor(req, traceql.AggregateModeFinal, trackDiffs)
	if err != nil {
		return nil, err
	}

	return &genericCombiner[*tempopb.QueryRangeResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.QueryRangeResponse { return &tempopb.QueryRangeResponse{} },
		current:        &tempopb.QueryRangeResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(partial *tempopb.QueryRangeResponse, _ *tempopb.QueryRangeResponse, resp PipelineResponse) error {
			if partial.Metrics != nil {
				// this is a coordination between the sharder and combiner. the sharder returns one response with summary metrics
				// only. the combiner correctly takes and accumulates that job. however, if the response has no jobs this is
				// an indicator this is a "real" response so we set CompletedJobs to 1 to increment in the combiner.
				if partial.Metrics.TotalJobs == 0 {
					partial.Metrics.CompletedJobs = 1
				}
			}

			samplingRate := resp.RequestData()
			if samplingRate != nil {
				fRate := samplingRate.(float64)

				if fRate <= 1.0 {
					// Set final sampling rate after integer rounding
					// Multiply up the sampling rate
					for _, series := range partial.Series {
						for i, sample := range series.Samples {
							sample.Value *= 1.0 / fRate
							series.Samples[i] = sample
						}
					}
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
			sortResponse(resp)
			return resp, nil
		},
		diff: func(_ *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) {
			resp := combiner.Diff()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}
			sortResponse(resp)
			return resp, nil
		},
	}, nil
}

func NewTypedQueryRange(req *tempopb.QueryRangeRequest, trackDiffs bool) (GRPCCombiner[*tempopb.QueryRangeResponse], error) {
	c, err := NewQueryRange(req, trackDiffs)
	if err != nil {
		return nil, err
	}

	return c.(GRPCCombiner[*tempopb.QueryRangeResponse]), nil
}

func sortResponse(res *tempopb.QueryRangeResponse) {
	// Sort all output, series alphabetically, samples by time
	sort.SliceStable(res.Series, func(i, j int) bool {
		return strings.Compare(res.Series[i].PromLabels, res.Series[j].PromLabels) == -1
	})
	for _, series := range res.Series {
		sort.Slice(series.Samples, func(i, j int) bool {
			return series.Samples[i].TimestampMs < series.Samples[j].TimestampMs
		})
	}
}
