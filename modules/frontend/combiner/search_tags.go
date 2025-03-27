package combiner

import (
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	_ GRPCCombiner[*tempopb.SearchTagsResponse]   = (*genericCombiner[*tempopb.SearchTagsResponse])(nil)
	_ GRPCCombiner[*tempopb.SearchTagsV2Response] = (*genericCombiner[*tempopb.SearchTagsV2Response])(nil)
)

func NewSearchTags(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32) Combiner {
	d := collector.NewDistinctStringWithDiff(maxDataBytes, maxTagsPerScope, staleValueThreshold)
	metricsCollector := collector.NewSimpleMetricsCollector()

	c := &genericCombiner[*tempopb.SearchTagsResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsResponse { return &tempopb.SearchTagsResponse{} },
		current:        &tempopb.SearchTagsResponse{TagNames: make([]string, 0)},
		combine: func(partial, _ *tempopb.SearchTagsResponse, _ PipelineResponse) error {
			if partial.Metrics != nil {
				metricsCollector.Add(partial.Metrics.InspectedBytes)
			}

			for _, v := range partial.TagNames {
				d.Collect(v)
			}
			return nil
		},
		finalize: func(final *tempopb.SearchTagsResponse) (*tempopb.SearchTagsResponse, error) {
			final.TagNames = d.Strings()

			// Attach metrics using the collector
			if final.Metrics == nil {
				final.Metrics = &tempopb.MetadataMetrics{}
			}
			final.Metrics.InspectedBytes = metricsCollector.TotalValue()

			return final, nil
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
			// Apply metrics from the collector to the diff response
			if response.Metrics == nil {
				response.Metrics = &tempopb.MetadataMetrics{}
			}
			response.Metrics.InspectedBytes = metricsCollector.TotalValue()
			return response, nil
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewSearchTagsV2(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32) Combiner {
	// Distinct collector map to collect scopes and scope values
	distinctValues := collector.NewScopedDistinctStringWithDiff(maxDataBytes, maxTagsPerScope, staleValueThreshold)
	metricsCollector := collector.NewSimpleMetricsCollector()

	c := &genericCombiner[*tempopb.SearchTagsV2Response]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsV2Response { return &tempopb.SearchTagsV2Response{} },
		current:        &tempopb.SearchTagsV2Response{Scopes: make([]*tempopb.SearchTagsV2Scope, 0)},
		combine: func(partial, _ *tempopb.SearchTagsV2Response, _ PipelineResponse) error {
			if partial.Metrics != nil {
				metricsCollector.Add(partial.Metrics.InspectedBytes)
			}

			for _, res := range partial.GetScopes() {
				for _, tag := range res.Tags {
					distinctValues.Collect(res.Name, tag)
				}
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

			// Attach metrics using the collector
			if final.Metrics == nil {
				final.Metrics = &tempopb.MetadataMetrics{}
			}
			final.Metrics.InspectedBytes = metricsCollector.TotalValue()

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

			// Apply metrics from the collector to the diff response
			if response.Metrics == nil {
				response.Metrics = &tempopb.MetadataMetrics{}
			}
			response.Metrics.InspectedBytes = metricsCollector.TotalValue()
			return response, nil
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func NewTypedSearchTags(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32) GRPCCombiner[*tempopb.SearchTagsResponse] {
	return NewSearchTags(maxDataBytes, maxTagsPerScope, staleValueThreshold).(GRPCCombiner[*tempopb.SearchTagsResponse])
}

func NewTypedSearchTagsV2(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32) GRPCCombiner[*tempopb.SearchTagsV2Response] {
	return NewSearchTagsV2(maxDataBytes, maxTagsPerScope, staleValueThreshold).(GRPCCombiner[*tempopb.SearchTagsV2Response])
}
