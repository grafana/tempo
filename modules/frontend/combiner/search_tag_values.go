package combiner

import (
	"fmt"
	"io"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

var (
	_ Combiner = (*genericCombiner[*tempopb.SearchTagValuesResponse])(nil)
	_ Combiner = (*genericCombiner[*tempopb.SearchTagValuesV2Response])(nil)
)

func NewSearchTagValues() Combiner {
	// Distinct collector with no limit
	d := util.NewDistinctValueCollector(0, func(_ string) int { return 0 })

	return &genericCombiner[*tempopb.SearchTagValuesResponse]{
		code:  200,
		final: &tempopb.SearchTagValuesResponse{TagValues: make([]string, 0)},
		combine: func(body io.ReadCloser, final *tempopb.SearchTagValuesResponse) error {
			response := &tempopb.SearchTagValuesResponse{}
			if err := jsonpb.Unmarshal(body, response); err != nil {
				return fmt.Errorf("error unmarshalling response body: %w", err)
			}
			for _, v := range response.TagValues {
				d.Collect(v)
			}
			return nil
		},
		result: func(response *tempopb.SearchTagValuesResponse) (string, error) {
			response.TagValues = d.Values()
			return new(jsonpb.Marshaler).MarshalToString(response)
		},
	}
}

func NewSearchTagValuesV2() Combiner {
	// Distinct collector with no limit
	d := util.NewDistinctValueCollector(0, func(_ tempopb.TagValue) int { return 0 })

	return &genericCombiner[*tempopb.SearchTagValuesV2Response]{
		final: &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{}},
		combine: func(body io.ReadCloser, final *tempopb.SearchTagValuesV2Response) error {
			response := &tempopb.SearchTagValuesV2Response{}
			if err := jsonpb.Unmarshal(body, response); err != nil {
				return fmt.Errorf("error unmarshalling response body: %w", err)
			}
			for _, v := range response.TagValues {
				d.Collect(*v)
			}
			return nil
		},
		result: func(response *tempopb.SearchTagValuesV2Response) (string, error) {
			values := d.Values()
			for _, v := range values {
				v2 := v
				response.TagValues = append(response.TagValues, &v2)
			}
			return new(jsonpb.Marshaler).MarshalToString(response)
		},
	}
}
