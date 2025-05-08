package combiner

import (
	"fmt"
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
func NewQueryRange(req *tempopb.QueryRangeRequest, maxSeriesLimit int) (Combiner, error) {
	// if a limit is being enforced, honor the request if it is less than the limit
	// else set it to max limit
	maxSeries := int(req.MaxSeries)
	if maxSeriesLimit > 0 && int(req.MaxSeries) > maxSeriesLimit || req.MaxSeries == 0 {
		maxSeries = maxSeriesLimit
	}
	combiner, err := traceql.QueryRangeCombinerFor(req, traceql.AggregateModeFinal, maxSeries)
	if err != nil {
		return nil, err
	}

	var prevResp *tempopb.QueryRangeResponse
	maxSeriesReachedErrorMsg := fmt.Sprintf("Response exceeds maximum series limit of %d, a partial response is returned. Warning: the accuracy of each individual value is not guaranteed.", maxSeries)

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
			truncateResponse(resp, maxSeries, req)

			// partial is when the max series is reached either in the querier or generators
			// since it might have been truncated - we need to add the warning the it may be inaccurate
			// max series reached is when the max series is actually reached at the query-frontend level
			if combiner.MaxSeriesReached() || combiner.Partial() {
				resp.Status = tempopb.PartialStatus_PARTIAL
				resp.Message = maxSeriesReachedErrorMsg
			}
			attachExemplars(req, resp)

			return resp, nil
		},
		diff: func(_ *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) {
			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}

			sortResponse(resp)
			truncateResponse(resp, maxSeries, req)
			attachExemplars(req, resp)

			// compare with prev resp and only return diffs
			diff := diffResponse(prevResp, resp)
			// store resp for next diff
			prevResp = resp

			if combiner.Partial() || combiner.MaxSeriesReached() {
				diff.Status = tempopb.PartialStatus_PARTIAL
				diff.Message = maxSeriesReachedErrorMsg
			}

			return diff, nil
		},
		quit: func(_ *tempopb.QueryRangeResponse) bool {
			return combiner.MaxSeriesReached()
		},
	}

	initHTTPCombiner(c, api.HeaderAcceptJSON)

	return c, nil
}

func NewTypedQueryRange(req *tempopb.QueryRangeRequest, maxSeries int) (GRPCCombiner[*tempopb.QueryRangeResponse], error) {
	c, err := NewQueryRange(req, maxSeries)
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

// diffResponse takes two QueryRangeResponses and returns a new QueryRangeResponse that contains only the differences between the two.
// it creates a completely new response, so the input responses are not modified. an in place diff would be nice for memory savings, but
// the diffResponse is returned and marshalled into proto. if we modify an object while it's being marshalled this can cause a panic
func diffResponse(prev, curr *tempopb.QueryRangeResponse) *tempopb.QueryRangeResponse {
	if prev == nil {
		return curr
	}

	seriesIdx := 0
	diff := &tempopb.QueryRangeResponse{
		Series: make([]*tempopb.TimeSeries, 0, len(curr.Series)/2), // prealloc half of series. untuned
	}

	// series are sorted, so we can do this in a single pass
	for _, s := range curr.Series {
		// is this a series that's new in curr that wasn't in prev? this check assumes that
		// a series can not be removed from the output as the input series are combined
		if seriesIdx >= len(prev.Series) || s.PromLabels != prev.Series[seriesIdx].PromLabels {
			diff.Series = append(diff.Series, s)
			continue
		}

		// promlabels are the same, have to check individual samples
		// just copy forward labels and exemplars
		diffSeries := &tempopb.TimeSeries{
			Labels:     s.Labels,
			PromLabels: s.PromLabels,
			Exemplars:  make([]tempopb.Exemplar, 0, len(s.Exemplars)/10), // prealloc 10% of exemplars. untuned
			Samples:    make([]tempopb.Sample, 0, len(s.Samples)/10),     // prealloc 10% of samples. untuned
		}

		// samples are sorted, so we can do this in a single pass
		dSamplesIdx := 0

		for _, sample := range s.Samples {
			// if this sample is not in the previous response, add it to the diff
			if dSamplesIdx >= len(prev.Series[seriesIdx].Samples) || sample.TimestampMs != prev.Series[seriesIdx].Samples[dSamplesIdx].TimestampMs {
				diffSeries.Samples = append(diffSeries.Samples, sample)
				continue
			}

			// if we get here, the sample is in both responses. only copy if the value is different
			if sample.Value != prev.Series[seriesIdx].Samples[dSamplesIdx].Value {
				diffSeries.Samples = append(diffSeries.Samples, sample)
			}

			dSamplesIdx++
		}

		// diff exemplars
		// exemplars are sorted, so we can do this in a single pass
		dExemplarsIdx := 0
		for _, exemplar := range s.Exemplars {
			// if this exemplar is not in the previous response, add it to the diff. technically two exemplars could share the same timestamp, if they do AND are sorted in opposite order of the prev response
			// then they will just both be marshalled into the output proto. this rare case seems preferable to always checking all exemplar labels for equality
			if dExemplarsIdx >= len(prev.Series[seriesIdx].Exemplars) || exemplar.TimestampMs != prev.Series[seriesIdx].Exemplars[dExemplarsIdx].TimestampMs {
				diffSeries.Exemplars = append(diffSeries.Exemplars, exemplar)
				continue
			}

			// if we get here, the exemplar is in both responses. only copy if the value is different
			if exemplar.Value != prev.Series[seriesIdx].Exemplars[dExemplarsIdx].Value {
				diffSeries.Exemplars = append(diffSeries.Exemplars, exemplar)
			}

			dExemplarsIdx++
		}

		if len(diffSeries.Samples) > 0 || len(diffSeries.Exemplars) > 0 {
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

func truncateResponse(resp *tempopb.QueryRangeResponse, maxSeries int, req *tempopb.QueryRangeRequest) {
	if maxSeries == 0 || len(resp.Series) <= maxSeries {
		return
	}
	rootexpr, err := traceql.Parse(req.Query)
	if err != nil {
		return
	}
	// if this query is a compare, we need make sure there is one total series per key
	// only include if both total series and corresponding count series exist
	const (
		baselinePrefix       = "baseline"
		selectionPrefix      = "selection"
		baselineTotalPrefix  = "baseline_total"
		selectionTotalPrefix = "selection_total"
		metaTypePrefix       = "__meta_type"
	)
	if _, ok := rootexpr.MetricsPipeline.(*traceql.MetricsCompare); ok {
		baselineCountMap := make(map[string][]int) // count is one single series for each key/value pair
		baselineTotalMap := make(map[string]int)   // total is always a single series for each key
		selectionCountMap := make(map[string][]int)
		selectionTotalMap := make(map[string]int)
		results := make([]*tempopb.TimeSeries, maxSeries)
		resultsIdx := 0

		for i, series := range resp.Series {
			for _, label := range series.Labels {
				if label.Key != metaTypePrefix {
					// record the corresponding index
					if strings.Contains(series.PromLabels, baselineTotalPrefix) {
						baselineTotalMap[label.Key] = i
						continue
					}
					// if it's not baselineTotal but has baseline it's baselineCount
					if strings.Contains(series.PromLabels, baselinePrefix) {
						baselineCountMap[label.Key] = append(baselineCountMap[label.Key], i)
						continue
					}
					if strings.Contains(series.PromLabels, selectionTotalPrefix) {
						selectionTotalMap[label.Key] = i
						continue
					}
					// if it's not selectionTotal but has selection it's selectionCount
					if strings.Contains(series.PromLabels, selectionPrefix) {
						selectionCountMap[label.Key] = append(selectionCountMap[label.Key], i)
						continue
					}
				}
			}
		}

		// do baseline first,
		// the total is more important so just check total first
		for a, i := range baselineTotalMap {
			// check if we have a count for this total
			if _, ok := baselineCountMap[a]; ok && resultsIdx < maxSeries {
				results[resultsIdx] = resp.Series[i]
				resultsIdx++
				for _, series := range baselineCountMap[a] {
					if resultsIdx >= maxSeries {
						break
					}
					results[resultsIdx] = resp.Series[series]
					resultsIdx++
				}
			}
		}
		// then do selection
		for a, i := range selectionTotalMap {
			// check if we have a count for this total
			if _, ok := selectionCountMap[a]; ok && resultsIdx < maxSeries {
				results[resultsIdx] = resp.Series[i]
				resultsIdx++
				for _, series := range selectionCountMap[a] {
					if resultsIdx >= maxSeries {
						break
					}
					results[resultsIdx] = resp.Series[series]
					resultsIdx++
				}
			}
		}
		resp.Series = results[:resultsIdx]
		return
	}
	// otherwise just truncate
	resp.Series = resp.Series[:maxSeries]
}
