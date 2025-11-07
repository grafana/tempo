package combiner

import (
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagsCombinerJSON(t *testing.T) {
	testTagsCombiner(t, api.MarshallingFormatJSON)
}

func TestTagsCombinerProtobuf(t *testing.T) {
	testTagsCombiner(t, api.MarshallingFormatProtobuf)
}

func testTagsCombiner(t *testing.T, marshalingFormat api.MarshallingFormat) {
	tests := []struct {
		name               string
		factory            func(int, uint32, uint32, api.MarshallingFormat) Combiner
		limitBytes         int
		maxTagsValues      uint32
		maxCacheHits       uint32
		result1            proto.Message
		result2            proto.Message
		expectedResult     proto.Message
		expectedShouldQuit bool

		actualResult proto.Message       // provides a way for the test runner to unmarshal the response
		sort         func(proto.Message) // the results are based on non-deterministic map iteration, provides a way for the runner to sort the results for comparison
	}{
		{
			name:           "SearchTags",
			factory:        NewSearchTags,
			result1:        &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}},
			result2:        &tempopb.SearchTagsResponse{TagNames: []string{"tag2", "tag3"}},
			expectedResult: &tempopb.SearchTagsResponse{TagNames: []string{"tag1", "tag2", "tag3"}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:   &tempopb.SearchTagsResponse{},
			sort:           func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagsResponse).TagNames) },
			limitBytes:     100,
		},
		{
			name:           "SearchTagsV2",
			factory:        NewSearchTagsV2,
			result1:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1"}}}},
			result2:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v2", "v1"}}}},
			expectedResult: &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1", "v2"}}}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:   &tempopb.SearchTagsV2Response{},
			sort: func(m proto.Message) {
				scopes := m.(*tempopb.SearchTagsV2Response).Scopes
				for _, scope := range scopes {
					sort.Strings(scope.Tags)
				}
				sort.Slice(scopes, func(i, j int) bool {
					return scopes[i].Name < scopes[j].Name
				})
			},
			limitBytes: 100,
		},
		{
			name:           "SearchTagValues",
			factory:        NewSearchTagValues,
			result1:        &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}},
			result2:        &tempopb.SearchTagValuesResponse{TagValues: []string{"tag2", "tag3"}},
			expectedResult: &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1", "tag2", "tag3"}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:   &tempopb.SearchTagValuesResponse{},
			sort:           func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagValuesResponse).TagValues) },
			limitBytes:     100,
		},
		{
			name:           "SearchTagValuesV2",
			factory:        NewSearchTagValuesV2,
			result1:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}},
			result2:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}},
			expectedResult: &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}, {Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:   &tempopb.SearchTagValuesV2Response{},
			sort: func(m proto.Message) {
				sort.Slice(m.(*tempopb.SearchTagValuesV2Response).TagValues, func(i, j int) bool {
					return m.(*tempopb.SearchTagValuesV2Response).TagValues[i].Value < m.(*tempopb.SearchTagValuesV2Response).TagValues[j].Value
				})
			},
			limitBytes: 100,
		},
		// limits
		{
			name:               "SearchTags - limited",
			factory:            NewSearchTags,
			result1:            &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}},
			result2:            &tempopb.SearchTagsResponse{TagNames: []string{"tag2", "tag3"}},
			expectedResult:     &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:       &tempopb.SearchTagsResponse{},
			sort:               func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagsResponse).TagNames) },
			expectedShouldQuit: true,
			limitBytes:         5,
		},
		{
			name:           "SearchTagsV2 - limited",
			factory:        NewSearchTagsV2,
			result1:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1"}}}},
			result2:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v2", "v1"}}}},
			expectedResult: &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1"}}}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:   &tempopb.SearchTagsV2Response{},
			sort: func(m proto.Message) {
				scopes := m.(*tempopb.SearchTagsV2Response).Scopes
				for _, scope := range scopes {
					sort.Strings(scope.Tags)
				}
				sort.Slice(scopes, func(i, j int) bool {
					return scopes[i].Name < scopes[j].Name
				})
			},
			expectedShouldQuit: true,
			limitBytes:         2,
		},
		{
			name:               "SearchTagValues - limited",
			factory:            NewSearchTagValues,
			result1:            &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}},
			result2:            &tempopb.SearchTagValuesResponse{TagValues: []string{"tag2", "tag3"}},
			expectedResult:     &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:       &tempopb.SearchTagValuesResponse{},
			sort:               func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagValuesResponse).TagValues) },
			expectedShouldQuit: true,
			limitBytes:         5,
		},
		{
			name:           "SearchTagValuesV2 - limited",
			factory:        NewSearchTagValuesV2,
			result1:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}},
			result2:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}},
			expectedResult: &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:   &tempopb.SearchTagValuesV2Response{},
			sort: func(m proto.Message) {
				sort.Slice(m.(*tempopb.SearchTagValuesV2Response).TagValues, func(i, j int) bool {
					return m.(*tempopb.SearchTagValuesV2Response).TagValues[i].Value < m.(*tempopb.SearchTagValuesV2Response).TagValues[j].Value
				})
			},
			expectedShouldQuit: true,
			limitBytes:         10,
		},
		{
			name:           "SearchTagValuesV2 - max values limited",
			factory:        NewSearchTagValuesV2,
			result1:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}},
			result2:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}},
			expectedResult: &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{}},
			actualResult:   &tempopb.SearchTagValuesV2Response{},
			sort: func(m proto.Message) {
				sort.Slice(m.(*tempopb.SearchTagValuesV2Response).TagValues, func(i, j int) bool {
					return m.(*tempopb.SearchTagValuesV2Response).TagValues[i].Value < m.(*tempopb.SearchTagValuesV2Response).TagValues[j].Value
				})
			},
			expectedShouldQuit: true,
			maxTagsValues:      1,
		},
		// with metrics
		{
			name:           "SearchTags - metrics",
			factory:        NewSearchTags,
			result1:        &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}},
			result2:        &tempopb.SearchTagsResponse{TagNames: []string{"tag2", "tag3"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}},
			expectedResult: &tempopb.SearchTagsResponse{TagNames: []string{"tag1", "tag2", "tag3"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}},
			actualResult:   &tempopb.SearchTagsResponse{},
			sort:           func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagsResponse).TagNames) },
			limitBytes:     100,
		},
		{
			name:           "SearchTagsV2 - metrics",
			factory:        NewSearchTagsV2,
			result1:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1"}}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}},
			result2:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v2", "v1"}}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}},
			expectedResult: &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1", "v2"}}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}},
			actualResult:   &tempopb.SearchTagsV2Response{},
			sort: func(m proto.Message) {
				scopes := m.(*tempopb.SearchTagsV2Response).Scopes
				for _, scope := range scopes {
					sort.Strings(scope.Tags)
				}
				sort.Slice(scopes, func(i, j int) bool {
					return scopes[i].Name < scopes[j].Name
				})
			},
			limitBytes: 100,
		},
		{
			name:           "SearchTagValues - metrics",
			factory:        NewSearchTagValues,
			result1:        &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}},
			result2:        &tempopb.SearchTagValuesResponse{TagValues: []string{"tag2", "tag3"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}},
			expectedResult: &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1", "tag2", "tag3"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}},
			actualResult:   &tempopb.SearchTagValuesResponse{},
			sort:           func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagValuesResponse).TagValues) },
			limitBytes:     100,
		},
		{
			name:           "SearchTagValuesV2 - metrics",
			factory:        NewSearchTagValuesV2,
			result1:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}},
			result2:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}},
			expectedResult: &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}, {Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}},
			actualResult:   &tempopb.SearchTagValuesV2Response{},
			sort: func(m proto.Message) {
				sort.Slice(m.(*tempopb.SearchTagValuesV2Response).TagValues, func(i, j int) bool {
					return m.(*tempopb.SearchTagValuesV2Response).TagValues[i].Value < m.(*tempopb.SearchTagValuesV2Response).TagValues[j].Value
				})
			},
			limitBytes: 100,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			combiner := tc.factory(tc.limitBytes, tc.maxTagsValues, tc.maxCacheHits, marshalingFormat)

			err := combiner.AddResponse(toHTTPResponseWithFormat(t, tc.result1, 200, nil, marshalingFormat))
			assert.NoError(t, err)

			err = combiner.AddResponse(toHTTPResponseWithFormat(t, tc.result2, 200, nil, marshalingFormat))
			assert.NoError(t, err)

			res, err := combiner.HTTPFinal()
			require.NoError(t, err)

			assert.Equal(t, 200, res.StatusCode)
			assert.Equal(t, tc.expectedShouldQuit, combiner.ShouldQuit())
			assert.Equal(t, 200, combiner.StatusCode())

			fromHTTPResponse(t, res, tc.actualResult)
			tc.sort(tc.expectedResult)
			tc.sort(tc.actualResult)
			require.Equal(t, tc.expectedResult, tc.actualResult)

			require.Equal(t, metrics(tc.expectedResult), metrics(tc.actualResult))
		})
	}
}

func metrics(message proto.Message) *tempopb.MetadataMetrics {
	switch m := message.(type) {
	case *tempopb.SearchTagsResponse:
		return m.Metrics
	case *tempopb.SearchTagsV2Response:
		return m.Metrics
	case *tempopb.SearchTagValuesResponse:
		return m.Metrics
	case *tempopb.SearchTagValuesV2Response:
		return m.Metrics
	}
	return nil
}

func TestTagsGRPCCombinerJSON(t *testing.T) {
	testTagsGRPCCombiner(t, api.MarshallingFormatJSON)
}

func TestTagsGRPCCombinerProtobuf(t *testing.T) {
	testTagsGRPCCombiner(t, api.MarshallingFormatProtobuf)
}

func testTagsGRPCCombiner(t *testing.T, format api.MarshallingFormat) {
	c := NewTypedSearchTags(0, 0, 0, format)
	res1 := &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	res2 := &tempopb.SearchTagsResponse{TagNames: []string{"tag1", "tag2"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	diff1 := &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	diff2 := &tempopb.SearchTagsResponse{TagNames: []string{"tag2"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}}
	expectedFinal := &tempopb.SearchTagsResponse{TagNames: []string{"tag1", "tag2"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}}
	testGRPCCombiner(t, c, res1, res2, diff1, diff2, expectedFinal, func(r *tempopb.SearchTagsResponse) { sort.Strings(r.TagNames) }, format)
}

func TestTagsV2GRPCCombinerJSON(t *testing.T) {
	testTagsV2GRPCCombiner(t, api.MarshallingFormatJSON)
}

func TestTagsV2GRPCCombinerProtobuf(t *testing.T) {
	testTagsV2GRPCCombiner(t, api.MarshallingFormatProtobuf)
}

func testTagsV2GRPCCombiner(t *testing.T, format api.MarshallingFormat) {
	c := NewTypedSearchTagsV2(0, 0, 0, format)
	res1 := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag1"}}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	res2 := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag1", "tag2"}}, {Name: "scope2", Tags: []string{"tag3"}}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	diff1 := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag1"}}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	diff2 := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag2"}}, {Name: "scope2", Tags: []string{"tag3"}}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}}
	expectedFinal := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag1", "tag2"}}, {Name: "scope2", Tags: []string{"tag3"}}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}}
	testGRPCCombiner(t, c, res1, res2, diff1, diff2, expectedFinal, func(r *tempopb.SearchTagsV2Response) {
		for _, scope := range r.Scopes {
			sort.Strings(scope.Tags)
		}
		sort.Slice(r.Scopes, func(i, j int) bool {
			return r.Scopes[i].Name < r.Scopes[j].Name
		})
	}, format)
}

func TestTagValuesGRPCCombinerJSON(t *testing.T) {
	testTagValuesGRPCCombiner(t, api.MarshallingFormatJSON)
}

func TestTagValuesGRPCCombinerProtobuf(t *testing.T) {
	testTagValuesGRPCCombiner(t, api.MarshallingFormatProtobuf)
}

func testTagValuesGRPCCombiner(t *testing.T, format api.MarshallingFormat) {
	c := NewTypedSearchTagValues(0, 0, 0, format)
	res1 := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	res2 := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1", "tag2"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	diff1 := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	diff2 := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag2"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}}
	expectedFinal := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1", "tag2"}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}}
	testGRPCCombiner(t, c, res1, res2, diff1, diff2, expectedFinal, func(r *tempopb.SearchTagValuesResponse) { sort.Strings(r.TagValues) }, format)
}

func TestTagValuesV2GRPCCombinerJSON(t *testing.T) {
	testTagValuesV2GRPCCombiner(t, api.MarshallingFormatJSON)
}

func TestTagValuesV2GRPCCombinerProtobuf(t *testing.T) {
	testTagValuesV2GRPCCombiner(t, api.MarshallingFormatProtobuf)
}

func testTagValuesV2GRPCCombiner(t *testing.T, format api.MarshallingFormat) {
	c := NewTypedSearchTagValuesV2(0, 0, 0, format)
	res1 := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	res2 := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}, {Value: "v2", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	diff1 := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
	diff2 := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v2", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}}
	expectedFinal := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}, {Value: "v2", Type: "string"}}, Metrics: &tempopb.MetadataMetrics{InspectedBytes: 2}}
	testGRPCCombiner(t, c, res1, res2, diff1, diff2, expectedFinal, func(r *tempopb.SearchTagValuesV2Response) {
		sort.Slice(r.TagValues, func(i, j int) bool {
			return r.TagValues[i].Value < r.TagValues[j].Value
		})
	}, format)
}

func testGRPCCombiner[T proto.Message](t *testing.T, combiner GRPCCombiner[T], result1 T, result2 T, diff1 T, diff2 T, expectedFinal T, sort func(T), format api.MarshallingFormat) {
	err := combiner.AddResponse(toHTTPResponseWithFormat(t, result1, 200, nil, format))
	require.NoError(t, err)

	actualDiff1, err := combiner.GRPCDiff()
	require.NoError(t, err)
	sort(actualDiff1)
	require.Equal(t, diff1, actualDiff1)

	err = combiner.AddResponse(toHTTPResponseWithFormat(t, result2, 200, nil, format))
	assert.NoError(t, err)

	actualDiff2, err := combiner.GRPCDiff()
	require.NoError(t, err)
	sort(actualDiff2)
	require.Equal(t, diff2, actualDiff2)

	actualFinal, err := combiner.GRPCFinal()
	require.NoError(t, err)

	sort(actualFinal)
	require.Equal(t, expectedFinal, actualFinal)
}
