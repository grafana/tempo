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

var (
	_ PipelineResponse                          = (*QueryRangeJobResponse)(nil)
	_ GRPCCombiner[*tempopb.QueryRangeResponse] = (*genericCombiner[*tempopb.QueryRangeResponse])(nil)
)

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
	lastCompletedThrough := shardtracker.TimestampNever
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
		metadata: func(resp PipelineResponse, _ *tempopb.QueryRangeResponse) error {
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
				if len(resp.Series) > maxSeries {
					resp.Series = resp.Series[:maxSeries]
				}
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

			// If we are still waiting for new data, then send an interim empty diff.
			if completedThrough == shardtracker.TimestampUnknown || completedThrough == lastCompletedThrough {
				return &tempopb.QueryRangeResponse{
					Series:  []*tempopb.TimeSeries{},
					Metrics: metricsCombiner.Metrics,
				}, nil
			}

			resp := combiner.Response()
			if resp == nil {
				resp = &tempopb.QueryRangeResponse{}
			}

			// only trim the response if we're not at the end of the stream. for the final response, we'll send all the data.
			if completedThrough != shardtracker.TimestampAlways {
				trimSeriesToCompletedWindow(resp.Series, lastCompletedThrough, completedThrough)
			}

			// Update lastCompletedThrough for next diff
			lastCompletedThrough = completedThrough

			sortResponse(resp)
			if combiner.MaxSeriesReached() {
				// Truncating the final response because even if we bail as soon as len(resp.Series) >= maxSeries
				// it's possible that the last response pushed us over the max series limit.
				if len(resp.Series) > maxSeries {
					resp.Series = resp.Series[:maxSeries]
				}
				resp.Status = tempopb.PartialStatus_PARTIAL
				resp.Message = maxSeriesReachedErrorMsg
			}
			attachExemplars(req, resp)
			resp.Metrics = metricsCombiner.Metrics

			return resp, nil
		},
		quit: func(_ *tempopb.QueryRangeResponse) bool {
			// if max series have been reached only quit if we've also completed at least one shard. this means that we received
			// both actual jobs results and metadata from the sharder. if we have received only job results and no metadata then the combiner
			// will return an empty response
			return combiner.MaxSeriesReached() && completionTracker.CompletedThroughSeconds() != shardtracker.TimestampUnknown
		},
		segment: segmentQueryRangeResponseToMaxPacketSize,
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

// segmentQueryRangeResponseToMaxPacketSize splits resp into one or more QueryRangeResponse values, each within
// maxSize bytes (by proto Size()), accounting for the response object itself. Metrics are included
// in every segment; Status and Message are set on the last segment.
func segmentQueryRangeResponseToMaxPacketSize(resp *tempopb.QueryRangeResponse, maxSize int) []*tempopb.QueryRangeResponse {
	// If not configured return as-is.
	if maxSize <= 0 {
		return []*tempopb.QueryRangeResponse{resp}
	}

	var out []*tempopb.QueryRangeResponse
	var current *tempopb.QueryRangeResponse
	var currentSz int

	startNextPacket := func() {
		current = &tempopb.QueryRangeResponse{
			Metrics: resp.Metrics,
		}
		currentSz = current.Size()
		out = append(out, current)
	}

	startNextPacket()

	for _, s := range resp.Series {
		seriesSz := protoSizeMath(s)

		// Start a new packet if there isn't room for this entry,
		// unless it's the first one, that way we always try to fit at least one.
		if len(current.Series) > 0 && currentSz+seriesSz > maxSize {
			startNextPacket()
		}

		current.Series = append(current.Series, s)
		currentSz += seriesSz
	}

	// Attach real status and message only to the last packet
	if len(out) > 0 {
		out[len(out)-1].Status = resp.Status
		out[len(out)-1].Message = resp.Message
	}

	return out
}

// trimSeriesToCompletedWindow filters series samples and exemplars to only include
// data points between lastCompletedThroughSeconds (exclusive) and completedThroughSeconds (inclusive).
// This is used during streaming to return only new data that has been completed since the last diff.
func trimSeriesToCompletedWindow(series []*tempopb.TimeSeries, lastCompletedThroughSeconds, completedThroughSeconds uint32) {
	lastCompletedThroughMs := int64(lastCompletedThroughSeconds) * 1000
	completedThroughMs := int64(completedThroughSeconds) * 1000

	for _, s := range series {
		// Filter samples to the completed window
		// Find first sample > lastCompletedThrough (skip already sent data)
		startIdx := 0
		for startIdx < len(s.Samples) && s.Samples[startIdx].TimestampMs <= completedThroughMs {
			startIdx++
		}
		// Find first sample > completedThrough (keep only newly completed data)
		endIdx := startIdx
		for endIdx < len(s.Samples) && s.Samples[endIdx].TimestampMs <= lastCompletedThroughMs {
			endIdx++
		}
		s.Samples = s.Samples[startIdx:endIdx]

		// Filter exemplars to the completed window
		startIdx = 0
		for startIdx < len(s.Exemplars) && s.Exemplars[startIdx].TimestampMs <= completedThroughMs {
			startIdx++
		}
		endIdx = startIdx
		for endIdx < len(s.Exemplars) && s.Exemplars[endIdx].TimestampMs <= lastCompletedThroughMs {
			endIdx++
		}
		s.Exemplars = s.Exemplars[startIdx:endIdx]
	}
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
