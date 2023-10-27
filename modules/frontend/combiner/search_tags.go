package combiner

import (
	"fmt"
	"io"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

var _ Combiner = (*genericCombiner[*tempopb.SearchTagsResponse])(nil)

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
