package search

import (
	"strconv"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

func TestPipelineMatchesTags(t *testing.T) {

	testCases := []struct {
		name        string
		request     map[string]string
		searchData  map[string][]string
		shouldMatch bool
	}{
		{
			name:        "match",
			searchData:  map[string][]string{"key": {"value"}},
			request:     map[string]string{"key": "value"},
			shouldMatch: true,
		},
		{
			name:        "noMatch",
			searchData:  map[string][]string{"key1": {"value"}},
			request:     map[string]string{"key2": "value"},
			shouldMatch: false,
		},
		{
			name:        "matchSubstring",
			searchData:  map[string][]string{"key": {"avalue"}},
			request:     map[string]string{"key": "val"},
			shouldMatch: true,
		},
		{
			name:        "matchMulti",
			searchData:  map[string][]string{"key1": {"value1"}, "key2": {"value2"}, "key3": {"value3"}, "key4": {"value4"}},
			request:     map[string]string{"key1": "value1", "key3": "value3"},
			shouldMatch: true,
		},
		{
			name:        "noMatchMulti",
			searchData:  map[string][]string{"key1": {"value1"}, "key2": {"value2"}},
			request:     map[string]string{"key1": "value1", "key3": "value3"},
			shouldMatch: false,
		},
		{
			name:        "rewriteError",
			searchData:  map[string][]string{StatusCodeTag: {strconv.Itoa(int(v1.Status_STATUS_CODE_ERROR))}},
			request:     map[string]string{"error": "t"},
			shouldMatch: true,
		},
		{
			name:        "rewriteStatusCode",
			searchData:  map[string][]string{StatusCodeTag: {strconv.Itoa(int(v1.Status_STATUS_CODE_ERROR))}},
			request:     map[string]string{StatusCodeTag: StatusCodeError},
			shouldMatch: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			p := NewSearchPipeline(&tempopb.SearchRequest{Tags: tc.request})
			data := tempofb.SearchEntryMutable{
				Tags: tempofb.NewSearchDataMapWithData(tc.searchData),
			}
			sd := tempofb.SearchEntryFromBytes(data.ToBytes())
			matches := p.Matches(sd)

			require.Equal(t, tc.shouldMatch, matches)
		})
	}
}

func TestPipelineMatchesTraceDuration(t *testing.T) {

	testCases := []struct {
		name          string
		spanStart     int64
		spanEnd       int64
		minDurationMs uint32
		maxDurationMs uint32
		shouldMatch   bool
	}{
		{
			name:          "no filtering",
			spanStart:     time.Now().UnixNano(),
			spanEnd:       time.Now().UnixNano(),
			minDurationMs: 0,
			maxDurationMs: 0,
			shouldMatch:   true,
		},
		{
			name:          "match both filters",
			minDurationMs: 10,
			maxDurationMs: 100,
			spanStart:     time.Now().UnixNano(),
			spanEnd:       time.Now().Add(50 * time.Millisecond).UnixNano(),
			shouldMatch:   true,
		},
		{
			name:          "no match either filter",
			minDurationMs: 10,
			maxDurationMs: 100,
			spanStart:     time.Now().UnixNano(),
			spanEnd:       time.Now().Add(200 * time.Millisecond).UnixNano(),
			shouldMatch:   false,
		},
		{
			// 4 billion nanoseconds = 4s
			name:          "match more than 32-bits of nanoseconds",
			minDurationMs: 30_000, // 30s
			maxDurationMs: 90_000, // 90s,
			spanStart:     time.Now().UnixNano(),
			spanEnd:       time.Now().Add(time.Minute).UnixNano(),
			shouldMatch:   true,
		},
		{
			// 4 billion nanoseconds = 4s
			name:          "no match more than 32-bits of nanoseconds",
			minDurationMs: 30_000, // 30s
			maxDurationMs: 90_000, // 90s,
			spanStart:     time.Now().UnixNano(),
			spanEnd:       time.Now().Add(15 * time.Second).UnixNano(),
			shouldMatch:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			p := NewSearchPipeline(&tempopb.SearchRequest{MinDurationMs: tc.minDurationMs, MaxDurationMs: tc.maxDurationMs})
			data := tempofb.SearchEntryMutable{
				StartTimeUnixNano: uint64(tc.spanStart),
				EndTimeUnixNano:   uint64(tc.spanEnd),
			}
			sd := tempofb.SearchEntryFromBytes(data.ToBytes())
			matches := p.Matches(sd)

			require.Equal(t, tc.shouldMatch, matches)
		})
	}
}

func TestPipelineMatchesBlock(t *testing.T) {

	// Run all tests against this header
	commonBlock := tempofb.NewSearchBlockHeaderMutable()
	commonBlock.AddTag("tag", "value")
	commonBlock.MinDur = uint64(1 * time.Second)
	commonBlock.MaxDur = uint64(10 * time.Second)
	header := tempofb.GetRootAsSearchBlockHeader(commonBlock.ToBytes(), 0)

	testCases := []struct {
		name        string
		request     tempopb.SearchRequest
		shouldMatch bool
	}{
		{
			name:        "no filters",
			request:     tempopb.SearchRequest{},
			shouldMatch: true,
		},
		{
			name:        "matches all",
			request:     tempopb.SearchRequest{Tags: map[string]string{"tag": "value"}, MinDurationMs: 5000, MaxDurationMs: 6000},
			shouldMatch: true,
		},
		{
			name:        "no matching tag",
			request:     tempopb.SearchRequest{Tags: map[string]string{"nomatch": "value"}},
			shouldMatch: false,
		},
		{
			name:        "no matching min duration",
			request:     tempopb.SearchRequest{MinDurationMs: 20000}, // Above max duration in block
			shouldMatch: false,
		},
		{
			name:        "no matching max duration",
			request:     tempopb.SearchRequest{MaxDurationMs: 500}, // Below smallest duration in block
			shouldMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewSearchPipeline(&tc.request)
			matches := p.MatchesBlock(header)
			require.Equal(t, tc.shouldMatch, matches)
		})
	}
}

func BenchmarkPipelineMatches(b *testing.B) {

	entry := tempofb.SearchEntryFromBytes((&tempofb.SearchEntryMutable{
		StartTimeUnixNano: 0,
		EndTimeUnixNano:   uint64(500 * time.Millisecond / time.Nanosecond), //500ms in nanoseconds
		Tags: tempofb.NewSearchDataMapWithData(map[string][]string{
			"key1": {"value10", "value11"},
			"key2": {"value20", "value21"},
			"key3": {"value30", "value31"},
			"key4": {"value40", "value41"},
		})}).ToBytes())

	testCases := []struct {
		name string
		req  *tempopb.SearchRequest
	}{
		{
			"match_tag",
			&tempopb.SearchRequest{
				Tags: map[string]string{
					"key2": "valu21",
				},
			},
		},
		{
			"nomatch_tag_minDuration",
			&tempopb.SearchRequest{
				MinDurationMs: 501,
				Tags: map[string]string{
					"key5": "nomatch",
				},
			},
		},
		{
			"nomatch_minDuration",
			&tempopb.SearchRequest{
				MinDurationMs: 501,
			},
		},
		{
			"match_minDuration",
			&tempopb.SearchRequest{
				MinDurationMs: 499,
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			pipeline := NewSearchPipeline(tc.req)

			for i := 0; i < b.N; i++ {
				pipeline.Matches(entry)
			}
		})
	}
}
