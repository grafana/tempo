package combiner

import (
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagsCombiner(t *testing.T) {
	tests := []struct {
		name               string
		factory            func(int) Combiner
		limit              int
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
			expectedResult: &tempopb.SearchTagsResponse{TagNames: []string{"tag1", "tag2", "tag3"}},
			actualResult:   &tempopb.SearchTagsResponse{},
			sort:           func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagsResponse).TagNames) },
			limit:          100,
		},
		{
			name:           "SearchTagsV2",
			factory:        NewSearchTagsV2,
			result1:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1"}}}},
			result2:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v2", "v1"}}}},
			expectedResult: &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1", "v2"}}}},
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
			limit: 100,
		},
		{
			name:           "SearchTagValues",
			factory:        NewSearchTagValues,
			result1:        &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}},
			result2:        &tempopb.SearchTagValuesResponse{TagValues: []string{"tag2", "tag3"}},
			expectedResult: &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1", "tag2", "tag3"}},
			actualResult:   &tempopb.SearchTagValuesResponse{},
			sort:           func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagValuesResponse).TagValues) },
			limit:          100,
		},
		{
			name:           "SearchTagValuesV2",
			factory:        NewSearchTagValuesV2,
			result1:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}},
			result2:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}},
			expectedResult: &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}, {Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}},
			actualResult:   &tempopb.SearchTagValuesV2Response{},
			sort: func(m proto.Message) {
				sort.Slice(m.(*tempopb.SearchTagValuesV2Response).TagValues, func(i, j int) bool {
					return m.(*tempopb.SearchTagValuesV2Response).TagValues[i].Value < m.(*tempopb.SearchTagValuesV2Response).TagValues[j].Value
				})
			},
			limit: 100,
		},
		// limits
		{
			name:               "SearchTags - limited",
			factory:            NewSearchTags,
			result1:            &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}},
			result2:            &tempopb.SearchTagsResponse{TagNames: []string{"tag2", "tag3"}},
			expectedResult:     &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}},
			actualResult:       &tempopb.SearchTagsResponse{},
			sort:               func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagsResponse).TagNames) },
			expectedShouldQuit: true,
			limit:              5,
		},
		{
			name:           "SearchTagsV2 - limited",
			factory:        NewSearchTagsV2,
			result1:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1"}}}},
			result2:        &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v2", "v1"}}}},
			expectedResult: &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"v1"}}}},
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
			limit:              2,
		},
		{
			name:               "SearchTagValues - limited",
			factory:            NewSearchTagValues,
			result1:            &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}},
			result2:            &tempopb.SearchTagValuesResponse{TagValues: []string{"tag2", "tag3"}},
			expectedResult:     &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}},
			actualResult:       &tempopb.SearchTagValuesResponse{},
			sort:               func(m proto.Message) { sort.Strings(m.(*tempopb.SearchTagValuesResponse).TagValues) },
			expectedShouldQuit: true,
			limit:              5,
		},
		{
			name:           "SearchTagValuesV2 - limited",
			factory:        NewSearchTagValuesV2,
			result1:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}},
			result2:        &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v2", Type: "string"}, {Value: "v3", Type: "string"}}},
			expectedResult: &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}},
			actualResult:   &tempopb.SearchTagValuesV2Response{},
			sort: func(m proto.Message) {
				sort.Slice(m.(*tempopb.SearchTagValuesV2Response).TagValues, func(i, j int) bool {
					return m.(*tempopb.SearchTagValuesV2Response).TagValues[i].Value < m.(*tempopb.SearchTagValuesV2Response).TagValues[j].Value
				})
			},
			expectedShouldQuit: true,
			limit:              10,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			combiner := tc.factory(tc.limit)

			err := combiner.AddResponse(toHTTPResponse(t, tc.result1, 200))
			assert.NoError(t, err)

			err = combiner.AddResponse(toHTTPResponse(t, tc.result2, 200))
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
		})
	}
}

func TestTagsGRPCCombiner(t *testing.T) {
	c := NewTypedSearchTags(0)
	res1 := &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}}
	res2 := &tempopb.SearchTagsResponse{TagNames: []string{"tag1", "tag2"}}
	diff1 := &tempopb.SearchTagsResponse{TagNames: []string{"tag1"}}
	diff2 := &tempopb.SearchTagsResponse{TagNames: []string{"tag2"}}
	expectedFinal := &tempopb.SearchTagsResponse{TagNames: []string{"tag1", "tag2"}}
	testGRPCCombiner(t, c, res1, res2, diff1, diff2, expectedFinal, func(r *tempopb.SearchTagsResponse) { sort.Strings(r.TagNames) })
}

func TestTagsV2GRPCCombiner(t *testing.T) {
	c := NewTypedSearchTagsV2(0)
	res1 := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag1"}}}}
	res2 := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag1", "tag2"}}, {Name: "scope2", Tags: []string{"tag3"}}}}
	diff1 := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag1"}}}}
	diff2 := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag2"}}, {Name: "scope2", Tags: []string{"tag3"}}}}
	expectedFinal := &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{{Name: "scope1", Tags: []string{"tag1", "tag2"}}, {Name: "scope2", Tags: []string{"tag3"}}}}
	testGRPCCombiner(t, c, res1, res2, diff1, diff2, expectedFinal, func(r *tempopb.SearchTagsV2Response) {
		for _, scope := range r.Scopes {
			sort.Strings(scope.Tags)
		}
		sort.Slice(r.Scopes, func(i, j int) bool {
			return r.Scopes[i].Name < r.Scopes[j].Name
		})
	})
}

func TestTagValuesGRPCCombiner(t *testing.T) {
	c := NewTypedSearchTagValues(0)
	res1 := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}}
	res2 := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1", "tag2"}}
	diff1 := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1"}}
	diff2 := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag2"}}
	expectedFinal := &tempopb.SearchTagValuesResponse{TagValues: []string{"tag1", "tag2"}}
	testGRPCCombiner(t, c, res1, res2, diff1, diff2, expectedFinal, func(r *tempopb.SearchTagValuesResponse) { sort.Strings(r.TagValues) })
}

func TestTagValuesV2GRPCCombiner(t *testing.T) {
	c := NewTypedSearchTagValuesV2(0)
	res1 := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}}
	res2 := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}, {Value: "v2", Type: "string"}}}
	diff1 := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}}}
	diff2 := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v2", Type: "string"}}}
	expectedFinal := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Value: "v1", Type: "string"}, {Value: "v2", Type: "string"}}}
	testGRPCCombiner(t, c, res1, res2, diff1, diff2, expectedFinal, func(r *tempopb.SearchTagValuesV2Response) {
		sort.Slice(r.TagValues, func(i, j int) bool {
			return r.TagValues[i].Value < r.TagValues[j].Value
		})
	})
}

func testGRPCCombiner[T proto.Message](t *testing.T, combiner GRPCCombiner[T], result1 T, result2 T, diff1 T, diff2 T, expectedFinal T, sort func(T)) {
	err := combiner.AddResponse(toHTTPResponse(t, result1, 200))
	require.NoError(t, err)

	actualDiff1, err := combiner.GRPCDiff()
	require.NoError(t, err)
	sort(actualDiff1)
	require.Equal(t, diff1, actualDiff1)

	err = combiner.AddResponse(toHTTPResponse(t, result2, 200))
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
