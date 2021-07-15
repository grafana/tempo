package search

import (
	"testing"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestPipelineMatchesTags(t *testing.T) {

	testCases := []struct {
		name        string
		request     map[string]string
		searchData  tempofb.SearchDataMap
		shouldMatch bool
	}{
		{
			name:        "match",
			searchData:  tempofb.SearchDataMap{"key": {"value"}},
			request:     map[string]string{"key": "value"},
			shouldMatch: true,
		},
		{
			name:        "noMatch",
			searchData:  tempofb.SearchDataMap{"key1": {"value"}},
			request:     map[string]string{"key2": "value"},
			shouldMatch: false,
		},
		{
			name:        "matchSubstring",
			searchData:  tempofb.SearchDataMap{"key": {"avalue"}},
			request:     map[string]string{"key": "val"},
			shouldMatch: true,
		},
		{
			name:        "matchMulti",
			searchData:  tempofb.SearchDataMap{"key1": {"value1"}, "key2": {"value2"}, "key3": {"value3"}, "key4": {"value4"}},
			request:     map[string]string{"key1": "value1", "key3": "value3"},
			shouldMatch: true,
		},
		{
			name:        "noMatchMulti",
			searchData:  tempofb.SearchDataMap{"key1": {"value1"}, "key2": {"value2"}},
			request:     map[string]string{"key1": "value1", "key3": "value3"},
			shouldMatch: false,
		}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			p := NewSearchPipeline(&tempopb.SearchRequest{Tags: tc.request})
			data := tempofb.SearchDataMutable{
				Tags: tc.searchData,
			}
			sd := tempofb.SearchDataFromBytes(data.ToBytes())
			matches := p.Matches(sd)

			require.Equal(t, tc.shouldMatch, matches)
		})
	}
}

func BenchmarkPipelineMatchesTags(b *testing.B) {

	searchData := tempofb.SearchDataFromBytes((&tempofb.SearchDataMutable{
		Tags: tempofb.SearchDataMap{
			"key1": {"value10", "value11"},
			"key2": {"value20", "value21"},
			"key3": {"value30", "value31"},
			"key4": {"value40", "value41"},
		}}).ToBytes())

	pipeline := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{
			"key2": "valu21",
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.Matches(searchData)
	}
}
