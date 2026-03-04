package combiner

import (
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	_ GRPCCombiner[*tempopb.SearchTagValuesResponse]   = (*genericCombiner[*tempopb.SearchTagValuesResponse])(nil)
	_ GRPCCombiner[*tempopb.SearchTagValuesV2Response] = (*genericCombiner[*tempopb.SearchTagValuesV2Response])(nil)
)

func NewSearchTagValues(maxDataBytes int, maxTagsValues uint32, staleValueThreshold uint32, marshalingFormat api.MarshallingFormat) Combiner {
	// Distinct collector with no limit
	d := collector.NewDistinctStringWithDiff(maxDataBytes, maxTagsValues, staleValueThreshold)
	metricsCombiner := NewMetadataMetricsCombiner()
	c := &genericCombiner[*tempopb.SearchTagValuesResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagValuesResponse { return &tempopb.SearchTagValuesResponse{} },
		current:        &tempopb.SearchTagValuesResponse{TagValues: make([]string, 0)},
		combine: func(partial, _ *tempopb.SearchTagValuesResponse, pipelineResp PipelineResponse) error {
			for _, v := range partial.TagValues {
				d.Collect(v)
			}
			metricsCombiner.Combine(partial.Metrics, pipelineResp)
			return nil
		},
		finalize: func(final *tempopb.SearchTagValuesResponse) (*tempopb.SearchTagValuesResponse, error) {
			final.TagValues = d.Strings()
			final.Metrics = metricsCombiner.Metrics
			return final, nil
		},
		quit: func(_ *tempopb.SearchTagValuesResponse) bool {
			return d.Exceeded()
		},
		diff: func(response *tempopb.SearchTagValuesResponse) (*tempopb.SearchTagValuesResponse, error) {
			resp, err := d.Diff()
			if err != nil {
				return nil, err
			}
			response.TagValues = resp
			response.Metrics = metricsCombiner.Metrics

			return response, nil
		},
		segment: segmentSearchTagValuesResponse,
	}
	initHTTPCombiner(c, marshalingFormat)
	return c
}

func NewTypedSearchTagValues(maxDataBytes int, maxTagsValues uint32, staleValueThreshold uint32, marshalingFormat api.MarshallingFormat) GRPCCombiner[*tempopb.SearchTagValuesResponse] {
	return NewSearchTagValues(maxDataBytes, maxTagsValues, staleValueThreshold, marshalingFormat).(GRPCCombiner[*tempopb.SearchTagValuesResponse])
}

func NewSearchTagValuesV2(maxDataBytes int, maxTagsValues uint32, staleValueThreshold uint32, marshalingFormat api.MarshallingFormat) Combiner {
	// Distinct collector with no limit and diff enabled
	d := collector.NewDistinctValueWithDiff(maxDataBytes, maxTagsValues, staleValueThreshold, func(tv tempopb.TagValue) int { return len(tv.Type) + len(tv.Value) })
	metricsCombiner := NewMetadataMetricsCombiner()

	c := &genericCombiner[*tempopb.SearchTagValuesV2Response]{
		httpStatusCode: 200,
		current:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{}},
		new:            func() *tempopb.SearchTagValuesV2Response { return &tempopb.SearchTagValuesV2Response{} },
		combine: func(partial, _ *tempopb.SearchTagValuesV2Response, pipelineResp PipelineResponse) error {
			for _, v := range partial.TagValues {
				d.Collect(*v)
			}
			metricsCombiner.Combine(partial.Metrics, pipelineResp)
			return nil
		},
		finalize: func(final *tempopb.SearchTagValuesV2Response) (*tempopb.SearchTagValuesV2Response, error) {
			values := d.Values()
			final.TagValues = make([]*tempopb.TagValue, 0, len(values))
			for _, v := range values {
				v2 := v
				final.TagValues = append(final.TagValues, &v2)
			}

			final.Metrics = metricsCombiner.Metrics

			return final, nil
		},
		quit: func(_ *tempopb.SearchTagValuesV2Response) bool {
			return d.Exceeded()
		},
		diff: func(response *tempopb.SearchTagValuesV2Response) (*tempopb.SearchTagValuesV2Response, error) {
			diff, err := d.Diff()
			if err != nil {
				return nil, err
			}
			response.TagValues = make([]*tempopb.TagValue, 0, len(diff))
			for _, v := range diff {
				v2 := v
				response.TagValues = append(response.TagValues, &v2)
			}

			response.Metrics = metricsCombiner.Metrics

			return response, nil
		},
		segment: segmentSearchTagValuesV2Response,
	}
	initHTTPCombiner(c, marshalingFormat)
	return c
}

func NewTypedSearchTagValuesV2(maxDataBytes int, maxTagsValues uint32, staleValueThreshold uint32, marshalingFormat api.MarshallingFormat) GRPCCombiner[*tempopb.SearchTagValuesV2Response] {
	return NewSearchTagValuesV2(maxDataBytes, maxTagsValues, staleValueThreshold, marshalingFormat).(GRPCCombiner[*tempopb.SearchTagValuesV2Response])
}

// segmentSearchTagValuesResponse splits response into one or more SearchTagValuesResponse values, each within
// maxSize bytes (by proto Size()). Metrics are included in every segment.
func segmentSearchTagValuesResponse(response *tempopb.SearchTagValuesResponse, maxSize int) []*tempopb.SearchTagValuesResponse {
	if maxSize <= 0 {
		return []*tempopb.SearchTagValuesResponse{response}
	}

	var out []*tempopb.SearchTagValuesResponse
	var current *tempopb.SearchTagValuesResponse
	var currentSz int

	startNextPacket := func() {
		current = &tempopb.SearchTagValuesResponse{
			TagValues: nil,
			Metrics:   response.Metrics,
		}
		currentSz = current.Size()
		out = append(out, current)
	}

	startNextPacket()

	for _, v := range response.TagValues {
		itemSz := protoStringSize(v)
		// Start a new packet if there isn't room for this entry,
		// unless it's the first one, that way we always try to fit at least one.
		if len(current.TagValues) > 0 && currentSz+itemSz > maxSize {
			startNextPacket()
		}
		current.TagValues = append(current.TagValues, v)
		currentSz += itemSz
	}

	return out
}

// segmentSearchTagValuesV2Response splits response into one or more SearchTagValuesV2Response values, each within
// maxSize bytes (by proto Size()). Metrics are included in every segment.
func segmentSearchTagValuesV2Response(response *tempopb.SearchTagValuesV2Response, maxSize int) []*tempopb.SearchTagValuesV2Response {
	if maxSize <= 0 {
		return []*tempopb.SearchTagValuesV2Response{response}
	}

	var out []*tempopb.SearchTagValuesV2Response
	var current *tempopb.SearchTagValuesV2Response
	var currentSz int

	startNextPacket := func() {
		current = &tempopb.SearchTagValuesV2Response{
			TagValues: nil,
			Metrics:   response.Metrics,
		}
		currentSz = current.Size()
		out = append(out, current)
	}

	startNextPacket()

	for _, tv := range response.TagValues {
		tvSz := tv.Size()
		// Start a new packet if there isn't room for this entry,
		// unless it's the first one, that way we always try to fit at least one.
		if len(current.TagValues) > 0 && currentSz+tvSz > maxSize {
			startNextPacket()
		}
		current.TagValues = append(current.TagValues, tv)
		currentSz += tvSz
	}

	return out
}
