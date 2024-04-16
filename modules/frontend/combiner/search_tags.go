package combiner

import (
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

var (
	_ GRPCCombiner[*tempopb.SearchTagsResponse]   = (*genericCombiner[*tempopb.SearchTagsResponse])(nil)
	_ GRPCCombiner[*tempopb.SearchTagsV2Response] = (*genericCombiner[*tempopb.SearchTagsV2Response])(nil)
)

func NewSearchTags(limitBytes int) Combiner {
	d := util.NewDistinctStringCollector(limitBytes)

	return &genericCombiner[*tempopb.SearchTagsResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsResponse { return &tempopb.SearchTagsResponse{} },
		current:        &tempopb.SearchTagsResponse{TagNames: make([]string, 0)},
		combine: func(partial, final *tempopb.SearchTagsResponse, _ PipelineResponse) error {
			for _, v := range partial.TagNames {
				d.Collect(v)
			}
			return nil
		},
		finalize: func(response *tempopb.SearchTagsResponse) (*tempopb.SearchTagsResponse, error) {
			response.TagNames = d.Strings()
			return response, nil
		},
		quit: func(_ *tempopb.SearchTagsResponse) bool {
			return d.Exceeded()
		},
		diff: func(response *tempopb.SearchTagsResponse) (*tempopb.SearchTagsResponse, error) {
			response.TagNames = d.Diff()
			return response, nil
		},
	}
}

func NewTypedSearchTags(limitBytes int) GRPCCombiner[*tempopb.SearchTagsResponse] {
	return NewSearchTags(limitBytes).(GRPCCombiner[*tempopb.SearchTagsResponse])
}

func NewSearchTagsV2(limitBytes int) Combiner {
	// Distinct collector map to collect scopes and scope values
	distinctValues := map[string]*util.DistinctStringCollector{}

	return &genericCombiner[*tempopb.SearchTagsV2Response]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagsV2Response { return &tempopb.SearchTagsV2Response{} },
		current:        &tempopb.SearchTagsV2Response{Scopes: make([]*tempopb.SearchTagsV2Scope, 0)},
		combine: func(partial, final *tempopb.SearchTagsV2Response, _ PipelineResponse) error {
			for _, res := range partial.GetScopes() {
				dvc := distinctValues[res.Name]
				if dvc == nil {
					dvc = util.NewDistinctStringCollector(limitBytes)
					distinctValues[res.Name] = dvc
				}
				for _, tag := range res.Tags {
					dvc.Collect(tag)
				}
			}
			return nil
		},
		finalize: func(final *tempopb.SearchTagsV2Response) (*tempopb.SearchTagsV2Response, error) {
			final.Scopes = make([]*tempopb.SearchTagsV2Scope, 0, len(distinctValues))

			for scope, dvc := range distinctValues {
				final.Scopes = append(final.Scopes, &tempopb.SearchTagsV2Scope{
					Name: scope,
					Tags: dvc.Strings(),
				})
			}
			return final, nil
		},
		quit: func(_ *tempopb.SearchTagsV2Response) bool {
			for _, dvc := range distinctValues {
				if dvc.Exceeded() {
					return true
				}
			}
			return false
		},
		diff: func(response *tempopb.SearchTagsV2Response) (*tempopb.SearchTagsV2Response, error) {
			response.Scopes = make([]*tempopb.SearchTagsV2Scope, 0, len(distinctValues))

			for scope, dvc := range distinctValues {
				diff := dvc.Diff()
				if len(diff) == 0 {
					continue
				}

				response.Scopes = append(response.Scopes, &tempopb.SearchTagsV2Scope{
					Name: scope,
					Tags: diff,
				})
			}

			return response, nil
		},
	}
}

func NewTypedSearchTagsV2(limitBytes int) GRPCCombiner[*tempopb.SearchTagsV2Response] {
	return NewSearchTagsV2(limitBytes).(GRPCCombiner[*tempopb.SearchTagsV2Response])
}
