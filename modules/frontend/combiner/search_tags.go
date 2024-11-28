package combiner

import (
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/tempopb"
	"go.uber.org/atomic"
)

var (
	_ GRPCCombiner[*tempopb.SearchTagsResponse]   = (*genericCombiner[*tempopb.SearchTagsResponse])(nil)
	_ GRPCCombiner[*tempopb.SearchTagsV2Response] = (*genericCombiner[*tempopb.SearchTagsV2Response])(nil)
)

func NewSearchTags(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32) Combiner {
	d := collector.NewDistinctStringWithDiff(maxDataBytes, maxTagsPerScope, staleValueThreshold)
	inspectedBytes := atomic.NewUint64(0)

	c := &genericCombiner[*tempopb.SearchTagsResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsResponse { return &tempopb.SearchTagsResponse{} },
		current:        &tempopb.SearchTagsResponse{TagNames: make([]string, 0)},
		combine: func(partial, _ *tempopb.SearchTagsResponse, _ PipelineResponse) error {
			for _, v := range partial.TagNames {
				d.Collect(v)
			}
			if partial.Metrics != nil {
				inspectedBytes.Add(partial.Metrics.InspectedBytes)
			}
			return nil
		},
		finalize: func(response *tempopb.SearchTagsResponse) (*tempopb.SearchTagsResponse, error) {
			response.TagNames = d.Strings()
			// return metrics with final results
			// TODO: merge with other metrics as well, when we have them, return only InspectedBytes for now
			response.Metrics = &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes.Load()}
			return response, nil
		},
		quit: func(_ *tempopb.SearchTagsResponse) bool {
			return d.Exceeded()
		},
		diff: func(response *tempopb.SearchTagsResponse) (*tempopb.SearchTagsResponse, error) {
			resp, err := d.Diff()
			if err != nil {
				return nil, err
			}

			response.TagNames = resp
			// TODO: merge with other metrics as well, when we have them, return only InspectedBytes for now
			// return metrics with diff results
			response.Metrics = &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes.Load()}
			return response, nil
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTags(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32) GRPCCombiner[*tempopb.SearchTagsResponse] {
	return NewSearchTags(maxDataBytes, maxTagsPerScope, staleValueThreshold).(GRPCCombiner[*tempopb.SearchTagsResponse])
}

func NewSearchTagsV2(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32) Combiner {
	// Distinct collector map to collect scopes and scope values
	distinctValues := collector.NewScopedDistinctStringWithDiff(maxDataBytes, maxTagsPerScope, staleValueThreshold)
	inspectedBytes := atomic.NewUint64(0)

	c := &genericCombiner[*tempopb.SearchTagsV2Response]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsV2Response { return &tempopb.SearchTagsV2Response{} },
		current:        &tempopb.SearchTagsV2Response{Scopes: make([]*tempopb.SearchTagsV2Scope, 0)},
		combine: func(partial, _ *tempopb.SearchTagsV2Response, _ PipelineResponse) error {
			for _, res := range partial.GetScopes() {
				for _, tag := range res.Tags {
					distinctValues.Collect(res.Name, tag)
				}
			}
			if partial.Metrics != nil {
				inspectedBytes.Add(partial.Metrics.InspectedBytes)
			}
			return nil
		},
		finalize: func(final *tempopb.SearchTagsV2Response) (*tempopb.SearchTagsV2Response, error) {
			collected := distinctValues.Strings()
			final.Scopes = make([]*tempopb.SearchTagsV2Scope, 0, len(collected))

			for scope, vals := range collected {
				final.Scopes = append(final.Scopes, &tempopb.SearchTagsV2Scope{
					Name: scope,
					Tags: vals,
				})
			}
			// return metrics with final results
			// TODO: merge with other metrics as well, when we have them, return only InspectedBytes for now
			final.Metrics = &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes.Load()}
			return final, nil
		},
		quit: func(_ *tempopb.SearchTagsV2Response) bool {
			return distinctValues.Exceeded()
		},
		diff: func(response *tempopb.SearchTagsV2Response) (*tempopb.SearchTagsV2Response, error) {
			collected, err := distinctValues.Diff()
			if err != nil {
				return nil, err
			}
			response.Scopes = make([]*tempopb.SearchTagsV2Scope, 0, len(collected))

			for scope, vals := range collected {
				response.Scopes = append(response.Scopes, &tempopb.SearchTagsV2Scope{
					Name: scope,
					Tags: vals,
				})
			}
			// TODO: merge with other metrics as well, when we have them, return only InspectedBytes for now
			// also return metrics with diff results
			response.Metrics = &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes.Load()}
			return response, nil
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTagsV2(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32) GRPCCombiner[*tempopb.SearchTagsV2Response] {
	return NewSearchTagsV2(maxDataBytes, maxTagsPerScope, staleValueThreshold).(GRPCCombiner[*tempopb.SearchTagsV2Response])
}
