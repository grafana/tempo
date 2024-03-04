package combiner

import (
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

var (
	_ GRPCCombiner[*tempopb.SearchTagValuesResponse]   = (*genericCombiner[*tempopb.SearchTagValuesResponse])(nil)
	_ GRPCCombiner[*tempopb.SearchTagValuesV2Response] = (*genericCombiner[*tempopb.SearchTagValuesV2Response])(nil)
)

func NewSearchTagValues() Combiner {
	// Distinct collector with no limit
	d := util.NewDistinctValueCollector(0, func(_ string) int { return 0 })

	return &genericCombiner[*tempopb.SearchTagValuesResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchTagValuesResponse { return &tempopb.SearchTagValuesResponse{} },
		current:        &tempopb.SearchTagValuesResponse{TagValues: make([]string, 0)},
		combine: func(partial, final *tempopb.SearchTagValuesResponse) error {
			for _, v := range partial.TagValues {
				d.Collect(v)
			}
			return nil
		},
		finalize: func(final *tempopb.SearchTagValuesResponse) (*tempopb.SearchTagValuesResponse, error) {
			final.TagValues = d.Values()
			return final, nil
		},
	}
}

func NewTypedSearchTagValues() GRPCCombiner[*tempopb.SearchTagValuesResponse] {
	return NewSearchTagValues().(GRPCCombiner[*tempopb.SearchTagValuesResponse])
}

func NewSearchTagValuesV2() Combiner {
	// Distinct collector with no limit
	d := util.NewDistinctValueCollector(0, func(_ tempopb.TagValue) int { return 0 })

	return &genericCombiner[*tempopb.SearchTagValuesV2Response]{
		current: &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{}},
		new:     func() *tempopb.SearchTagValuesV2Response { return &tempopb.SearchTagValuesV2Response{} },
		combine: func(partial, final *tempopb.SearchTagValuesV2Response) error {
			for _, v := range partial.TagValues {
				d.Collect(*v)
			}
			return nil
		},
		finalize: func(final *tempopb.SearchTagValuesV2Response) (*tempopb.SearchTagValuesV2Response, error) {
			final.TagValues = make([]*tempopb.TagValue, 0, len(d.Values()))
			values := d.Values()
			for _, v := range values {
				v2 := v
				final.TagValues = append(final.TagValues, &v2)
			}
			return final, nil
		},
	}
}

func NewTypedSearchTagValuesV2() GRPCCombiner[*tempopb.SearchTagValuesV2Response] {
	return NewSearchTagValuesV2().(GRPCCombiner[*tempopb.SearchTagValuesV2Response])
}
