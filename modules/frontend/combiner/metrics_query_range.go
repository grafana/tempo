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
func NewQueryRange(req *tempopb.QueryRangeRequest) (Combiner, error) {
	combiner, err := traceql.QueryRangeCombinerFor(req, traceql.AggregateModeFinal)
	if err != nil {
		return nil, err
	}

	var prevResp *tempopb.QueryRangeResponse

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
			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}
			sortResponse(resp)

			// compare with prev resp and only return diffs
			diff := diffResponse(prevResp, resp)
			// store resp for next diff
			prevResp = resp

			attachExemplars(req, diff)

			return diff, nil
		},
	}

	initHTTPCombiner(c, api.HeaderAcceptJSON)

	return c, nil
}

func NewTypedQueryRange(req *tempopb.QueryRangeRequest) (GRPCCombiner[*tempopb.QueryRangeResponse], error) {
	c, err := NewQueryRange(req)
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

// jpe - review below
// ++benchmarks
// can we do this in place? in current? we can't modify prev b/c we need it for next diff
// if we attempt this its important that we have not previously returned a pointer to curr outside
// the combiner. if we did it would be possible that the grpc layer is marshalling it while we modify it
// assume sorted
func diffResponse(prev, curr *tempopb.QueryRangeResponse) *tempopb.QueryRangeResponse {
	if prev == nil {
		// if we do the thing suggested above we need a full copy here
		return curr
	}

	// everything is sorted
	seriesIdx := 0
	diff := &tempopb.QueryRangeResponse{}
	for _, s := range curr.Series {
		// is this a series that's new in curr that wasn't in prev? this check assumes that
		// a series can not be removed from the output as the input series are combined
		if seriesIdx >= len(prev.Series) || s.PromLabels != prev.Series[seriesIdx].PromLabels {
			diff.Series = append(diff.Series, s)
			continue
		}

		// promlabels are the same, have to check individual samples
		// copy in labels and take any exemplars that exist
		diffSeries := &tempopb.TimeSeries{
			Labels:     s.Labels,
			PromLabels: s.PromLabels,
			Exemplars:  s.Exemplars, // taking all current exemplars. improve?
		}

		// samples are sorted, so we can do this in a single pass
		dSamplesIdx := 0
		for _, sample := range s.Samples {
			// if this sample is not in the previous response, add it to the diff
			if dSamplesIdx >= len(prev.Series[seriesIdx].Samples) || sample.TimestampMs != prev.Series[seriesIdx].Samples[dSamplesIdx].TimestampMs { // jpe use > or < instead of == to make the alg more robust?
				diffSeries.Samples = append(diffSeries.Samples, sample)
				continue
			}

			// if we get here, the sample is in both responses. only copy if the value is different
			if sample.Value != prev.Series[seriesIdx].Samples[dSamplesIdx].Value {
				diffSeries.Samples = append(diffSeries.Samples, sample)
			}

			dSamplesIdx++
		}

		if len(diffSeries.Samples) > 0 {
			diff.Series = append(diff.Series, diffSeries)
		}
		seriesIdx++
	}

	// no need to diff metrics, just take current
	diff.Metrics = curr.Metrics

	return diff
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
