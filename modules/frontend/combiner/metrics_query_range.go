package combiner

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/grafana/tempo/pkg/api"
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

	c := &genericCombiner[*tempopb.QueryRangeResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.QueryRangeResponse { return &tempopb.QueryRangeResponse{} },
		current:        &tempopb.QueryRangeResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(partial *tempopb.QueryRangeResponse, _ *tempopb.QueryRangeResponse, _ PipelineResponse) error {
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
			sortResponse(resp)
			attachExemplars(req, resp)

			return resp, nil
		},
		diff: func(_ *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) {
			resp := combiner.Diff()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}
			sortResponse(resp)
			attachExemplars(req, resp)

			return resp, nil
		},
	}

	initHTTPCombiner(c, api.HeaderAcceptJSON)

	return c, nil
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
		sort.Slice(series.Exemplars, func(i, j int) bool {
			return series.Exemplars[i].TimestampMs < series.Exemplars[j].TimestampMs
		})
	}
}

// attachExemplars to the final series outputs. Placeholder exemplars for things like rate()
// have NaNs, and we can't attach them until the very end.
func attachExemplars(req *tempopb.QueryRangeRequest, res *tempopb.QueryRangeResponse) {
	for _, ss := range res.Series {
		for i, e := range ss.Exemplars {

			// Only needed for NaNs
			if !math.IsNaN(e.Value) {
				continue
			}

			exemplarInterval := traceql.IntervalOfMs(e.TimestampMs, req.Start, req.End, req.Step)

			// Look for sample in the same slot.
			// BinarySearch is possible because all samples were sorted previously.
			j, ok := slices.BinarySearchFunc(ss.Samples, exemplarInterval, func(s tempopb.Sample, _ int) int {
				// NOTE - Look for sample in same interval, not same value.
				si := traceql.IntervalOfMs(s.TimestampMs, req.Start, req.End, req.Step)

				// This returns negative, zero, or positive
				return si - exemplarInterval
			})
			if ok {
				ss.Exemplars[i].Value = ss.Samples[j].Value
			}
		}
	}
}
