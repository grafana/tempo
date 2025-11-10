package combiner

import (
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"

	"github.com/grafana/tempo/modules/frontend/shardtracker"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

// QueryRangeJobResponse wraps shardtracker.JobMetadata and implements PipelineResponse.
type QueryRangeJobResponse struct {
	shardtracker.JobMetadata
}

func (q *QueryRangeJobResponse) HTTPResponse() *http.Response {
	return nil
}

func (q *QueryRangeJobResponse) RequestData() any {
	return nil
}

func (q *QueryRangeJobResponse) IsMetadata() bool {
	return true
}

var _ PipelineResponse = (*QueryRangeJobResponse)(nil)
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

	completionTracker := &shardtracker.CompletionTracker{}
	maxSeriesReachedErrorMsg := fmt.Sprintf("Response exceeds maximum series limit of %d, a partial response is returned. Warning: the accuracy of each individual value is not guaranteed.", maxSeries)

	metricsCombiner := NewQueryRangeMetricsCombiner()
	c := &genericCombiner[*tempopb.QueryRangeResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.QueryRangeResponse { return &tempopb.QueryRangeResponse{} },
		current:        &tempopb.QueryRangeResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(partial *tempopb.QueryRangeResponse, _ *tempopb.QueryRangeResponse, resp PipelineResponse) error {
			combiner.Combine(partial)
			metricsCombiner.Combine(partial.Metrics, resp)

			// Track shard completion
			if shardIdx, ok := resp.RequestData().(int); ok {
				completionTracker.AddShardIdx(shardIdx)
			}

			return nil
		},
		metadata: func(resp PipelineResponse, final *tempopb.QueryRangeResponse) error {
			// jpe - set metadata on the final response?
			if qr, ok := resp.(*QueryRangeJobResponse); ok && qr != nil {
				qrMetrics := &tempopb.SearchMetrics{
					TotalBlocks:     uint32(qr.TotalBlocks), //nolint:gosec
					TotalJobs:       uint32(qr.TotalJobs),   //nolint:gosec
					TotalBlockBytes: qr.TotalBytes,
				}
				metricsCombiner.Combine(qrMetrics, resp)

				completionTracker.AddShards(qr.Shards)
			}
			return nil
		},
		finalize: func(_ *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) {
			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}

			sortResponse(resp)
			if combiner.MaxSeriesReached() {
				// Truncating the final response because even if we bail as soon as len(resp.Series) >= maxSeries
				// it's possible that the last response pushed us over the max series limit.
				resp.Series = resp.Series[:maxSeries]
				resp.Status = tempopb.PartialStatus_PARTIAL
				resp.Message = maxSeriesReachedErrorMsg
			}
			attachExemplars(req, resp)
			resp.Metrics = metricsCombiner.Metrics
			return resp, nil
		},
		diff: func(_ *tempopb.QueryRangeResponse) (*tempopb.QueryRangeResponse, error) {
			// Check if any shards have completed
			completedThrough := completionTracker.CompletedThroughSeconds()

			// If no shards have completed yet, return empty response
			if completedThrough == 0 {
				return &tempopb.QueryRangeResponse{
					Series:  []*tempopb.TimeSeries{},
					Metrics: metricsCombiner.Metrics,
				}, nil
			}
			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}

			// jpe do the below completedThrough trimming before this logic?
			sortResponse(resp)
			if combiner.MaxSeriesReached() {
				// Truncating the final response because even if we bail as soon as len(resp.Series) >= maxSeries
				// it's possible that the last response pushed us over the max series limit.
				resp.Series = resp.Series[:maxSeries]
			}
			attachExemplars(req, resp)

			// Find first sample and exemplar more recent than completedThrough and send everything after it
			if metricsCombiner.Metrics.CompletedJobs == metricsCombiner.Metrics.TotalJobs && metricsCombiner.Metrics.TotalJobs > 0 { // jpe - why is this needed?
				completedThrough = 1
			}

			// jpe: todo: track "lastCompletedThrough" and only send what's necessary. on the final response send all
			completedThroughMs := int64(completedThrough) * 1000
			for _, series := range resp.Series {
				idx := 0
				for idx < len(series.Samples) && series.Samples[idx].TimestampMs <= completedThroughMs {
					idx++
				}
				series.Samples = series.Samples[idx:]

				idx = 0
				for idx < len(series.Exemplars) && series.Exemplars[idx].TimestampMs <= completedThroughMs {
					idx++
				}
				series.Exemplars = series.Exemplars[idx:]
			}

			if combiner.MaxSeriesReached() {
				resp.Status = tempopb.PartialStatus_PARTIAL
				resp.Message = maxSeriesReachedErrorMsg
			}

			return resp, nil
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
		li := len(res.Series[i].Labels)
		lj := len(res.Series[j].Labels)
		if li != lj {
			return li < lj
		}
		for k := range res.Series[i].Labels {
			ki := res.Series[i].Labels[k].Key
			kj := res.Series[j].Labels[k].Key
			if ki != kj {
				return ki < kj
			}

			si := res.Series[i].Labels[k].Value.String()
			sj := res.Series[j].Labels[k].Value.String()
			if si != sj {
				return si < sj
			}
		}
		return false
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
	intervalMapper := traceql.NewIntervalMapperFromReq(req)
	for _, ss := range res.Series {
		for i, e := range ss.Exemplars {

			// Only needed for NaNs
			if !math.IsNaN(e.Value) {
				continue
			}

			exemplarInterval := intervalMapper.IntervalMs(e.TimestampMs)

			// Look for sample in the same slot.
			// BinarySearch is possible because all samples were sorted previously.
			j, ok := slices.BinarySearchFunc(ss.Samples, exemplarInterval, func(s tempopb.Sample, _ int) int {
				// NOTE - Look for sample in same interval, not same value.
				si := intervalMapper.IntervalMs(s.TimestampMs)

				// This returns negative, zero, or positive
				return si - exemplarInterval
			})
			if ok {
				ss.Exemplars[i].Value = ss.Samples[j].Value
			}
		}
	}
}
