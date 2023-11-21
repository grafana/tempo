package combiner

import (
	"fmt"
	"io"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

var (
	_ Combiner = (*genericCombiner[*tempopb.SearchTagsResponse])(nil)
	_ Combiner = (*genericCombiner[*tempopb.SearchTagsV2Response])(nil)
)

func NewSearchTags() Combiner {
	// Distinct collector with no limit
	d := util.NewDistinctValueCollector(0, func(_ string) int { return 0 })

	return &genericCombiner[*tempopb.SearchTagsResponse]{
		code:  200,
		final: &tempopb.SearchTagsResponse{TagNames: make([]string, 0)},
		combine: func(body io.ReadCloser, final *tempopb.SearchTagsResponse) error {
			response := &tempopb.SearchTagsResponse{}
			if err := jsonpb.Unmarshal(body, response); err != nil {
				return fmt.Errorf("error unmarshalling response body: %w", err)
			}
			for _, v := range response.TagNames {
				d.Collect(v)
			}
			return nil
		},
		result: func(response *tempopb.SearchTagsResponse) (string, error) {
			response.TagNames = d.Values()
			return new(jsonpb.Marshaler).MarshalToString(response)
		},
	}
}

func NewSearchTagsV2() Combiner {
	// Distinct collector map to collect scopes and scope values
	distinctValues := map[string]*util.DistinctValueCollector[string]{}

	return &genericCombiner[*tempopb.SearchTagsV2Response]{
		code:  200,
		final: &tempopb.SearchTagsV2Response{Scopes: make([]*tempopb.SearchTagsV2Scope, 0)},
		combine: func(body io.ReadCloser, final *tempopb.SearchTagsV2Response) error {
			response := &tempopb.SearchTagsV2Response{}
			if err := jsonpb.Unmarshal(body, response); err != nil {
				return fmt.Errorf("error unmarshalling response body: %w", err)
			}
			for _, res := range response.GetScopes() {
				dvc := distinctValues[res.Name]
				if dvc == nil {
					// no limit collector to collect scope values
					dvc = util.NewDistinctValueCollector(0, func(_ string) int { return 0 })
					distinctValues[res.Name] = dvc
				}
				for _, tag := range res.Tags {
					dvc.Collect(tag)
				}
			}
			return nil
		},
		result: func(response *tempopb.SearchTagsV2Response) (string, error) {
			for scope, dvc := range distinctValues {
				response.Scopes = append(response.Scopes, &tempopb.SearchTagsV2Scope{
					Name: scope,
					Tags: dvc.Values(),
				})
			}
			return new(jsonpb.Marshaler).MarshalToString(response)
		},
	}
}
