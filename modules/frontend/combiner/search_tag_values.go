package combiner

import (
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/tempopb"
	"go.uber.org/atomic"
)

var (
	_ GRPCCombiner[*tempopb.SearchTagValuesResponse]   = (*genericCombiner[*tempopb.SearchTagValuesResponse])(nil)
	_ GRPCCombiner[*tempopb.SearchTagValuesV2Response] = (*genericCombiner[*tempopb.SearchTagValuesV2Response])(nil)
)

func NewSearchTagValues(limitBytes int) Combiner {
	// Distinct collector with no limit
	d := collector.NewDistinctStringWithDiff(limitBytes)
	inspectedBytes := atomic.NewUint64(0)

	c := &genericCombiner[*tempopb.SearchTagValuesResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagValuesResponse { return &tempopb.SearchTagValuesResponse{} },
		current:        &tempopb.SearchTagValuesResponse{TagValues: make([]string, 0)},
		combine: func(partial, _ *tempopb.SearchTagValuesResponse, _ PipelineResponse) error {
			for _, v := range partial.TagValues {
				d.Collect(v)
			}
			if partial.Metrics != nil {
				inspectedBytes.Add(partial.Metrics.InspectedBytes)
			}
			return nil
		},
		finalize: func(final *tempopb.SearchTagValuesResponse) (*tempopb.SearchTagValuesResponse, error) {
			final.TagValues = d.Strings()
			// return metrics in final response
			// TODO: merge with other metrics as well, when we have them, return only InspectedBytes for now
			final.Metrics = &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes.Load()}
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
			// also return latest metrics along with diff
			// TODO: merge with other metrics as well, when we have them, return only InspectedBytes for now
			response.Metrics = &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes.Load()}
			return response, nil
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTagValues(limitBytes int) GRPCCombiner[*tempopb.SearchTagValuesResponse] {
	return NewSearchTagValues(limitBytes).(GRPCCombiner[*tempopb.SearchTagValuesResponse])
}

func NewSearchTagValuesV2(limitBytes int) Combiner {
	// Distinct collector with no limit and diff enabled
	d := collector.NewDistinctValueWithDiff(limitBytes, func(tv tempopb.TagValue) int { return len(tv.Type) + len(tv.Value) })
	inspectedBytes := atomic.NewUint64(0)

	c := &genericCombiner[*tempopb.SearchTagValuesV2Response]{
		httpStatusCode: 200,
		current:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{}},
		new:            func() *tempopb.SearchTagValuesV2Response { return &tempopb.SearchTagValuesV2Response{} },
		combine: func(partial, _ *tempopb.SearchTagValuesV2Response, _ PipelineResponse) error {
			for _, v := range partial.TagValues {
				d.Collect(*v)
			}
			if partial.Metrics != nil {
				inspectedBytes.Add(partial.Metrics.InspectedBytes)
			}
			return nil
		},
		finalize: func(final *tempopb.SearchTagValuesV2Response) (*tempopb.SearchTagValuesV2Response, error) {
			values := d.Values()
			final.TagValues = make([]*tempopb.TagValue, 0, len(values))
			for _, v := range values {
				v2 := v
				final.TagValues = append(final.TagValues, &v2)
			}
			// load Inspected Bytes here and return along with final response
			// TODO: merge with other metrics as well, when we have them, return only InspectedBytes for now
			final.Metrics = &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes.Load()}
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
			// also return metrics along with diffs
			// TODO: merge with other metrics as well, when we have them, return only InspectedBytes for now
			response.Metrics = &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes.Load()}
			return response, nil
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTagValuesV2(limitBytes int) GRPCCombiner[*tempopb.SearchTagValuesV2Response] {
	return NewSearchTagValuesV2(limitBytes).(GRPCCombiner[*tempopb.SearchTagValuesV2Response])
}
