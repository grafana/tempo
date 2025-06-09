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

func NewSearchTagValues(maxDataBytes int, maxTagsValues uint32, staleValueThreshold uint32) Combiner {
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
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTagValues(maxDataBytes int, maxTagsValues uint32, staleValueThreshold uint32) GRPCCombiner[*tempopb.SearchTagValuesResponse] {
	return NewSearchTagValues(maxDataBytes, maxTagsValues, staleValueThreshold).(GRPCCombiner[*tempopb.SearchTagValuesResponse])
}

func NewSearchTagValuesV2(maxDataBytes int, maxTagsValues uint32, staleValueThreshold uint32) Combiner {
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
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTagValuesV2(maxDataBytes int, maxTagsValues uint32, staleValueThreshold uint32) GRPCCombiner[*tempopb.SearchTagValuesV2Response] {
	return NewSearchTagValuesV2(maxDataBytes, maxTagsValues, staleValueThreshold).(GRPCCombiner[*tempopb.SearchTagValuesV2Response])
}
