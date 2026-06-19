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

func NewSearchTags(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32, marshalingFormat api.MarshallingFormat) Combiner {
	d := collector.NewDistinctStringWithDiff(maxDataBytes, maxTagsPerScope, staleValueThreshold)
	metricsCombiner := NewMetadataMetricsCombiner()

	c := &genericCombiner[*tempopb.SearchTagsResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsResponse { return &tempopb.SearchTagsResponse{} },
		current:        &tempopb.SearchTagsResponse{TagNames: make([]string, 0)},
		combine: func(partial, _ *tempopb.SearchTagsResponse, pipelineResp PipelineResponse) error {
			metricsCombiner.Combine(partial.Metrics, pipelineResp)

			for _, v := range partial.TagNames {
				d.Collect(v)
			}
			return nil
		},
		finalize: func(final *tempopb.SearchTagsResponse) (*tempopb.SearchTagsResponse, error) {
			final.TagNames = d.Strings()
			final.Metrics = metricsCombiner.Metrics

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
			response.Metrics = metricsCombiner.Metrics
			return response, nil
		},
		segment: segmentSearchTagsResponse,
	}
	initHTTPCombiner(c, marshalingFormat)
	return c
}

func NewSearchTagsV2(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32, marshalingFormat api.MarshallingFormat) Combiner {
	// Distinct collector map to collect scopes and scope values
	distinctValues := collector.NewScopedDistinctStringWithDiff(maxDataBytes, maxTagsPerScope, staleValueThreshold)
	metricsCombiner := NewMetadataMetricsCombiner()

	c := &genericCombiner[*tempopb.SearchTagsV2Response]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsV2Response { return &tempopb.SearchTagsV2Response{} },
		current:        &tempopb.SearchTagsV2Response{Scopes: make([]*tempopb.SearchTagsV2Scope, 0)},
		combine: func(partial, _ *tempopb.SearchTagsV2Response, pipelineResp PipelineResponse) error {
			metricsCombiner.Combine(partial.Metrics, pipelineResp)

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

			final.Metrics = metricsCombiner.Metrics
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

			response.Metrics = metricsCombiner.Metrics
			return response, nil
		},
		segment: segmentSearchTagsV2Response,
	}
	initHTTPCombiner(c, marshalingFormat)
	return c
}

func NewTypedSearchTags(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32, marshalingFormat api.MarshallingFormat) GRPCCombiner[*tempopb.SearchTagsResponse] {
	return NewSearchTags(maxDataBytes, maxTagsPerScope, staleValueThreshold, marshalingFormat).(GRPCCombiner[*tempopb.SearchTagsResponse])
}

func NewTypedSearchTagsV2(maxDataBytes int, maxTagsPerScope uint32, staleValueThreshold uint32, marshalingFormat api.MarshallingFormat) GRPCCombiner[*tempopb.SearchTagsV2Response] {
	return NewSearchTagsV2(maxDataBytes, maxTagsPerScope, staleValueThreshold, marshalingFormat).(GRPCCombiner[*tempopb.SearchTagsV2Response])
}

// segmentSearchTagsResponse splits response into one or more SearchTagsResponse values, each within
// maxSize bytes (by proto Size()). Metrics are included in every segment.
func segmentSearchTagsResponse(response *tempopb.SearchTagsResponse, maxSize int) []*tempopb.SearchTagsResponse {
	if maxSize <= 0 {
		return []*tempopb.SearchTagsResponse{response}
	}

	var out []*tempopb.SearchTagsResponse
	var current *tempopb.SearchTagsResponse
	var currentSz int

	startNextPacket := func() {
		current = &tempopb.SearchTagsResponse{
			TagNames: nil,
			Metrics:  response.Metrics,
		}
		currentSz = current.Size()
		out = append(out, current)
	}

	startNextPacket()

	for _, name := range response.TagNames {
		sz := protoStringSize(name)
		// Start a new packet if there isn't room for this entry,
		// unless it's the first one, that way we always try to fit at least one.
		if len(current.TagNames) > 0 && currentSz+sz > maxSize {
			startNextPacket()
		}
		current.TagNames = append(current.TagNames, name)
		currentSz += sz
	}

	return out
}

// segmentSearchTagsV2Response splits response into one or more SearchTagsV2Response values, each within
// maxSize bytes (by proto Size()). Metrics are included in every segment. Scopes are split at the tag
// level: when a scope doesn't fit, the same scope name is repeated in the next packet and tags keep
// being added to it.
func segmentSearchTagsV2Response(response *tempopb.SearchTagsV2Response, maxSize int) []*tempopb.SearchTagsV2Response {
	if maxSize <= 0 {
		return []*tempopb.SearchTagsV2Response{response}
	}

	var out []*tempopb.SearchTagsV2Response
	var current *tempopb.SearchTagsV2Response
	var currentSz int

	startNextPacket := func() {
		current = &tempopb.SearchTagsV2Response{
			Scopes:  nil,
			Metrics: response.Metrics,
		}
		currentSz = current.Size()
		out = append(out, current)
	}

	startNextPacket()

	for _, scope := range response.Scopes {
		dest := &tempopb.SearchTagsV2Scope{Name: scope.Name}
		current.Scopes = append(current.Scopes, dest)

		for _, tag := range scope.Tags {
			sz := protoStringSize(tag)

			// Start a new packet if there isn't room for this entry,
			// unless it's the first one, that way we always try to fit at least one.
			if len(dest.Tags) > 0 && currentSz+sz > maxSize {
				startNextPacket()

				// Restart current scope in this new packet.
				dest = &tempopb.SearchTagsV2Scope{Name: scope.Name}
				current.Scopes = append(current.Scopes, dest)
			}

			dest.Tags = append(dest.Tags, tag)
			currentSz += sz
		}
	}

	return out
}
