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

func NewSearchTags(limitBytes int) Combiner {
	d := collector.NewDistinctStringWithDiff(limitBytes)
	// TODO: can we just use a regular int? do we need atomic int here??
	inspectedBytes := atomic.NewUint64(0)

	c := &genericCombiner[*tempopb.SearchTagsResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsResponse { return &tempopb.SearchTagsResponse{} },
		current:        &tempopb.SearchTagsResponse{TagNames: make([]string, 0)},
		combine: func(partial, _ *tempopb.SearchTagsResponse, _ PipelineResponse) error {
			for _, v := range partial.TagNames {
				d.Collect(v)
			}
			inspectedBytes.Add(partial.Metrics.InspectedBytes)
			return nil
		},
		finalize: func(response *tempopb.SearchTagsResponse) (*tempopb.SearchTagsResponse, error) {
			response.TagNames = d.Strings()
			// return metrics with final results
			response.Metrics.InspectedBytes = inspectedBytes.Load()
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
			// also return metrics with diff results
			response.Metrics.InspectedBytes = inspectedBytes.Load()
			return response, nil
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTags(limitBytes int) GRPCCombiner[*tempopb.SearchTagsResponse] {
	return NewSearchTags(limitBytes).(GRPCCombiner[*tempopb.SearchTagsResponse])
}

func NewSearchTagsV2(limitBytes int) Combiner {
	// Distinct collector map to collect scopes and scope values
	distinctValues := collector.NewScopedDistinctStringWithDiff(limitBytes)
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
			inspectedBytes.Add(partial.Metrics.InspectedBytes)
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
			final.Metrics.InspectedBytes = inspectedBytes.Load()
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
			// also return metrics with diff results
			response.Metrics.InspectedBytes = inspectedBytes.Load()
			return response, nil
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTagsV2(limitBytes int) GRPCCombiner[*tempopb.SearchTagsV2Response] {
	return NewSearchTagsV2(limitBytes).(GRPCCombiner[*tempopb.SearchTagsV2Response])
}
